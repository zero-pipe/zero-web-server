package ptzapp

import (
	domaindevice "zero-web-kit/internal/domain/device"
	sipinfra "zero-web-kit/internal/infrastructure/sip"
)

type Service struct {
	devices domaindevice.Repository
	sip     *sipinfra.Server
}

func NewService(devices domaindevice.Repository, sipServer *sipinfra.Server) *Service {
	return &Service{devices: devices, sip: sipServer}
}

func (s *Service) Control(deviceID, channelID, command string, horizonSpeed, verticalSpeed, zoomSpeed int) error {
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil {
		return err
	}
	return s.sip.SendPTZ(device, channelID, command, horizonSpeed, verticalSpeed, zoomSpeed)
}
