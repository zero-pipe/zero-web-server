package sipinfra

import (
	"context"
	"fmt"
	"log"
	"strings"

	domainchannel "zero-web-server/internal/domain/channel"
	domaindevice "zero-web-server/internal/domain/device"
	domainptz "zero-web-server/internal/domain/ptz"
	domainrecord "zero-web-server/internal/domain/record"
	"zero-web-server/internal/gbsip"
	"zero-web-server/internal/infrastructure/config"
	redisinfra "zero-web-server/internal/infrastructure/redis"
	"zero-web-server/internal/port"

	"github.com/emiago/sipgo/sip"
	"github.com/zero-pipe/gb28181-go/manscdp"
	gbserver "github.com/zero-pipe/gb28181-go/server"
	"github.com/zero-pipe/gb28181-go/session"
)

type DeviceService = port.DeviceLifecycle

type AlarmHandler interface {
	HandleNotify(deviceID, channelGBID string, alarm *AlarmNotify) error
}

type PositionHandler interface {
	HandleNotify(deviceID, channelGBID string, pos *MobilePositionNotify) error
}

// PTZPositionHandler receives GB28181 PTZPosition (2022) notifications.
type PTZPositionHandler interface {
	HandlePTZPosition(deviceID, channelGBID string, pan, tilt, zoom int) error
}

// MediaHandler receives GB28181 MediaStatus (File to End) notifications.
type MediaHandler interface {
	HandleMediaStatus(deviceID, channelGBID, notifyType string) error
}

// Server wraps gb28181-go platform SIP server with ZWS domain adapters.
type Server struct {
	lib    *gbserver.Server
	bridge *bridge
	cfg    config.SIPConfig
	sipCfg gbserver.Config
}

func NewServer(cfg config.SIPConfig, serverID, password string, deviceSvc DeviceService, redis *redisinfra.Client) (*Server, error) {
	b := &bridge{
		deviceSvc: deviceSvc,
		redis:     redis,
		password:  password,
		serverID:  serverID,
		redirect:  cfg.RedirectContact,
	}
	libCfg := toLibConfig(cfg, serverID, false)
	libCfg.Password = password
	lib, err := gbserver.New(libCfg, gbserver.Handlers{
		Auth:           b,
		Register:       b,
		Message:        b,
		Telemetry:      telemetryBridge{b: b},
		CascadeInbound: b,
		Redirect:       b,
		Transport:      b,
	})
	if err != nil {
		return nil, err
	}
	return &Server{lib: lib, bridge: b, cfg: cfg, sipCfg: libCfg}, nil
}

func (s *Server) SetRequirePreRegister(v bool) {
	cfg := s.lib.Config()
	cfg.RequirePreRegister = v
	s.lib.ApplyConfig(cfg)
	s.sipCfg = cfg
}

func (s *Server) SetAlarmHandler(h AlarmHandler)             { s.bridge.alarm = h }
func (s *Server) SetPositionHandler(h PositionHandler)       { s.bridge.position = h }
func (s *Server) SetMediaHandler(h MediaHandler)             { s.bridge.media = h }
func (s *Server) SetPTZPositionHandler(h PTZPositionHandler) { s.bridge.ptzPosition = h }
func (s *Server) SetSubordinateHandler(h SubordinateHandler) { s.bridge.subordinate = h }
func (s *Server) SetCascadeInbound(h gbserver.CascadeInboundHandler) {
	s.bridge.cascade = h
}

func (s *Server) SetLocalIP(ip string) { s.lib.SetLocalIP(ip) }

func (s *Server) ApplyConfig(cfg config.SIPConfig) {
	s.cfg = cfg
	libCfg := toLibConfig(cfg, s.bridge.serverID, s.lib.Config().RequirePreRegister)
	s.bridge.password = cfg.Password
	s.bridge.redirect = cfg.RedirectContact
	s.lib.ApplyConfig(libCfg)
	s.sipCfg = libCfg
}

func (s *Server) Config() config.SIPConfig { return s.cfg }
func (s *Server) LocalIP() string          { return s.lib.LocalIP() }
func (s *Server) Domain() string           { return s.lib.Domain() }

func GuessLocalIP() string { return gbserver.GuessLocalIP() }

func (s *Server) Start(ctx context.Context) error { return s.lib.Start(ctx) }

func (s *Server) RecordManager() *RecordManager {
	return &RecordManager{inner: s.lib.Records()}
}

func (s *Server) PresetManager() *PresetManager {
	return &PresetManager{inner: s.lib.Presets()}
}

func (s *Server) InviteManager() *InviteManager {
	return &InviteManager{inner: s.lib.Invites()}
}

func (s *Server) NextInfoCSeq() int { return s.lib.NextInfoCSeq() }

func (s *Server) SendCatalogQuery(device *domaindevice.Device) error {
	return s.lib.SendCatalogQuery(toPeer(device))
}

func (s *Server) SendDeviceInfoQuery(device *domaindevice.Device) error {
	return s.lib.SendDeviceInfoQuery(toPeer(device))
}

func (s *Server) SendDeviceControl(device *domaindevice.Device, channelID, xmlBody string) error {
	_ = channelID
	return s.lib.SendDeviceControl(toPeer(device), xmlBody)
}

func (s *Server) SendPTZ(device *domaindevice.Device, channelID, direction string, h, v, z int) error {
	return s.lib.SendPTZ(toPeer(device), channelID, direction, h, v, z)
}

func (s *Server) SendFrontEndCmd(device *domaindevice.Device, channelID string, cmdCode, parameter1, parameter2, combineCode2 int) error {
	return s.lib.SendFrontEndCmd(toPeer(device), channelID, cmdCode, parameter1, parameter2, combineCode2)
}

// SendGuardCmd 发送布防/撤防（GuardCmd=SetGuard|ResetGuard）。channelID 空则用设备国标编号。
func (s *Server) SendGuardCmd(device *domaindevice.Device, channelID, guardCmd string) error {
	if device == nil {
		return fmt.Errorf("设备为空")
	}
	cmd := strings.TrimSpace(guardCmd)
	if cmd != "SetGuard" && cmd != "ResetGuard" {
		return fmt.Errorf("无效布防命令: %s", guardCmd)
	}
	target := strings.TrimSpace(channelID)
	if target == "" {
		target = device.DeviceID
	}
	body := manscdp.BuildGuardCmd(target, s.lib.NextSN(), cmd)
	return s.lib.SendDeviceControl(toPeer(device), body)
}

// SendRecordCmd 发送设备录像控制（RecordCmd=Record|StopRecord）。channelID 空则用设备国标编号。
func (s *Server) SendRecordCmd(device *domaindevice.Device, channelID, recordCmd string) error {
	if device == nil {
		return fmt.Errorf("设备为空")
	}
	cmd := strings.TrimSpace(recordCmd)
	if cmd != "Record" && cmd != "StopRecord" {
		return fmt.Errorf("无效录像命令: %s", recordCmd)
	}
	target := strings.TrimSpace(channelID)
	if target == "" {
		target = device.DeviceID
	}
	body := manscdp.BuildRecordCmd(target, s.lib.NextSN(), cmd)
	return s.lib.SendDeviceControl(toPeer(device), body)
}

// SendDragZoom 发送拉框放大/缩小。
func (s *Server) SendDragZoom(device *domaindevice.Device, channelID string, zoomIn bool, length, width, midX, midY, lengthX, lengthY int) error {
	if device == nil {
		return fmt.Errorf("设备为空")
	}
	target := strings.TrimSpace(channelID)
	if target == "" {
		target = device.DeviceID
	}
	body := manscdp.BuildDragZoom(target, s.lib.NextSN(), zoomIn, length, width, midX, midY, lengthX, lengthY)
	return s.lib.SendDeviceControl(toPeer(device), body)
}

// SendHomePosition 发送看守位控制。
func (s *Server) SendHomePosition(device *domaindevice.Device, channelID string, enabled, resetTime, presetIndex int) error {
	if device == nil {
		return fmt.Errorf("设备为空")
	}
	target := strings.TrimSpace(channelID)
	if target == "" {
		target = device.DeviceID
	}
	body := manscdp.BuildHomePosition(target, s.lib.NextSN(), enabled, resetTime, presetIndex)
	return s.lib.SendDeviceControl(toPeer(device), body)
}

// SendBasicParamConfig 下发基础参数配置。
func (s *Server) SendBasicParamConfig(device *domaindevice.Device, name string, expiration, heartBeatInterval, heartBeatCount, positionCapability int) error {
	if device == nil {
		return fmt.Errorf("设备为空")
	}
	body := manscdp.BuildBasicParamConfig(device.DeviceID, s.lib.NextSN(), name, expiration, heartBeatInterval, heartBeatCount, positionCapability)
	return s.lib.SendDeviceControl(toPeer(device), body)
}

// SendConfigDownloadQuery 查询设备配置（BasicParam 等），当前不阻塞等待响应。
// QueryConfigDownload 查询基础参数等配置，返回 SN 与解析结果通道。
func (s *Server) QueryConfigDownload(device *domaindevice.Device, configType string) (string, <-chan gbsip.BasicParam) {
	sn, raw := s.lib.QueryConfigDownload(toPeer(device), configType)
	return sn, fanout(raw, func(m *manscdp.Message) (gbsip.BasicParam, bool) {
		if m.BasicParam == nil {
			return gbsip.BasicParam{}, false
		}
		return *m.BasicParam, true
	})
}

// QueryHomePosition 查询看守位（2022）。
func (s *Server) QueryHomePosition(device *domaindevice.Device, channelID string) (string, <-chan gbsip.HomePosition) {
	sn, raw := s.lib.QueryHomePosition(toPeer(device), channelID)
	return sn, fanout(raw, func(m *manscdp.Message) (gbsip.HomePosition, bool) {
		if m.HomePosition == nil {
			return gbsip.HomePosition{}, false
		}
		return *m.HomePosition, true
	})
}

// QueryCruiseTrackList 查询巡航轨迹列表（2022）。
func (s *Server) QueryCruiseTrackList(device *domaindevice.Device, channelID string) (string, <-chan gbsip.CruiseTrackList) {
	sn, raw := s.lib.QueryCruiseTrackList(toPeer(device), channelID)
	return sn, fanout(raw, func(m *manscdp.Message) (gbsip.CruiseTrackList, bool) {
		if m.CruiseTrackList == nil {
			return gbsip.CruiseTrackList{}, false
		}
		return *m.CruiseTrackList, true
	})
}

// QueryCruiseTrack 查询单条巡航轨迹（2022）。
func (s *Server) QueryCruiseTrack(device *domaindevice.Device, channelID string, number int) (string, <-chan gbsip.CruiseTrack) {
	sn, raw := s.lib.QueryCruiseTrack(toPeer(device), channelID, number)
	return sn, fanout(raw, func(m *manscdp.Message) (gbsip.CruiseTrack, bool) {
		if m.CruiseTrack == nil {
			return gbsip.CruiseTrack{}, false
		}
		return *m.CruiseTrack, true
	})
}

// QueryPTZPosition 查询云台精准位置（2022）。
func (s *Server) QueryPTZPosition(device *domaindevice.Device, channelID string) (string, <-chan gbsip.PTZPosition) {
	sn, raw := s.lib.QueryPTZPosition(toPeer(device), channelID)
	return sn, fanout(raw, func(m *manscdp.Message) (gbsip.PTZPosition, bool) {
		if m.PTZPosition == nil {
			return gbsip.PTZPosition{}, false
		}
		return *m.PTZPosition, true
	})
}

// QuerySDCardStatus 查询存储卡状态（2022）。
func (s *Server) QuerySDCardStatus(device *domaindevice.Device, channelID string) (string, <-chan gbsip.SDCardStatus) {
	sn, raw := s.lib.QuerySDCardStatus(toPeer(device), channelID)
	return sn, fanout(raw, func(m *manscdp.Message) (gbsip.SDCardStatus, bool) {
		if m.SDCardStatus == nil {
			return gbsip.SDCardStatus{}, false
		}
		return *m.SDCardStatus, true
	})
}

// CancelQuery 取消未应答的查询。
func (s *Server) CancelQuery(sn string) { s.lib.CancelQuery(sn) }

// fanout 把通用查询应答通道转换为具体类型通道，附带超时/取消处理。
func fanout[T any](raw <-chan *manscdp.Message, pick func(*manscdp.Message) (T, bool)) <-chan T {
	out := make(chan T, 1)
	go func() {
		defer close(out)
		msg, ok := <-raw
		if !ok || msg == nil {
			return
		}
		if v, ok := pick(msg); ok {
			out <- v
		}
	}()
	return out
}

// SendPTZPrecise 发送 PTZ 精准控制（2022）。
func (s *Server) SendPTZPrecise(device *domaindevice.Device, channelID string, pan, tilt, zoom float64) error {
	if device == nil {
		return fmt.Errorf("设备为空")
	}
	target := strings.TrimSpace(channelID)
	if target == "" {
		target = device.DeviceID
	}
	body := manscdp.BuildPTZPreciseCtrl(target, s.lib.NextSN(), pan, tilt, zoom)
	return s.lib.SendDeviceControl(toPeer(device), body)
}

// SendFormatSDCard 发送存储卡格式化（2022，无应答）。
func (s *Server) SendFormatSDCard(device *domaindevice.Device, channelID string) error {
	if device == nil {
		return fmt.Errorf("设备为空")
	}
	target := strings.TrimSpace(channelID)
	if target == "" {
		target = device.DeviceID
	}
	body := manscdp.BuildFormatSDCard(target, s.lib.NextSN())
	return s.lib.SendDeviceControl(toPeer(device), body)
}

// SendTargetTrack 发送目标跟踪（2022，无应答）。
func (s *Server) SendTargetTrack(device *domaindevice.Device, channelID, trackType string) error {
	if device == nil {
		return fmt.Errorf("设备为空")
	}
	target := strings.TrimSpace(channelID)
	if target == "" {
		target = device.DeviceID
	}
	body := manscdp.BuildTargetTrack(target, s.lib.NextSN(), trackType)
	return s.lib.SendDeviceControl(toPeer(device), body)
}

// SendAlarmCmd 发送报警复位（AlarmCmd=ResetAlarm）。
func (s *Server) SendAlarmCmd(device *domaindevice.Device, channelID, alarmCmd string) error {
	if device == nil {
		return fmt.Errorf("设备为空")
	}
	target := strings.TrimSpace(channelID)
	if target == "" {
		target = device.DeviceID
	}
	sn, ch := s.lib.SendAlarmCmd(toPeer(device), target, alarmCmd)
	go func() {
		if res, ok := <-ch; ok {
			log.Printf("[gb28181] AlarmCmd result: sn=%s device=%s result=%s", sn, target, res.Result)
		}
	}()
	return nil
}

// SendTeleBoot 发送远程重启。
func (s *Server) SendTeleBoot(device *domaindevice.Device, channelID string) error {
	if device == nil {
		return fmt.Errorf("设备为空")
	}
	target := strings.TrimSpace(channelID)
	if target == "" {
		target = device.DeviceID
	}
	sn, ch := s.lib.SendTeleBoot(toPeer(device), target)
	go func() {
		if res, ok := <-ch; ok {
			log.Printf("[gb28181] TeleBoot result: sn=%s device=%s result=%s", sn, target, res.Result)
		}
	}()
	return nil
}

// SendIFrameCmd 发送强制关键帧。
func (s *Server) SendIFrameCmd(device *domaindevice.Device, channelID string) error {
	if device == nil {
		return fmt.Errorf("设备为空")
	}
	target := strings.TrimSpace(channelID)
	if target == "" {
		target = device.DeviceID
	}
	sn, ch := s.lib.SendIFrameCmd(toPeer(device), target)
	go func() {
		if res, ok := <-ch; ok {
			log.Printf("[gb28181] IFrame result: sn=%s device=%s result=%s", sn, target, res.Result)
		}
	}()
	return nil
}

// SendDeviceStatusQuery 查询设备状态，返回 SN 与转换后的结果通道。
func (s *Server) SendDeviceStatusQuery(device *domaindevice.Device) (string, <-chan port.GBDeviceStatus) {
	sn, raw := s.lib.SendDeviceStatusQuery(toPeer(device))
	out := make(chan port.GBDeviceStatus, 1)
	go func() {
		defer close(out)
		st, ok := <-raw
		if !ok || st == nil {
			return
		}
		out <- port.GBDeviceStatus{
			Online:     st.Online,
			Status:     st.Status,
			Encode:     st.Encode,
			Record:     st.Record,
			DeviceTime: st.DeviceTime,
		}
	}()
	return sn, out
}

// CancelDeviceStatusQuery 取消未应答的状态查询。
func (s *Server) CancelDeviceStatusQuery(sn string) { s.lib.CancelDeviceStatusQuery(sn) }

func (s *Server) SendAudioBroadcast(device *domaindevice.Device, channelGBID string) error {
	return s.lib.SendAudioBroadcast(toPeer(device), channelGBID)
}

func (s *Server) SendSubscribe(device *domaindevice.Device, eventType, body string, expiresSec int) error {
	return s.lib.SendSubscribe(toPeer(device), eventType, body, expiresSec)
}

func (s *Server) SendSubscribeCancel(device *domaindevice.Device, eventType, body string) error {
	return s.lib.SendSubscribeCancel(toPeer(device), eventType, body)
}

func (s *Server) SendRecordInfoQuery(device *domaindevice.Device, channelID, startTime, endTime string) (string, <-chan *domainrecord.RecordInfo) {
	sn, ch := s.lib.SendRecordInfoQuery(toPeer(device), channelID, startTime, endTime)
	out := make(chan *domainrecord.RecordInfo, 1)
	go func() {
		defer close(out)
		info, ok := <-ch
		if !ok || info == nil {
			return
		}
		out <- toDomainRecordInfo(info)
	}()
	return sn, out
}

func (s *Server) CancelRecordQuery(sn string) { s.lib.CancelRecordQuery(sn) }

func (s *Server) SendPresetQuery(device *domaindevice.Device, channelID string) (string, <-chan []domainptz.Preset) {
	sn, ch := s.lib.SendPresetQuery(toPeer(device), channelID)
	out := make(chan []domainptz.Preset, 1)
	go func() {
		defer close(out)
		items, ok := <-ch
		if !ok {
			return
		}
		out <- toDomainPresets(items)
	}()
	return sn, out
}

func (s *Server) CancelPresetQuery(sn string) { s.lib.CancelPresetQuery(sn) }

func (s *Server) SendInvitePlay(device *domaindevice.Device, channel *domainchannel.Channel, sdpBody, ssrc, stream, streamMode string, tcpConnect func(host string, port int) error, onOK func(*sip.Response)) error {
	return s.lib.SendInvitePlay(toPeer(device), gbserver.InviteTarget{ChannelID: channel.GBDeviceID},
		sdpBody, ssrc, stream, streamMode, tcpConnect, onOK)
}

func (s *Server) SendInviteSession(device *domaindevice.Device, channel *domainchannel.Channel, sdpBody, ssrc, stream string, sessionType SessionType, startTime, endTime string, downloadSpeed int) error {
	return s.lib.SendInviteSession(toPeer(device), gbserver.InviteTarget{ChannelID: channel.GBDeviceID},
		sdpBody, ssrc, stream, session.SessionType(sessionType), startTime, endTime, downloadSpeed)
}

func (s *Server) SendPlaybackControl(stream, content string) error {
	return s.lib.SendPlaybackControl(stream, content)
}

func (s *Server) CloseInviteSession(stream string) error {
	return s.lib.CloseInviteSession(stream)
}

func toDomainRecordInfo(info *session.RecordInfo) *domainrecord.RecordInfo {
	out := &domainrecord.RecordInfo{
		DeviceID: info.DeviceID, ChannelID: info.ChannelID, SN: info.SN,
		SumNum: info.SumNum, Count: info.Count,
	}
	for _, it := range info.RecordList {
		out.RecordList = append(out.RecordList, domainrecord.RecordItem{
			DeviceID: it.DeviceID, Name: it.Name, FilePath: it.FilePath,
			FileSize: it.FileSize, StartTime: it.StartTime, EndTime: it.EndTime,
			Secrecy: it.Secrecy, Type: it.Type, RecorderID: it.RecorderID,
		})
	}
	return out
}

func toDomainPresets(items []manscdp.Preset) []domainptz.Preset {
	out := make([]domainptz.Preset, len(items))
	for i, p := range items {
		out[i] = domainptz.Preset{PresetID: p.PresetID, PresetName: p.PresetName}
	}
	return out
}
