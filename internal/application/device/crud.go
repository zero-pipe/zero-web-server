package deviceapp

import (
	"context"
	"errors"
	domaindevice "zero-web-server/internal/domain/device"
)

func (s *Service) List(page, count int, query string, online *bool) ([]*domaindevice.Device, int64, error) {
	return s.devices.List(page, count, query, online)
}

func (s *Service) Delete(deviceID string) error {
	serverID := ""
	if d, err := s.devices.GetByDeviceID(deviceID); err == nil && d != nil {
		serverID = d.ServerID
	}
	_ = s.channels.DeleteByDevice(deviceID)
	if err := s.devices.DeleteByDeviceID(deviceID); err != nil {
		return err
	}
	if s.redis != nil {
		_ = s.redis.RemoveDevice(context.Background(), deviceID)
		if serverID != "" {
			_ = s.redis.RemoveDeviceExpiry(context.Background(), serverID, deviceID)
		}
	}
	return nil
}

func (s *Service) ChangeChannelAudio(channelID int, audio bool) error {
	if channelID <= 0 {
		return errors.New("通道ID无效")
	}
	if _, err := s.channels.GetByID(channelID); err != nil {
		return errors.New("通道不存在")
	}
	return s.channels.ChangeAudio(channelID, audio)
}

func (s *Service) AddDevice(device *domaindevice.Device) error {
	device.CreateTime = nowStr()
	device.UpdateTime = device.CreateTime
	return s.devices.Create(device)
}

func (s *Service) UpdateDevice(device *domaindevice.Device) error {
	device.UpdateTime = nowStr()
	return s.devices.Update(device)
}
