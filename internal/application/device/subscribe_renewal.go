package deviceapp

import (
	"sync"
	"time"

	domaindevice "zero-web-kit/internal/domain/device"
)

type subscribeRenewal struct {
	tasks sync.Map // deviceID -> *time.Timer
}

var renewal = &subscribeRenewal{}

func (s *Service) AutoSubscribeOnOnline(device *domaindevice.Device) {
	if device == nil || !device.OnLine {
		return
	}
	if device.SubscribeCycleForCatalog > 0 {
		_ = s.SubscribeCatalog(device.ID, device.SubscribeCycleForCatalog)
		s.scheduleRenewal(device, "catalog", device.SubscribeCycleForCatalog, func() {
			_ = s.SubscribeCatalog(device.ID, device.SubscribeCycleForCatalog)
		})
	}
	if device.SubscribeCycleForAlarm > 0 {
		_ = s.SubscribeAlarm(device.ID, device.SubscribeCycleForAlarm)
		s.scheduleRenewal(device, "alarm", device.SubscribeCycleForAlarm, func() {
			_ = s.SubscribeAlarm(device.ID, device.SubscribeCycleForAlarm)
		})
	}
	if device.SubscribeCycleForMobilePosition > 0 {
		interval := device.MobilePositionSubmissionInterval
		if interval <= 0 {
			interval = 5
		}
		_ = s.SubscribeMobilePosition(device.ID, device.SubscribeCycleForMobilePosition, interval)
		s.scheduleRenewal(device, "mobile", device.SubscribeCycleForMobilePosition, func() {
			_ = s.SubscribeMobilePosition(device.ID, device.SubscribeCycleForMobilePosition, interval)
		})
	}
}

func (s *Service) scheduleRenewal(device *domaindevice.Device, kind string, cycleSec int, fn func()) {
	key := device.DeviceID + ":" + kind
	if v, ok := renewal.tasks.Load(key); ok {
		if t, ok := v.(*time.Timer); ok {
			t.Stop()
		}
	}
	delay := time.Duration(cycleSec)*time.Second - 5*time.Second
	if delay < 30*time.Second {
		delay = 30 * time.Second
	}
	timer := time.AfterFunc(delay, func() {
		if d, err := s.GetByDeviceID(device.DeviceID); err == nil && d.OnLine {
			fn()
			s.scheduleRenewal(d, kind, cycleSec, fn)
		}
	})
	renewal.tasks.Store(key, timer)
}
