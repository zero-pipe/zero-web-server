package ptzapp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	domaindevice "zero-web-server/internal/domain/device"
	domainptz "zero-web-server/internal/domain/ptz"
	sipinfra "zero-web-server/internal/infrastructure/sip"
)

const (
	presetCmdSet    = 0x81
	presetCmdCall   = 0x82
	presetCmdDelete = 0x83
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

func (s *Service) presetCmd(deviceID, channelID string, cmdCode, presetID int) error {
	if presetID < 1 || presetID > 255 {
		return fmt.Errorf("预置位编号需在 1-255")
	}
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil {
		return fmt.Errorf("设备不存在")
	}
	return s.sip.SendFrontEndCmd(device, channelID, cmdCode, 0, presetID, 0)
}

func ParsePresetID(raw string) (int, error) {
	id, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("无效的预置位编号: %s", raw)
	}
	return id, nil
}
