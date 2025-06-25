package deviceapp

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	domainchannel "zero-web-kit/internal/domain/channel"
	domaindevice "zero-web-kit/internal/domain/device"
	sipinfra "zero-web-kit/internal/infrastructure/sip"
	redisinfra "zero-web-kit/internal/infrastructure/redis"
)

var (
	ErrDeviceNotFound = errors.New("设备不存在")
)

type Service struct {
	devices    domaindevice.Repository
	channels   domainchannel.Repository
	redis      *redisinfra.Client
	sip        *sipinfra.Server
	syncStatus sync.Map // deviceID -> *SyncStatus
}

type SyncStatus struct {
	Total    int
	Current  int
	SyncIng  bool
	ErrorMsg string
	Started  time.Time
}

func NewService(devices domaindevice.Repository, channels domainchannel.Repository, redis *redisinfra.Client) *Service {
	return &Service{
		devices:  devices,
		channels: channels,
		redis:    redis,
	}
}

func (s *Service) SetSIP(sipServer *sipinfra.Server) {
	s.sip = sipServer
}

func (s *Service) GetByDeviceID(deviceID string) (*domaindevice.Device, error) {
	dbDev, dbErr := s.devices.GetByDeviceID(deviceID)
	if s.redis != nil {
		if d, err := s.redis.GetDevice(context.Background(), deviceID); err == nil {
			if dbErr == nil && dbDev != nil {
				d.ID = dbDev.ID
				if d.Password == "" {
					d.Password = dbDev.Password
				}
				if d.Name == "" || d.Name == d.DeviceID {
					if dbDev.Name != "" {
						d.Name = dbDev.Name
					}
				}
			}
			return d, nil
		}
	}
	if dbErr != nil {
		return nil, dbErr
	}
	return dbDev, nil
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
		if err := s.devices.Create(device); err != nil {
			return nil, err
		}
		return device, nil
	}
	device.ID = existing.ID
	if device.Password == "" {
		device.Password = existing.Password
	}
	if device.Name == "" {
		device.Name = existing.Name
	}
	if err := s.devices.Update(device); err != nil {
		return nil, err
	}
	return device, nil
}

func (s *Service) Online(device *domaindevice.Device) error {
	device.OnLine = true
	if err := s.devices.Update(device); err != nil {
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

func (s *Service) HandleCatalog(deviceID string, items []sipinfra.CatalogItem) error {
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return err
	}
	s.ensureDeviceDBID(device)
	now := nowStr()
	channels := make([]*domainchannel.Channel, 0, len(items))
	for _, item := range items {
		if item.DeviceID == "" {
			continue
		}
		ch := &domainchannel.Channel{
			DeviceID:     deviceID,
			GBDeviceID:   item.DeviceID,
			Name:         item.Name,
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
		if ch.Name == "" {
			ch.Name = item.DeviceID
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
	return s.channels.ResetByDevice(deviceID, device.ID, channels)
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
	name := device.Name
	if name == "" || name == device.DeviceID {
		name = "Camera-" + device.DeviceID
	}
	ch := &domainchannel.Channel{
		DeviceID:     device.DeviceID,
		GBDeviceID:   device.DeviceID,
		Name:         name,
		Manufacturer: device.Manufacturer,
		Model:        device.Model,
		Status:       "ON",
		PTZType:      3,
		CreateTime:   now,
		UpdateTime:   now,
	}
	return s.channels.ResetByDevice(device.DeviceID, device.ID, []*domainchannel.Channel{ch})
}

func (s *Service) List(page, count int, query string, online *bool) ([]*domaindevice.Device, int64, error) {
	return s.devices.List(page, count, query, online)
}

func (s *Service) Delete(deviceID string) error {
	_ = s.channels.DeleteByDevice(deviceID)
	return s.devices.DeleteByDeviceID(deviceID)
}

func (s *Service) SyncCatalog(deviceID string) error {
	device, err := s.GetByDeviceID(deviceID)
	if err != nil {
		return ErrDeviceNotFound
	}
	s.syncStatus.Store(deviceID, &SyncStatus{Total: 0, Current: 0, SyncIng: true, Started: time.Now()})
	if isIPCDevice(deviceID) {
		return s.syncIPCChannels(device)
	}
	if s.sip == nil {
		return errors.New("SIP服务未启动")
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

func (s *Service) ChangeChannelAudio(channelID int, audio bool) error {
	if channelID <= 0 {
		return errors.New("通道ID无效")
	}
	if _, err := s.channels.GetByID(channelID); err != nil {
		return errors.New("通道不存在")
	}
	return s.channels.ChangeAudio(channelID, audio)
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

func (s *Service) AddDevice(device *domaindevice.Device) error {
	device.CreateTime = nowStr()
	device.UpdateTime = device.CreateTime
	device.ServerID = device.ServerID
	return s.devices.Create(device)
}

func (s *Service) UpdateDevice(device *domaindevice.Device) error {
	device.UpdateTime = nowStr()
	return s.devices.Update(device)
}

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
	if err := s.devices.Update(device); err != nil {
		return err
	}
	if s.redis != nil {
		_ = s.redis.UpdateDevice(context.Background(), device)
	}
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
	if device.Manufacturer != "" && device.Model != "" {
		return
	}
	if err := s.sip.SendDeviceInfoQuery(device); err != nil {
		log.Printf("GB28181 device info query: device=%s err=%v", deviceID, err)
	}
}

func (s *Service) GetKeepaliveStatistics(deviceID string, count int) []domaindevice.TimeStatistics {
	return s.getTimeStatistics(deviceID, count, true)
}

func (s *Service) GetRegisterStatistics(deviceID string, count int) []domaindevice.TimeStatistics {
	return s.getTimeStatistics(deviceID, count, false)
}

func (s *Service) getTimeStatistics(deviceID string, count int, keepalive bool) []domaindevice.TimeStatistics {
	if s.redis == nil || deviceID == "" {
		return []domaindevice.TimeStatistics{}
	}
	var (
		stamps []int64
		err    error
	)
	if keepalive {
		stamps, err = s.redis.GetKeepaliveTimeStamps(context.Background(), deviceID, count)
	} else {
		stamps, err = s.redis.GetRegisterTimeStamps(context.Background(), deviceID, count)
	}
	if err != nil || len(stamps) == 0 {
		return []domaindevice.TimeStatistics{}
	}
	return formatTimeStatistics(stamps, count)
}

func formatTimeStatistics(timeStampList []int64, count int) []domaindevice.TimeStatistics {
	list := make([]domaindevice.TimeStatistics, 0, len(timeStampList))
	for i, ts := range timeStampList {
		item := domaindevice.TimeStatistics{
			Time: time.UnixMilli(ts).Format("2006-01-02 15:04:05"),
		}
		if i > 0 {
			item.TimeDiff = (ts - timeStampList[i-1]) / 1000
		}
		list = append(list, item)
	}
	if len(list) > 1 {
		list = list[1:]
	}
	if count > 0 && len(list) > count {
		list = list[len(list)-count:]
	}
	return list
}

func nowStr() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// Ensure Service implements sipinfra.DeviceService
var _ sipinfra.DeviceService = (*Service)(nil)
