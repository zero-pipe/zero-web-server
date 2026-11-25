package deviceapp

import (
	"context"
	domaindevice "zero-web-server/internal/domain/device"
)

func (s *Service) GetByDeviceID(deviceID string) (*domaindevice.Device, error) {
	dbDev, dbErr := s.devices.GetByDeviceID(deviceID)
	if dbErr != nil {
		// 库已删则清 Redis 设备缓存，避免删设备后重注册带回旧名称。
		// 禁止在此调用 RemoveExpiry：它内部会再 GetByDeviceID，设备不存在时无限递归，导致 SIP 无法回 401。
		if s.redis != nil && deviceID != "" {
			_ = s.redis.RemoveDevice(context.Background(), deviceID)
		}
		return nil, dbErr
	}
	if s.redis != nil {
		if d, err := s.redis.GetDevice(context.Background(), deviceID); err == nil {
			mergeDeviceIdentityFromDB(d, dbDev)
			return d, nil
		}
	}
	return dbDev, nil
}

func (s *Service) GetByInternalCode(code string) (*domaindevice.Device, error) {
	return s.devices.GetByInternalCode(code)
}

// mergeDeviceIdentityFromDB 用库表补齐 Redis 缓存里缺失的身份字段，避免注册/心跳把厂商等写空。
func mergeDeviceIdentityFromDB(dst, db *domaindevice.Device) {
	if dst == nil || db == nil {
		return
	}
	dst.ID = db.ID
	if db.InternalCode != "" {
		dst.InternalCode = db.InternalCode
	}
	if dst.Password == "" {
		dst.Password = db.Password
	}
	if dst.Name == "" || dst.Name == dst.DeviceID {
		if db.Name != "" {
			dst.Name = db.Name
		}
	}
	if dst.Manufacturer == "" {
		dst.Manufacturer = db.Manufacturer
	}
	if dst.Model == "" {
		dst.Model = db.Model
	}
	if dst.Firmware == "" {
		dst.Firmware = db.Firmware
	}
	if dst.StreamMode == "" {
		dst.StreamMode = db.StreamMode
	}
	if dst.Charset == "" {
		dst.Charset = db.Charset
	}
	if dst.CustomName == "" {
		dst.CustomName = db.CustomName
	}
}

func (s *Service) ensureDeviceDBID(device *domaindevice.Device) {
	if device == nil || device.ID > 0 {
		return
	}
	if dbDev, err := s.devices.GetByDeviceID(device.DeviceID); err == nil {
		device.ID = dbDev.ID
	}
}

func (s *Service) SaveRegister(device *domaindevice.Device) (*domaindevice.Device, error) {
	existing, err := s.devices.GetByDeviceID(device.DeviceID)
	if err != nil {
		if s.requirePreRegister {
			return nil, ErrDeviceNotPreRegistered
		}
		if err := s.devices.Create(device); err != nil {
			return nil, err
		}
		return device, nil
	}
	device.ID = existing.ID
	// REGISTER 只带信令地址，身份信息以库中已有为准，避免把 DeviceInfo 写入的厂商/型号冲掉
	mergeDeviceIdentityFromDB(device, existing)
	if device.Name == "" {
		device.Name = existing.Name
	}
	if device.MediaServerID == "" {
		device.MediaServerID = existing.MediaServerID
	}
	if device.HeartBeatInterval == 0 {
		device.HeartBeatInterval = existing.HeartBeatInterval
	}
	if device.HeartBeatCount == 0 {
		device.HeartBeatCount = existing.HeartBeatCount
	}
	// 仅回写注册信令字段：对象可能来自 Redis 缓存，全行 Update 会把未加载列写空。
	fields := map[string]interface{}{
		"ip":           device.IP,
		"port":         device.Port,
		"host_address": device.HostAddress,
		"expires":      device.Expires,
		"transport":    device.Transport,
		"update_time":  device.UpdateTime,
	}
	if device.LocalIP != "" {
		fields["local_ip"] = device.LocalIP
	}
	if err := s.devices.UpdateFields(device.DeviceID, fields); err != nil {
		return nil, err
	}
	return device, nil
}

func (s *Service) Online(device *domaindevice.Device) error {
	device.OnLine = true
	// 只更新在线标记，避免整对象全行覆盖（对象可能来自缓存，且与并发写冲突）。
	if err := s.devices.UpdateOnline(device.DeviceID, true); err != nil {
		return err
	}
	if s.redis != nil {
		_ = s.redis.UpdateDevice(context.Background(), device)
	}
	s.TouchExpiry(device)
	return nil
}

func (s *Service) Offline(device *domaindevice.Device) error {
	device.OnLine = false
	if err := s.devices.UpdateOnline(device.DeviceID, false); err != nil {
		return err
	}
	if s.redis != nil {
		_ = s.redis.UpdateDevice(context.Background(), device)
		s.RemoveExpiry(device.DeviceID)
	}
	return nil
}

func (s *Service) HandleKeepalive(deviceID, ip string, port int) error {
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return err
	}
	wasOffline := !device.OnLine
	if ip != "" {
		device.IP = ip
		device.Port = port
		// 心跳携带的信令地址分列写回，避免全行覆盖。
		_ = s.devices.UpdateFields(deviceID, map[string]interface{}{
			"ip": ip, "port": port, "update_time": nowStr(),
		})
	}
	device.UpdateTime = nowStr()
	if !device.OnLine {
		device.OnLine = true
	}
	if err := s.Online(device); err != nil {
		return err
	}
	if wasOffline {
		s.AutoSubscribeOnOnline(device)
		// 离线恢复（含删除后重加、SIP 会话残留等未触发新 REGISTER 的场景）时，
		// 也补一次目录/通道同步，避免通道列表长期为空。
		go s.autoSyncCatalogOnce(device.DeviceID)
	}
	s.TouchExpiry(device)
	return nil
}

func (s *Service) OnDeviceOnline(device *domaindevice.Device) {
	if device == nil {
		return
	}
	s.ensureDeviceDBID(device)
	s.AutoSubscribeOnOnline(device)
	go s.autoSyncCatalogOnce(device.DeviceID)
	go s.autoQueryDeviceInfo(device.DeviceID)
}
