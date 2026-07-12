package deviceapp

import (
	"context"
	"log"
	"time"

	domaindevice "zero-web-kit/internal/domain/device"
	redisinfra "zero-web-kit/internal/infrastructure/redis"
)

type StatusMonitor struct {
	devices  domaindevice.Repository
	redis    *redisinfra.Client
	serverID string
	stop     chan struct{}
}

func NewStatusMonitor(devices domaindevice.Repository, redis *redisinfra.Client, serverID string) *StatusMonitor {
	return &StatusMonitor{
		devices: devices, redis: redis, serverID: serverID, stop: make(chan struct{}),
	}
}

func (m *StatusMonitor) Start() {
	if m.redis == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-m.stop:
				return
			case <-ticker.C:
				m.checkExpired()
			}
		}
	}()
}

func (m *StatusMonitor) Stop() { close(m.stop) }

func (m *StatusMonitor) checkExpired() {
	ctx := context.Background()
	ids, err := m.redis.ListExpiredDevices(ctx, m.serverID)
	if err != nil || len(ids) == 0 {
		return
	}
	for _, deviceID := range ids {
		if device, err := m.devices.GetByDeviceID(deviceID); err == nil && device.OnLine {
			device.OnLine = false
			_ = m.devices.UpdateOnline(deviceID, false)
			_ = m.redis.UpdateDevice(ctx, device)
			log.Printf("device %s offline due to keepalive timeout", deviceID)
		}
	}
	_ = m.redis.RemoveExpiredDevices(ctx, m.serverID, ids)
}

func (s *Service) TouchExpiry(device *domaindevice.Device) {
	if s.redis == nil {
		return
	}
	interval := device.HeartBeatInterval
	count := device.HeartBeatCount
	if interval <= 0 {
		interval = 60
	}
	if count <= 0 {
		count = 3
	}
	expiresSec := interval * count
	if device.Expires > 0 && device.Expires < expiresSec {
		expiresSec = device.Expires
	}
	expireAt := time.Now().Add(time.Duration(expiresSec) * time.Second).UnixMilli()
	_ = s.redis.SetDeviceExpiry(context.Background(), device.ServerID, device.DeviceID, expireAt)
}

func (s *Service) RemoveExpiry(deviceID string) {
	if s.redis == nil || deviceID == "" {
		return
	}
	// 直接查库，避免经 GetByDeviceID 与缓存清理形成递归
	device, err := s.devices.GetByDeviceID(deviceID)
	if err != nil || device == nil {
		return
	}
	_ = s.redis.RemoveDeviceExpiry(context.Background(), device.ServerID, deviceID)
}
