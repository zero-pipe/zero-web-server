package ptzapp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"zero-web-server/internal/application/channeldrv"
	domainasset "zero-web-server/internal/domain/asset"
	domainchannel "zero-web-server/internal/domain/channel"
	domaindevice "zero-web-server/internal/domain/device"
	domainptz "zero-web-server/internal/domain/ptz"
	"zero-web-server/internal/gbsip"
	"zero-web-server/internal/port"
)

// GB/T 28181 front-end PTZCmd codes (annex A).
const (
	presetCmdSet    = 0x81
	presetCmdCall   = 0x82
	presetCmdDelete = 0x83

	cruiseAddPoint = 0x84
	cruiseDelPoint = 0x85
	cruiseSetSpeed = 0x86
	cruiseSetTime  = 0x87
	cruiseStart    = 0x88

	scanStart    = 0x89
	scanSetLeft  = 0x8A
	scanSetRight = 0x8B

	auxOn  = 0x8C
	auxOff = 0x8D

	fiStop      = 0x40
	fiFocusFar  = 0x41
	fiFocusNear = 0x42
	fiIrisOpen  = 0x44
	fiIrisClose = 0x48
)

type Service struct {
	channels      domainchannel.Repository
	caps          domainasset.Repository
	drivers       channeldrv.Registry
	devices       domaindevice.Repository
	sip           port.GBPTZNode
	control       port.GBDeviceControl
	presetTimeout time.Duration
	positions     sync.Map // "deviceID/channelGBID" -> gbsip.PTZPosition
}

func NewService(devices domaindevice.Repository, channels domainchannel.Repository,
	caps domainasset.Repository, drivers channeldrv.Registry, sipServer port.GBPTZNode, control port.GBDeviceControl) *Service {
	return &Service{
		channels:      channels,
		caps:          caps,
		drivers:       drivers,
		devices:       devices,
		sip:           sipServer,
		control:       control,
		presetTimeout: 4 * time.Second,
	}
}

// HandlePTZPosition 缓存设备推送的云台精准位置（2022 订阅通知）。
func (s *Service) HandlePTZPosition(deviceID, channelGBID string, pan, tilt, zoom int) error {
	s.positions.Store(deviceID+"/"+channelGBID, gbsip.PTZPosition{Pan: pan, Tilt: tilt, Zoom: zoom})
	return nil
}

// LatestPTZPosition 返回最近一次缓存的云台精准位置。
func (s *Service) LatestPTZPosition(deviceID, channelGBID string) (*gbsip.PTZPosition, bool) {
	v, ok := s.positions.Load(deviceID + "/" + channelGBID)
	if !ok {
		return nil, false
	}
	p := v.(gbsip.PTZPosition)
	return &p, true
}

// QueryPTZPosition 主动查询设备云台精准位置（2022）。
func (s *Service) QueryPTZPosition(deviceID, channelGBID string) (*gbsip.PTZPosition, error) {
	if s.control == nil {
		return nil, fmt.Errorf("SIP 未就绪")
	}
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("设备不存在")
	}
	if !device.OnLine {
		return nil, fmt.Errorf("设备离线")
	}
	sn, ch := s.control.QueryPTZPosition(device, channelGBID)
	select {
	case p, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("设备未应答")
		}
		s.positions.Store(deviceID+"/"+channelGBID, p)
		return &p, nil
	case <-time.After(5 * time.Second):
		s.control.CancelQuery(sn)
		return nil, fmt.Errorf("设备应答超时")
	}
}

// SubscribePTZPosition 订阅设备云台精准位置变化（2022）。
func (s *Service) SubscribePTZPosition(deviceID, channelGBID string, expires int) error {
	if s.control == nil {
		return fmt.Errorf("SIP 未就绪")
	}
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil {
		return fmt.Errorf("设备不存在")
	}
	if !device.OnLine {
		return fmt.Errorf("设备离线")
	}
	if expires <= 0 {
		expires = 3600
	}
	body := gbsip.BuildPTZPositionQuery(channelGBID, "1")
	return s.control.SendSubscribe(device, "PTZPosition", body, expires)
}

func (s *Service) Control(deviceID, channelID, command string, horizonSpeed, verticalSpeed, zoomSpeed int) error {
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil {
		return err
	}
	if err := s.checkPTZCap(deviceID, channelID); err != nil {
		return err
	}
	if err := s.ptzViaDriver(deviceID, channelID, command, horizonSpeed, verticalSpeed, zoomSpeed); err == nil {
		return nil
	} else if !errors.Is(err, errNoDriver) {
		return err
	}
	return s.sip.SendPTZ(device, channelID, command, horizonSpeed, verticalSpeed, zoomSpeed)
}

var errNoDriver = errors.New("no driver")

// ptzViaDriver 走统一驱动（协议差异与速度换算在驱动内），找不到资源/驱动时回落旧通道。
func (s *Service) ptzViaDriver(deviceID, channelID, command string, h, v, z int) error {
	if s.drivers == nil || s.caps == nil || s.channels == nil {
		return errNoDriver
	}
	ch, err := s.channels.GetOne(deviceID, channelID)
	if err != nil || ch == nil {
		return errNoDriver
	}
	res, err := s.caps.GetResourceByInternalCode(ch.InternalCode)
	if err != nil || res == nil {
		return errNoDriver
	}
	binding, err := s.caps.GetBindingByDevice(res.DeviceID)
	if err != nil || binding == nil {
		return errNoDriver
	}
	drv, ok := s.drivers.Get(binding.Protocol)
	if !ok {
		return errNoDriver
	}
	maxSpeed := h
	if v > maxSpeed {
		maxSpeed = v
	}
	cmd := channeldrv.PTZCommand{
		Direction: command,
		Speed:     normalizeSpeed(maxSpeed, 255),
		Zoom:      normalizeSpeed(z, 16),
	}
	if err := drv.PTZ(context.Background(), res, binding, cmd); err != nil {
		_ = s.caps.SetCapabilityAvailable("resource", res.ID, "ptz", false)
		return err
	}
	_ = s.caps.SetCapabilityAvailable("resource", res.ID, "ptz", true)
	return nil
}

func normalizeSpeed(v, max int) float64 {
	if v <= 0 {
		return 0
	}
	f := float64(v) / float64(max)
	if f > 1 {
		f = 1
	}
	return f
}

// checkPTZCap 云台能力校验：无声明放行，声明不支持/当前不可用则拒绝。
func (s *Service) checkPTZCap(deviceID, channelID string) error {
	if s.caps == nil || s.channels == nil {
		return nil
	}
	ch, err := s.channels.GetOne(deviceID, channelID)
	if err != nil || ch == nil {
		return nil
	}
	res, err := s.caps.GetResourceByInternalCode(ch.InternalCode)
	if err != nil || res == nil {
		return nil
	}
	caps, err := s.caps.ListCapabilities("resource", res.ID)
	if err != nil {
		return nil
	}
	for _, c := range caps {
		if c.Operation != "ptz" {
			continue
		}
		if !c.Supported {
			return fmt.Errorf("该通道不支持云台控制")
		}
		if !c.Available {
			return fmt.Errorf("该通道云台能力当前不可用（设备离线或上次调用失败）")
		}
	}
	return nil
}

// QueryPreset sends PresetQuery and waits for the device PresetQuery response.
func (s *Service) QueryPreset(ctx context.Context, deviceID, channelID string) ([]domainptz.Preset, error) {
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("设备不存在")
	}
	sn, ch := s.sip.SendPresetQuery(device, channelID)
	timeout := s.presetTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if remain := time.Until(deadline); remain > 0 && remain < timeout {
			timeout = remain
		}
	}
	select {
	case presets := <-ch:
		if presets == nil {
			return []domainptz.Preset{}, nil
		}
		return presets, nil
	case <-ctx.Done():
		s.sip.CancelPresetQuery(sn)
		return nil, ctx.Err()
	case <-time.After(timeout):
		s.sip.CancelPresetQuery(sn)
		return nil, gbsip.ErrPresetTimeout
	}
}

func (s *Service) AddPreset(deviceID, channelID string, presetID int) error {
	return s.presetCmd(deviceID, channelID, presetCmdSet, presetID)
}

func (s *Service) CallPreset(deviceID, channelID string, presetID int) error {
	return s.presetCmd(deviceID, channelID, presetCmdCall, presetID)
}

func (s *Service) DeletePreset(deviceID, channelID string, presetID int) error {
	return s.presetCmd(deviceID, channelID, presetCmdDelete, presetID)
}

func (s *Service) Focus(deviceID, channelID, command string, speed int) error {
	cmd, err := mapFocusCmd(command)
	if err != nil {
		return err
	}
	return s.frontEnd(deviceID, channelID, cmd, clampByte(speed), 0, 0)
}

func (s *Service) Iris(deviceID, channelID, command string, speed int) error {
	cmd, err := mapIrisCmd(command)
	if err != nil {
		return err
	}
	return s.frontEnd(deviceID, channelID, cmd, 0, clampByte(speed), 0)
}

func (s *Service) Wiper(deviceID, channelID, command string) error {
	return s.Auxiliary(deviceID, channelID, command, 1)
}

func (s *Service) Auxiliary(deviceID, channelID, command string, switchID int) error {
	if switchID < 1 || switchID > 255 {
		return fmt.Errorf("开关编号需在 1-255")
	}
	cmd := auxOn
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "on":
		cmd = auxOn
	case "off":
		cmd = auxOff
	default:
		return fmt.Errorf("无效辅助开关命令: %s", command)
	}
	return s.frontEnd(deviceID, channelID, cmd, switchID, 0, 0)
}

func (s *Service) AddCruisePoint(deviceID, channelID string, cruiseID, presetID int) error {
	if err := checkGroupID(cruiseID); err != nil {
		return err
	}
	if presetID < 1 || presetID > 255 {
		return fmt.Errorf("预置位编号需在 1-255")
	}
	return s.frontEnd(deviceID, channelID, cruiseAddPoint, cruiseID, presetID, 0)
}

func (s *Service) DeleteCruisePoint(deviceID, channelID string, cruiseID, presetID int) error {
	if err := checkGroupID(cruiseID); err != nil {
		return err
	}
	// presetID=0 means clear all points in the cruise group (WVP-compatible).
	if presetID < 0 || presetID > 255 {
		return fmt.Errorf("预置位编号需在 0-255")
	}
	return s.frontEnd(deviceID, channelID, cruiseDelPoint, cruiseID, presetID, 0)
}

func (s *Service) SetCruiseSpeed(deviceID, channelID string, cruiseID, speed int) error {
	if err := checkGroupID(cruiseID); err != nil {
		return err
	}
	lo, hi := encode12Bit(speed)
	return s.frontEnd(deviceID, channelID, cruiseSetSpeed, cruiseID, lo, hi)
}

func (s *Service) SetCruiseTime(deviceID, channelID string, cruiseID, dwellTime int) error {
	if err := checkGroupID(cruiseID); err != nil {
		return err
	}
	lo, hi := encode12Bit(dwellTime)
	return s.frontEnd(deviceID, channelID, cruiseSetTime, cruiseID, lo, hi)
}

func (s *Service) StartCruise(deviceID, channelID string, cruiseID int) error {
	if err := checkGroupID(cruiseID); err != nil {
		return err
	}
	return s.frontEnd(deviceID, channelID, cruiseStart, cruiseID, 0, 0)
}

func (s *Service) StopCruise(deviceID, channelID string, cruiseID int) error {
	_ = cruiseID
	// 国标：停止巡航使用云台停止指令。
	return s.Control(deviceID, channelID, "stop", 0, 0, 0)
}

func (s *Service) StartScan(deviceID, channelID string, scanID int) error {
	if err := checkGroupID(scanID); err != nil {
		return err
	}
	return s.frontEnd(deviceID, channelID, scanStart, scanID, 0, 0)
}

func (s *Service) StopScan(deviceID, channelID string, scanID int) error {
	_ = scanID
	return s.Control(deviceID, channelID, "stop", 0, 0, 0)
}

func (s *Service) SetScanLeft(deviceID, channelID string, scanID int) error {
	if err := checkGroupID(scanID); err != nil {
		return err
	}
	return s.frontEnd(deviceID, channelID, scanSetLeft, scanID, 0, 0)
}

func (s *Service) SetScanRight(deviceID, channelID string, scanID int) error {
	if err := checkGroupID(scanID); err != nil {
		return err
	}
	return s.frontEnd(deviceID, channelID, scanSetRight, scanID, 0, 0)
}

func (s *Service) SetScanSpeed(deviceID, channelID string, scanID, speed int) error {
	if err := checkGroupID(scanID); err != nil {
		return err
	}
	// 与主流平台一致：扫描速度按 12 位编码塞进 8AH 参数区。
	lo, hi := encode12Bit(speed)
	return s.frontEnd(deviceID, channelID, scanSetLeft, scanID, lo, hi)
}

func (s *Service) presetCmd(deviceID, channelID string, cmdCode, presetID int) error {
	if presetID < 1 || presetID > 255 {
		return fmt.Errorf("预置位编号需在 1-255")
	}
	return s.frontEnd(deviceID, channelID, cmdCode, 0, presetID, 0)
}

func (s *Service) frontEnd(deviceID, channelID string, cmdCode, parameter1, parameter2, combineCode2 int) error {
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil {
		return fmt.Errorf("设备不存在")
	}
	return s.sip.SendFrontEndCmd(device, channelID, cmdCode, parameter1, parameter2, combineCode2)
}

func ParsePresetID(raw string) (int, error) {
	id, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("无效的预置位编号: %s", raw)
	}
	return id, nil
}

func mapFocusCmd(command string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "near":
		return fiFocusNear, nil
	case "far":
		return fiFocusFar, nil
	case "stop":
		return fiStop, nil
	default:
		return 0, fmt.Errorf("无效聚焦命令: %s", command)
	}
}

func mapIrisCmd(command string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "in", "open":
		return fiIrisOpen, nil
	case "out", "close":
		return fiIrisClose, nil
	case "stop":
		return fiStop, nil
	default:
		return 0, fmt.Errorf("无效光圈命令: %s", command)
	}
}

func checkGroupID(id int) error {
	if id < 0 || id > 255 {
		return fmt.Errorf("组号需在 0-255")
	}
	return nil
}

func clampByte(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func encode12Bit(v int) (lo, hi int) {
	if v < 1 {
		v = 1
	}
	if v > 4095 {
		v = 4095
	}
	return v & 0xFF, (v >> 8) & 0x0F
}
