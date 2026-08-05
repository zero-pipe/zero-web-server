package sipinfra

import (
	"context"
	"fmt"
	"strings"

	domainchannel "zero-web-server/internal/domain/channel"
	domaindevice "zero-web-server/internal/domain/device"
	domainptz "zero-web-server/internal/domain/ptz"
	domainrecord "zero-web-server/internal/domain/record"
	"zero-web-server/internal/infrastructure/config"
	redisinfra "zero-web-server/internal/infrastructure/redis"

	"github.com/emiago/sipgo/sip"
	"github.com/zero-pipe/gb28181-go/manscdp"
	gbserver "github.com/zero-pipe/gb28181-go/server"
	"github.com/zero-pipe/gb28181-go/session"
)

type DeviceService interface {
	GetByDeviceID(deviceID string) (*domaindevice.Device, error)
	Online(device *domaindevice.Device) error
	Offline(device *domaindevice.Device) error
	SaveRegister(device *domaindevice.Device) (*domaindevice.Device, error)
	HandleKeepalive(deviceID, ip string, port int) error
	HandleCatalog(deviceID string, items []CatalogItem) error
	HandleDeviceInfo(deviceID, name, manufacturer, model, firmware string) error
	OnDeviceOnline(device *domaindevice.Device)
}

type AlarmHandler interface {
	HandleNotify(deviceID, channelGBID string, alarm *AlarmNotify) error
}

type PositionHandler interface {
	HandleNotify(deviceID, channelGBID string, pos *MobilePositionNotify) error
}

// Server wraps gb28181-go platform SIP server with ZWS domain adapters.
type Server struct {
	lib     *gbserver.Server
	bridge  *bridge
	cfg     config.SIPConfig
	sipCfg  gbserver.Config
}

func NewServer(cfg config.SIPConfig, serverID, password string, deviceSvc DeviceService, redis *redisinfra.Client) (*Server, error) {
	b := &bridge{
		deviceSvc: deviceSvc,
		redis:     redis,
		password:  password,
		serverID:  serverID,
	}
	libCfg := toLibConfig(cfg, serverID, false)
	libCfg.Password = password
	lib, err := gbserver.New(libCfg, gbserver.Handlers{
		Auth:           b,
		Register:       b,
		Message:        b,
		Telemetry:      telemetryBridge{b: b},
		CascadeInbound: b,
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
func (s *Server) SetSubordinateHandler(h SubordinateHandler) { s.bridge.subordinate = h }
func (s *Server) SetCascadeInbound(h gbserver.CascadeInboundHandler) {
	s.bridge.cascade = h
}

func (s *Server) SetLocalIP(ip string) { s.lib.SetLocalIP(ip) }

func (s *Server) ApplyConfig(cfg config.SIPConfig) {
	s.cfg = cfg
	libCfg := toLibConfig(cfg, s.bridge.serverID, s.lib.Config().RequirePreRegister)
	s.bridge.password = cfg.Password
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
func (s *Server) SendConfigDownloadQuery(device *domaindevice.Device, configType string) error {
	if device == nil {
		return fmt.Errorf("设备为空")
	}
	return s.lib.SendConfigDownloadQuery(toPeer(device), configType)
}

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