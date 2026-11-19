package deviceapp

import (
	"fmt"
	"time"

	domaindevice "zero-web-server/internal/domain/device"
	"zero-web-server/internal/gbsip"
	"zero-web-server/internal/port"
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

// QueryBasicParam 发起 ConfigDownload 并返回设备实际的基础参数；
// 设备无应答时回退到本地已知字段。
func (s *Service) QueryBasicParam(deviceID string) (map[string]interface{}, error) {
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("设备不存在")
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
	if s.sip == nil || !device.OnLine {
		return out, nil
	}
	sn, ch := s.sip.QueryConfigDownload(device, "BasicParam")
	select {
	case p, ok := <-ch:
		if !ok {
			return out, nil
		}
		out["name"] = p.Name
		out["expiration"] = p.Expiration
		out["heartBeatInterval"] = p.HeartBeatInterval
		out["heartBeatCount"] = p.HeartBeatCount
		out["positionCapability"] = p.PositionCapability
		out["longitude"] = p.Longitude
		out["latitude"] = p.Latitude
		return out, nil
	case <-time.After(5 * time.Second):
		s.sip.CancelQuery(sn)
		return out, nil
	}
}

// QueryHomePosition 查询看守位（2022）。
func (s *Service) QueryHomePosition(deviceID, channelID string) (*gbsip.HomePosition, error) {
	device, err := s.controlTarget(deviceID)
	if err != nil {
		return nil, err
	}
	sn, ch := s.sip.QueryHomePosition(device, channelID)
	return waitQuery(s.sip, sn, ch)
}

// QueryCruiseTrackList 查询巡航轨迹列表（2022）。
func (s *Service) QueryCruiseTrackList(deviceID, channelID string) (*gbsip.CruiseTrackList, error) {
	device, err := s.controlTarget(deviceID)
	if err != nil {
		return nil, err
	}
	sn, ch := s.sip.QueryCruiseTrackList(device, channelID)
	return waitQuery(s.sip, sn, ch)
}

// QueryCruiseTrack 查询单条巡航轨迹（2022）。
func (s *Service) QueryCruiseTrack(deviceID, channelID string, number int) (*gbsip.CruiseTrack, error) {
	device, err := s.controlTarget(deviceID)
	if err != nil {
		return nil, err
	}
	sn, ch := s.sip.QueryCruiseTrack(device, channelID, number)
	return waitQuery(s.sip, sn, ch)
}

// QueryPTZPosition 查询云台精准位置（2022）。
func (s *Service) QueryPTZPosition(deviceID, channelID string) (*gbsip.PTZPosition, error) {
	device, err := s.controlTarget(deviceID)
	if err != nil {
		return nil, err
	}
	sn, ch := s.sip.QueryPTZPosition(device, channelID)
	return waitQuery(s.sip, sn, ch)
}

// QuerySDCardStatus 查询存储卡状态（2022）。
func (s *Service) QuerySDCardStatus(deviceID, channelID string) (*gbsip.SDCardStatus, error) {
	device, err := s.controlTarget(deviceID)
	if err != nil {
		return nil, err
	}
	sn, ch := s.sip.QuerySDCardStatus(device, channelID)
	return waitQuery(s.sip, sn, ch)
}

// controlTarget 校验 SIP 就绪、设备存在且在线。
func (s *Service) controlTarget(deviceID string) (*domaindevice.Device, error) {
	if s.sip == nil {
		return nil, fmt.Errorf("SIP 未就绪")
	}
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("设备不存在")
	}
	if !device.OnLine {
		return nil, fmt.Errorf("设备离线")
	}
	return device, nil
}

// waitQuery 等待查询应答，超时返回中文错误。
func waitQuery[T any](sip port.GBDeviceControl, sn string, ch <-chan T) (*T, error) {
	select {
	case v, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("设备未应答")
		}
		return &v, nil
	case <-time.After(5 * time.Second):
		sip.CancelQuery(sn)
		return nil, fmt.Errorf("设备应答超时")
	}
}

// AlarmReset 发送报警复位（AlarmCmd=ResetAlarm）。
func (s *Service) AlarmReset(deviceID, channelID string) error {
	return s.controlDevice(deviceID, func(device *domaindevice.Device) error {
		return s.sip.SendAlarmCmd(device, channelID, "ResetAlarm")
	})
}

// TeleBoot 远程重启设备。
func (s *Service) TeleBoot(deviceID, channelID string) error {
	return s.controlDevice(deviceID, func(device *domaindevice.Device) error {
		return s.sip.SendTeleBoot(device, channelID)
	})
}

// IFrame 请求强制关键帧。
func (s *Service) IFrame(deviceID, channelID string) error {
	return s.controlDevice(deviceID, func(device *domaindevice.Device) error {
		return s.sip.SendIFrameCmd(device, channelID)
	})
}

// controlDevice 统一做 SIP 就绪、设备在线校验后执行控制。
func (s *Service) controlDevice(deviceID string, fn func(device *domaindevice.Device) error) error {
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
	return fn(device)
}

// QueryDeviceStatus 查询设备状态（DeviceStatus），5 秒内无应答返回错误。
func (s *Service) QueryDeviceStatus(deviceID string) (*port.GBDeviceStatus, error) {
	if s.sip == nil {
		return nil, fmt.Errorf("SIP 未就绪")
	}
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("设备不存在")
	}
	if !device.OnLine {
		return nil, fmt.Errorf("设备离线")
	}
	sn, ch := s.sip.SendDeviceStatusQuery(device)
	select {
	case st, ok := <-ch:
		if !ok || st.Online == "" && st.Status == "" {
			return nil, fmt.Errorf("设备未应答状态查询")
		}
		return &st, nil
	case <-time.After(5 * time.Second):
		s.sip.CancelDeviceStatusQuery(sn)
		return nil, fmt.Errorf("设备状态查询超时")
	}
}
