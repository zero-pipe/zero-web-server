package deviceapp

import (
	"context"
	"errors"
	"strconv"
	"strings"

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
	if audio {
		if err := s.checkChannelCapByID(channelID, "audio"); err != nil {
			return err
		}
	}
	return s.channels.ChangeAudio(channelID, audio)
}

// ChangeChannelStreamIdentification 设置通道码流类型。
// channelID>0 时按通道；否则按设备批量（deviceKey 可为 GB 设备号或设备库主键）。
func (s *Service) ChangeChannelStreamIdentification(channelID int, deviceKey, value string) error {
	value = strings.TrimSpace(value)
	if channelID > 0 {
		if _, err := s.channels.GetByID(channelID); err != nil {
			return errors.New("通道不存在")
		}
		return s.channels.UpdateStreamIdentification(channelID, "", value)
	}
	deviceKey = strings.TrimSpace(deviceKey)
	if deviceKey == "" {
		return errors.New("设备ID无效")
	}
	gbID := ""
	if dev, err := s.devices.GetByDeviceID(deviceKey); err == nil && dev != nil {
		gbID = dev.DeviceID
	} else if n, e := strconv.Atoi(deviceKey); e == nil && n > 0 {
		if dev, e2 := s.devices.GetByID(n); e2 == nil && dev != nil {
			gbID = dev.DeviceID
		}
	}
	if gbID == "" {
		return errors.New("设备不存在")
	}
	return s.channels.UpdateStreamIdentification(0, gbID, value)
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
