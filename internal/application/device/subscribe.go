package deviceapp

import (
	"errors"
	"fmt"
	"strconv"

	sipinfra "zero-web-server/internal/infrastructure/sip"
)

var ErrDeviceOffline = errors.New("设备已离线")

func (s *Service) SubscribeCatalog(id, cycle int) error {
	device, err := s.devices.GetByID(id)
	if err != nil {
		return ErrDeviceNotFound
	}
	if cycle > 0 && !device.OnLine {
		return ErrDeviceOffline
	}
	if device.SubscribeCycleForCatalog == cycle {
		return nil
	}
	if s.sip == nil {
		return errors.New("SIP服务未启动")
	}
	sn := strconv.Itoa(id)
	if cycle > 0 {
		body := sipinfra.BuildSubscribeCatalog(device.DeviceID, sn)
		if err := s.sip.SendSubscribe(device, "Catalog", body, cycle); err != nil {
			return fmt.Errorf("目录订阅失败: %w", err)
		}
		device.SubscribeCycleForCatalog = cycle
	} else {
		body := sipinfra.BuildSubscribeCatalog(device.DeviceID, sn)
		_ = s.sip.SendSubscribeCancel(device, "Catalog", body)
		device.SubscribeCycleForCatalog = 0
	}
	return s.UpdateDevice(device)
}

func (s *Service) SubscribeAlarm(id, cycle int) error {
	device, err := s.devices.GetByID(id)
	if err != nil {
		return ErrDeviceNotFound
	}
	if cycle > 0 && !device.OnLine {
		return ErrDeviceOffline
	}
	if device.SubscribeCycleForAlarm == cycle {
		return nil
	}
	if s.sip == nil {
		return errors.New("SIP服务未启动")
	}
	sn := strconv.Itoa(id)
	if cycle > 0 {
		body := sipinfra.BuildSubscribeAlarm(device.DeviceID, sn)
		if err := s.sip.SendSubscribe(device, "Alarm", body, cycle); err != nil {
			return fmt.Errorf("报警订阅失败: %w", err)
		}
		device.SubscribeCycleForAlarm = cycle
	} else {
		body := sipinfra.BuildSubscribeAlarm(device.DeviceID, sn)
		_ = s.sip.SendSubscribeCancel(device, "Alarm", body)
		device.SubscribeCycleForAlarm = 0
	}
	return s.UpdateDevice(device)
}

func (s *Service) SubscribeMobilePosition(id, cycle, interval int) error {
	device, err := s.devices.GetByID(id)
	if err != nil {
		return ErrDeviceNotFound
	}
	if cycle > 0 && !device.OnLine {
		device.SubscribeCycleForMobilePosition = cycle
		device.MobilePositionSubmissionInterval = interval
		_ = s.UpdateDevice(device)
		return ErrDeviceOffline
	}
	if device.SubscribeCycleForMobilePosition == cycle {
		return nil
	}
	if s.sip == nil {
		return errors.New("SIP服务未启动")
	}
	sn := strconv.Itoa(id)
	if cycle > 0 {
		body := sipinfra.BuildSubscribeMobilePosition(device.DeviceID, sn, interval)
		if err := s.sip.SendSubscribe(device, "MobilePosition", body, cycle); err != nil {
			return fmt.Errorf("移动位置订阅失败: %w", err)
		}
		device.SubscribeCycleForMobilePosition = cycle
		device.MobilePositionSubmissionInterval = interval
	} else {
		body := sipinfra.BuildSubscribeMobilePosition(device.DeviceID, sn, interval)
		_ = s.sip.SendSubscribeCancel(device, "MobilePosition", body)
		device.SubscribeCycleForMobilePosition = 0
	}
	return s.UpdateDevice(device)
}
