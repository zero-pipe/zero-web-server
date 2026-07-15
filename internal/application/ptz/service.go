package ptzapp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	domaindevice "zero-web-server/internal/domain/device"
	domainptz "zero-web-server/internal/domain/ptz"
	sipinfra "zero-web-server/internal/infrastructure/sip"
)

// GB/T 28181 front-end PTZCmd codes (annex A).
const (
	presetCmdSet    = 0x81
	presetCmdCall   = 0x82
	presetCmdDelete = 0x83

	cruiseAddPoint  = 0x84
	cruiseDelPoint  = 0x85
	cruiseSetSpeed  = 0x86
	cruiseSetTime   = 0x87
	cruiseStart     = 0x88

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
	devices       domaindevice.Repository
	sip           *sipinfra.Server
	presetTimeout time.Duration
}

func NewService(devices domaindevice.Repository, sipServer *sipinfra.Server) *Service {
	return &Service{
		devices:       devices,
		sip:           sipServer,
		presetTimeout: 4 * time.Second,
	}
}

func (s *Service) Control(deviceID, channelID, command string, horizonSpeed, verticalSpeed, zoomSpeed int) error {
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil {
		return err
	}
	return s.sip.SendPTZ(device, channelID, command, horizonSpeed, verticalSpeed, zoomSpeed)
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
		return nil, sipinfra.ErrPresetTimeout
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
