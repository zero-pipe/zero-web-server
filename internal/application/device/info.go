package deviceapp

import (
	"context"
	"log"
	"time"
)

func (s *Service) HandleDeviceInfo(deviceID, name, manufacturer, model, firmware string) error {
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return err
	}
	if name != "" {
		device.Name = name
	}
	if manufacturer != "" {
		device.Manufacturer = manufacturer
	}
	if model != "" {
		device.Model = model
	}
	if firmware != "" {
		device.Firmware = firmware
	}
	device.UpdateTime = nowStr()
	if err := s.devices.UpdateFields(device.DeviceID, map[string]interface{}{
		"name":         device.Name,
		"manufacturer": device.Manufacturer,
		"model":        device.Model,
		"firmware":     device.Firmware,
		"update_time":  device.UpdateTime,
	}); err != nil {
		return err
	}
	if s.redis != nil {
		_ = s.redis.UpdateDevice(context.Background(), device)
	}
	log.Printf("GB28181 device info: device=%s name=%s manufacturer=%s model=%s",
		deviceID, device.Name, device.Manufacturer, device.Model)
	return nil
}

func (s *Service) autoQueryDeviceInfo(deviceID string) {
	if s.sip == nil || deviceID == "" {
		return
	}
	time.Sleep(3 * time.Second)
	device, err := s.GetByDeviceID(deviceID)
	if err != nil || device == nil || !device.OnLine {
		return
	}
	// 厂商为空则补查；部分设备 Model 可空
	if device.Manufacturer != "" {
		return
	}
	if err := s.sip.SendDeviceInfoQuery(device); err != nil {
		log.Printf("GB28181 device info query: device=%s err=%v", deviceID, err)
		return
	}
	log.Printf("GB28181 device info query sent: device=%s", deviceID)
}
