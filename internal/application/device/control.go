package deviceapp

import (
	"fmt"
)

func (s *Service) Guard(deviceID, guardCmd string) error {
	if s.sip == nil {
		return fmt.Errorf("SIP 未就绪")
	}
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return fmt.Errorf("设备不存在")
	}
	if !device.OnLine {
		return fmt.Errorf("设备离线")
	}
	return s.sip.SendGuardCmd(device, device.DeviceID, guardCmd)
}

// Record 向国标设备发送录像控制（Record / StopRecord）。
func (s *Service) Record(deviceID, channelID, recordCmd string) error {
	if s.sip == nil {
		return fmt.Errorf("SIP 未就绪")
	}
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return fmt.Errorf("设备不存在")
	}
	if !device.OnLine {
		return fmt.Errorf("设备离线")
	}
	return s.sip.SendRecordCmd(device, channelID, recordCmd)
}

// DragZoom 拉框放大/缩小。
func (s *Service) DragZoom(deviceID, channelID string, zoomIn bool, length, width, midX, midY, lengthX, lengthY int) error {
	if s.sip == nil {
		return fmt.Errorf("SIP 未就绪")
	}
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return fmt.Errorf("设备不存在")
	}
	if !device.OnLine {
		return fmt.Errorf("设备离线")
	}
	return s.sip.SendDragZoom(device, channelID, zoomIn, length, width, midX, midY, lengthX, lengthY)
}

// HomePosition 看守位。
func (s *Service) HomePosition(deviceID, channelID string, enabled, resetTime, presetIndex int) error {
	if s.sip == nil {
		return fmt.Errorf("SIP 未就绪")
	}
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return fmt.Errorf("设备不存在")
	}
	if !device.OnLine {
		return fmt.Errorf("设备离线")
	}
	return s.sip.SendHomePosition(device, channelID, enabled, resetTime, presetIndex)
}

// SetBasicParam 下发基础参数。
func (s *Service) SetBasicParam(deviceID, name string, expiration, heartBeatInterval, heartBeatCount, positionCapability int) error {
	if s.sip == nil {
		return fmt.Errorf("SIP 未就绪")
	}
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return fmt.Errorf("设备不存在")
	}
	if !device.OnLine {
		return fmt.Errorf("设备离线")
	}
	return s.sip.SendBasicParamConfig(device, name, expiration, heartBeatInterval, heartBeatCount, positionCapability)
}

// QueryBasicParam 发起 ConfigDownload（暂无等待设备回包，返回设备本地已知字段）。
func (s *Service) QueryBasicParam(deviceID string) (map[string]interface{}, error) {
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("设备不存在")
	}
	if s.sip != nil && device.OnLine {
		_ = s.sip.SendConfigDownloadQuery(device, "BasicParam")
	}
	out := map[string]interface{}{
		"deviceId":           device.DeviceID,
		"name":               device.Name,
		"expiration":         device.Expires,
		"heartBeatInterval":  device.HeartBeatInterval,
		"heartBeatCount":     device.HeartBeatCount,
		"positionCapability": 0,
		"longitude":          nil,
		"latitude":           nil,
	}
	return out, nil
}
