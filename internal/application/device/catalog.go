package deviceapp

import (
	"context"
	"errors"
	"log"
	"time"
	domainchannel "zero-web-server/internal/domain/channel"
	domaindevice "zero-web-server/internal/domain/device"
	"zero-web-server/internal/gbsip"
)

func (s *Service) autoSyncCatalogOnce(deviceID string) {
	if deviceID == "" {
		return
	}
	_, total, err := s.channels.ListByDevice(deviceID, 1, 1, "", nil)
	if err != nil || total > 0 {
		return
	}
	time.Sleep(2 * time.Second)
	if err := s.SyncCatalog(deviceID); err != nil {
		log.Printf("GB28181 auto catalog sync: device=%s err=%v", deviceID, err)
	}
	time.Sleep(10 * time.Second)
	_, total2, _ := s.channels.ListByDevice(deviceID, 1, 1, "", nil)
	if total2 > 0 {
		return
	}
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return
	}
	s.ensureDeviceDBID(device)
	if err := s.ensureDefaultIPCChannel(device); err != nil {
		log.Printf("GB28181 default channel: device=%s err=%v", deviceID, err)
		return
	}
	log.Printf("GB28181 default channel created: device=%s", deviceID)
}

func (s *Service) HandleCatalog(deviceID string, items []gbsip.CatalogItem) error {
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return err
	}
	s.ensureDeviceDBID(device)
	now := nowStr()
	deviceName := device.Name
	if device.CustomName != "" {
		deviceName = device.CustomName
	}
	channels := make([]*domainchannel.Channel, 0, len(items))
	idx := 0
	for _, item := range items {
		if item.DeviceID == "" {
			continue
		}
		idx++
		ch := &domainchannel.Channel{
			DeviceID:     deviceID,
			GBDeviceID:   item.DeviceID,
			Name:         channelDisplayName(deviceName, idx),
			Manufacturer: item.Manufacturer,
			Model:        item.Model,
			Parental:     item.Parental,
			ParentID:     item.ParentID,
			Status:       item.Status,
			Longitude:    item.Longitude,
			Latitude:     item.Latitude,
			PTZType:      item.PTZType,
			CreateTime:   now,
			UpdateTime:   now,
		}
		if ch.Status == "" {
			ch.Status = "ON"
		}
		channels = append(channels, ch)
	}
	if len(channels) == 0 {
		if err := s.ensureDefaultIPCChannel(device); err == nil {
			_, total, _ := s.channels.ListByDevice(deviceID, 1, 1, "", nil)
			if total > 0 {
				s.syncStatus.Store(deviceID, &SyncStatus{Total: int(total), Current: int(total), SyncIng: false})
				log.Printf("GB28181 catalog fallback: device=%s default_ipc_channel=1", deviceID)
				return nil
			}
		}
		s.syncStatus.Store(deviceID, &SyncStatus{ErrorMsg: "目录为空，请检查摄像机目录配置", SyncIng: false})
		return nil
	}
	s.syncStatus.Store(deviceID, &SyncStatus{Total: len(channels), Current: len(channels), SyncIng: false})
	log.Printf("GB28181 catalog synced: device=%s channels=%d", deviceID, len(channels))
	if err := s.channels.ResetByDevice(deviceID, device.ID, channels); err != nil {
		return err
	}
	s.backfillDeviceFromChannels(device, channels)
	return nil
}

// backfillDeviceFromChannels 设备厂商/型号常在 Catalog Item 里，DeviceInfo 未回时从通道补齐。
// 不再用通道名回填设备名（通道已统一为 设备名_channel_N）。
func (s *Service) backfillDeviceFromChannels(device *domaindevice.Device, channels []*domainchannel.Channel) {
	if device == nil || len(channels) == 0 {
		return
	}
	changed := false
	for _, ch := range channels {
		if ch == nil {
			continue
		}
		if device.Manufacturer == "" && ch.Manufacturer != "" {
			device.Manufacturer = ch.Manufacturer
			changed = true
		}
		if device.Model == "" && ch.Model != "" {
			device.Model = ch.Model
			changed = true
		}
		if changed && device.Manufacturer != "" && device.Model != "" {
			break
		}
	}
	if !changed {
		return
	}
	device.UpdateTime = nowStr()
	if err := s.devices.UpdateFields(device.DeviceID, map[string]interface{}{
		"manufacturer": device.Manufacturer,
		"model":        device.Model,
		"update_time":  device.UpdateTime,
	}); err != nil {
		log.Printf("GB28181 backfill device identity: device=%s err=%v", device.DeviceID, err)
		return
	}
	if s.redis != nil {
		_ = s.redis.UpdateDevice(context.Background(), device)
	}
	log.Printf("GB28181 device identity from catalog: device=%s name=%s manufacturer=%s",
		device.DeviceID, device.Name, device.Manufacturer)
}

func (s *Service) ensureDefaultIPCChannel(device *domaindevice.Device) error {
	if device == nil || len(device.DeviceID) < 13 {
		return errors.New("invalid device")
	}
	devType := device.DeviceID[10:13]
	if devType != "132" && devType != "131" {
		return errors.New("not ipc")
	}
	_, total, err := s.channels.ListByDevice(device.DeviceID, 1, 1, "", nil)
	if err != nil {
		return err
	}
	if total > 0 {
		return nil
	}
	now := nowStr()
	deviceName := device.Name
	if device.CustomName != "" {
		deviceName = device.CustomName
	}
	// 132 设备常见做法：通道国标号为同号段的 131 类型（…131…0001），
	// EasyGBDSimulator 等多通道模拟器也按此编码；INVITE 设备号会导致设备侧通道表 miss。
	ch := &domainchannel.Channel{
		DeviceID:     device.DeviceID,
		GBDeviceID:   defaultIPCChannelGBID(device.DeviceID),
		Name:         channelDisplayName(deviceName, 1),
		Manufacturer: device.Manufacturer,
		Model:        device.Model,
		Status:       "ON",
		PTZType:      3,
		CreateTime:   now,
		UpdateTime:   now,
	}
	return s.channels.ResetByDevice(device.DeviceID, device.ID, []*domainchannel.Channel{ch})
}

// defaultIPCChannelGBID 将 132 IPC 设备号映射为默认摄像机通道号（类型位改 131，末四位 0001）。
func defaultIPCChannelGBID(deviceID string) string {
	if len(deviceID) < 20 {
		return deviceID
	}
	if deviceID[10:13] != "132" {
		return deviceID
	}
	b := []byte(deviceID)
	b[10], b[11], b[12] = '1', '3', '1'
	copy(b[len(b)-4:], []byte("0001"))
	return string(b)
}

func (s *Service) SyncCatalog(deviceID string) error {
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return ErrDeviceNotFound
	}
	s.syncStatus.Store(deviceID, &SyncStatus{Total: 0, Current: 0, SyncIng: true, Started: time.Now()})
	if s.sip == nil {
		return errors.New("SIP服务未启动")
	}
	// IPC 也发 DeviceInfo/Catalog：厂商、真实设备名都在协议响应里，不能只建默认通道
	if device.Manufacturer == "" {
		if err := s.sip.SendDeviceInfoQuery(device); err != nil {
			log.Printf("GB28181 device info on sync: device=%s err=%v", deviceID, err)
		}
	}
	if isIPCDevice(deviceID) {
		_ = s.sip.SendCatalogQuery(device)
		return s.syncIPCChannels(device)
	}
	return s.sip.SendCatalogQuery(device)
}

func isIPCDevice(deviceID string) bool {
	if len(deviceID) < 13 {
		return false
	}
	switch deviceID[10:13] {
	case "131", "132":
		return true
	default:
		return false
	}
}

func (s *Service) syncIPCChannels(device *domaindevice.Device) error {
	s.ensureDeviceDBID(device)
	if err := s.ensureDefaultIPCChannel(device); err != nil {
		s.syncStatus.Store(device.DeviceID, &SyncStatus{ErrorMsg: err.Error(), SyncIng: false})
		return err
	}
	_, total, err := s.channels.ListByDevice(device.DeviceID, 1, 1, "", nil)
	if err != nil {
		s.syncStatus.Store(device.DeviceID, &SyncStatus{ErrorMsg: err.Error(), SyncIng: false})
		return err
	}
	s.syncStatus.Store(device.DeviceID, &SyncStatus{
		Total: int(total), Current: int(total), SyncIng: false,
	})
	log.Printf("GB28181 IPC channel sync: device=%s channels=%d", device.DeviceID, total)
	return nil
}

func (s *Service) GetSyncStatus(deviceID string) SyncStatus {
	if v, ok := s.syncStatus.Load(deviceID); ok {
		if st, ok := v.(*SyncStatus); ok {
			if st.SyncIng && !st.Started.IsZero() && time.Since(st.Started) > 30*time.Second {
				if device, err := s.GetByDeviceID(deviceID); err == nil && isIPCDevice(deviceID) {
					_ = s.syncIPCChannels(device)
					if v2, ok2 := s.syncStatus.Load(deviceID); ok2 {
						if st2, ok3 := v2.(*SyncStatus); ok3 && !st2.SyncIng {
							return *st2
						}
					}
				}
				return SyncStatus{ErrorMsg: "同步超时，请确认设备在线；单目摄像机无需目录查询，可刷新通道列表查看"}
			}
			return *st
		}
	}
	return SyncStatus{}
}

func (s *Service) ListChannels(deviceID string, page, count int, query string, online *bool) ([]*domainchannel.Channel, int64, error) {
	list, total, err := s.channels.ListByDevice(deviceID, page, count, query, online)
	if err != nil || total > 0 || page > 1 {
		return list, total, err
	}
	device, derr := s.GetByDeviceID(deviceID)
	if derr != nil {
		return list, total, err
	}
	s.ensureDeviceDBID(device)
	if err := s.ensureDefaultIPCChannel(device); err != nil {
		return list, total, err
	}
	return s.channels.ListByDevice(deviceID, page, count, query, online)
}

func (s *Service) GetChannel(deviceID, channelDeviceID string) (*domainchannel.Channel, error) {
	return s.channels.GetOne(deviceID, channelDeviceID)
}
