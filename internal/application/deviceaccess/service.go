package deviceaccess

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	deviceapp "zero-web-kit/internal/application/device"
	onvifapp "zero-web-kit/internal/application/onvif"
	domaindevice "zero-web-kit/internal/domain/device"
	domainonvif "zero-web-kit/internal/domain/onvif"
)

const (
	AccessPassive = "passive"
	AccessActive  = "active"

	ProtocolGB28181 = "gb28181"
	ProtocolONVIF   = "onvif"

	StatusPending = "pending"
	StatusOnline  = "online"
	StatusOffline = "offline"
)

type DeviceView struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	AccessMode   string       `json:"accessMode"`
	Protocol     string       `json:"protocol"`
	Vendor       string       `json:"vendor"`
	DeviceType   string       `json:"deviceType,omitempty"`
	Status       string       `json:"status"`
	ChannelCount int          `json:"channelCount"`
	AddressText  string       `json:"addressText"`
	RawID        string       `json:"rawId"`
	DBID         int          `json:"dbId,omitempty"` // 国标库内自增 id（订阅接口用）
	Capabilities []string     `json:"capabilities"`
	// 国标能力列；其它协议为空
	StreamMode                      string `json:"streamMode,omitempty"`
	SubscribeCycleForCatalog        int    `json:"subscribeCycleForCatalog,omitempty"`
	SubscribeCycleForMobilePosition int    `json:"subscribeCycleForMobilePosition,omitempty"`
	SubscribeCycleForAlarm          int    `json:"subscribeCycleForAlarm,omitempty"`
	GB           *GBConfig    `json:"gb,omitempty"`
	Onvif        *OnvifConfig `json:"onvif,omitempty"`
	CreateTime   string       `json:"createTime"`
	UpdateTime   string       `json:"updateTime"`
}

type GBConfig struct {
	DeviceID      string `json:"deviceId"`
	Password      string `json:"password"`
	SdpIP         string `json:"sdpIp"`
	MediaServerID string `json:"mediaServerId"`
	Charset       string `json:"charset"`
}

type OnvifConfig struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateRequest struct {
	AccessMode string       `json:"accessMode"`
	Protocol   string       `json:"protocol"`
	Name       string       `json:"name"`
	Vendor     string       `json:"vendor"`
	DeviceType string       `json:"deviceType"`
	GB         *GBConfig    `json:"gb"`
	Onvif      *OnvifConfig `json:"onvif"`
}

// UpdateRequest 编辑设备：名称/厂商/协议侧可改字段（编号、IP 等主键类字段一般不改）
type UpdateRequest struct {
	Name   string       `json:"name"`
	Vendor string       `json:"vendor"`
	GB     *GBConfig    `json:"gb"`
	Onvif  *OnvifConfig `json:"onvif"`
}

type ListQuery struct {
	Page       int
	Count      int
	Query      string
	AccessMode string
	Protocol   string
	Status     string
}

type Service struct {
	gb    *deviceapp.Service
	onvif *onvifapp.Service
}

func NewService(gb *deviceapp.Service, onvif *onvifapp.Service) *Service {
	return &Service{gb: gb, onvif: onvif}
}

func (s *Service) List(ctx context.Context, q ListQuery) ([]*DeviceView, int64, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Count <= 0 {
		q.Count = 15
	}

	wantGB := q.Protocol == "" || q.Protocol == ProtocolGB28181
	wantONVIF := q.Protocol == "" || q.Protocol == ProtocolONVIF
	if q.AccessMode == AccessPassive {
		wantONVIF = false
	}
	if q.AccessMode == AccessActive {
		wantGB = false
	}

	var views []*DeviceView

	if wantGB {
		var online *bool
		switch q.Status {
		case StatusOnline:
			v := true
			online = &v
		case StatusOffline, StatusPending:
			v := false
			online = &v
		}
		list, _, err := s.gb.List(1, 5000, q.Query, online)
		if err != nil {
			return nil, 0, err
		}
		for _, d := range list {
			v := viewFromGB(d)
			if matchStatus(v, q.Status) {
				views = append(views, v)
			}
		}
	}

	if wantONVIF {
		list, _, err := s.onvif.ListDevices(ctx, 1, 5000, q.Query)
		if err != nil {
			return nil, 0, err
		}
		for _, d := range list {
			v := viewFromONVIF(d)
			if n, err := s.onvif.ChannelCount(ctx, d.ID); err == nil {
				v.ChannelCount = n
			}
			if matchStatus(v, q.Status) {
				views = append(views, v)
			}
		}
	}

	sort.Slice(views, func(i, j int) bool {
		return views[i].UpdateTime > views[j].UpdateTime
	})

	total := int64(len(views))
	start := (q.Page - 1) * q.Count
	if start >= len(views) {
		return []*DeviceView{}, total, nil
	}
	end := start + q.Count
	if end > len(views) {
		end = len(views)
	}
	return views[start:end], total, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*DeviceView, error) {
	req.AccessMode = strings.ToLower(strings.TrimSpace(req.AccessMode))
	req.Protocol = strings.ToLower(strings.TrimSpace(req.Protocol))
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, fmt.Errorf("设备名称不能为空")
	}
	if err := s.ensureNameUnique(ctx, req.Name); err != nil {
		return nil, err
	}

	switch {
	case req.AccessMode == AccessPassive && req.Protocol == ProtocolGB28181:
		return s.createGB(req)
	case req.AccessMode == AccessActive && req.Protocol == ProtocolONVIF:
		return s.createONVIF(ctx, req)
	default:
		return nil, fmt.Errorf("不支持的接入组合: mode=%s protocol=%s", req.AccessMode, req.Protocol)
	}
}

func (s *Service) ensureNameUnique(ctx context.Context, name string) error {
	return s.ensureNameUniqueExcept(ctx, name, "", "")
}

func (s *Service) ensureNameUniqueExcept(ctx context.Context, name, exceptProtocol, exceptRawID string) error {
	gbList, _, err := s.gb.List(1, 5000, name, nil)
	if err != nil {
		return err
	}
	for _, d := range gbList {
		if exceptProtocol == ProtocolGB28181 && d.DeviceID == exceptRawID {
			continue
		}
		n := d.Name
		if d.CustomName != "" {
			n = d.CustomName
		}
		if strings.EqualFold(strings.TrimSpace(n), name) {
			return fmt.Errorf("设备名称已存在: %s", name)
		}
	}
	onvifList, _, err := s.onvif.ListDevices(ctx, 1, 5000, name)
	if err != nil {
		return err
	}
	for _, d := range onvifList {
		raw := strconv.FormatInt(d.ID, 10)
		if exceptProtocol == ProtocolONVIF && raw == exceptRawID {
			continue
		}
		n := d.Name
		if d.CustomName != "" {
			n = d.CustomName
		}
		if strings.EqualFold(strings.TrimSpace(n), name) {
			return fmt.Errorf("设备名称已存在: %s", name)
		}
	}
	return nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*DeviceView, error) {
	protocol, raw, err := ParseID(id)
	if err != nil {
		return nil, err
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, fmt.Errorf("设备名称不能为空")
	}
	if err := s.ensureNameUniqueExcept(ctx, req.Name, protocol, raw); err != nil {
		return nil, err
	}

	switch protocol {
	case ProtocolGB28181:
		return s.updateGB(raw, req)
	case ProtocolONVIF:
		oid, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("无效的 ONVIF 设备ID")
		}
		return s.updateONVIF(ctx, oid, req)
	default:
		return nil, fmt.Errorf("未知协议")
	}
}

func (s *Service) updateGB(deviceID string, req UpdateRequest) (*DeviceView, error) {
	existing, err := s.gb.GetByDeviceID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("设备不存在")
	}
	existing.Name = req.Name
	existing.Manufacturer = strings.TrimSpace(req.Vendor)
	if req.GB != nil {
		existing.Password = req.GB.Password
		existing.SDPIP = req.GB.SdpIP
		if req.GB.MediaServerID != "" {
			existing.MediaServerID = req.GB.MediaServerID
		}
		if req.GB.Charset != "" {
			existing.Charset = req.GB.Charset
		}
	}
	if err := s.gb.UpdateDevice(existing); err != nil {
		return nil, err
	}
	return viewFromGB(existing), nil
}

func (s *Service) updateONVIF(ctx context.Context, id int64, req UpdateRequest) (*DeviceView, error) {
	upd := onvifapp.UpdateDeviceRequest{
		Name:   req.Name,
		Vendor: strings.TrimSpace(req.Vendor),
	}
	if req.Onvif != nil {
		upd.Username = req.Onvif.Username
		upd.Password = req.Onvif.Password
		if req.Onvif.Port > 0 {
			upd.Port = req.Onvif.Port
		}
	}
	dev, err := s.onvif.UpdateDevice(ctx, id, upd)
	if err != nil {
		return nil, err
	}
	return viewFromONVIF(dev), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	protocol, raw, err := ParseID(id)
	if err != nil {
		return err
	}
	switch protocol {
	case ProtocolGB28181:
		return s.gb.Delete(raw)
	case ProtocolONVIF:
		oid, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("无效的 ONVIF 设备ID")
		}
		return s.onvif.DeleteDevice(ctx, oid)
	default:
		return fmt.Errorf("未知协议")
	}
}

func (s *Service) Sync(ctx context.Context, id string) error {
	protocol, raw, err := ParseID(id)
	if err != nil {
		return err
	}
	switch protocol {
	case ProtocolGB28181:
		return s.gb.SyncCatalog(raw)
	case ProtocolONVIF:
		oid, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("无效的 ONVIF 设备ID")
		}
		_, err = s.onvif.SyncChannels(ctx, oid)
		return err
	default:
		return fmt.Errorf("未知协议")
	}
}

func (s *Service) createGB(req CreateRequest) (*DeviceView, error) {
	if req.GB == nil || strings.TrimSpace(req.GB.DeviceID) == "" {
		return nil, fmt.Errorf("国标设备编号不能为空")
	}
	name := strings.TrimSpace(req.Name)
	charset := req.GB.Charset
	if charset == "" {
		charset = "GB2312"
	}
	mediaServerID := req.GB.MediaServerID
	if mediaServerID == "" {
		mediaServerID = "auto"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	d := &domaindevice.Device{
		DeviceID:          strings.TrimSpace(req.GB.DeviceID),
		Name:              name,
		Manufacturer:      strings.TrimSpace(req.Vendor),
		Password:          req.GB.Password,
		SDPIP:             req.GB.SdpIP,
		MediaServerID:     mediaServerID,
		Charset:           charset,
		StreamMode:        "UDP",
		OnLine:            false,
		HeartBeatInterval: 60,
		HeartBeatCount:    3,
		CreateTime:        now,
		UpdateTime:        now,
	}
	if err := s.gb.AddDevice(d); err != nil {
		return nil, err
	}
	return viewFromGB(d), nil
}

func (s *Service) createONVIF(ctx context.Context, req CreateRequest) (*DeviceView, error) {
	if req.Onvif == nil || strings.TrimSpace(req.Onvif.IP) == "" {
		return nil, fmt.Errorf("ONVIF 设备 IP 不能为空")
	}
	port := req.Onvif.Port
	if port <= 0 {
		port = 80
	}
	dev, err := s.onvif.AddDevice(ctx, onvifapp.AddDeviceRequest{
		Name:     strings.TrimSpace(req.Name),
		IP:       strings.TrimSpace(req.Onvif.IP),
		Port:     port,
		Username: req.Onvif.Username,
		Password: req.Onvif.Password,
	})
	if err != nil {
		return nil, err
	}
	v := viewFromONVIF(dev)
	if n, err := s.onvif.ChannelCount(ctx, dev.ID); err == nil {
		v.ChannelCount = n
	}
	return v, nil
}

func viewFromGB(d *domaindevice.Device) *DeviceView {
	status := StatusPending
	addr := "待注册"
	if d.OnLine {
		status = StatusOnline
		if d.HostAddress != "" {
			addr = d.HostAddress
		} else if d.IP != "" {
			addr = fmt.Sprintf("%s:%d", d.IP, d.Port)
		}
	} else if d.HostAddress != "" || d.IP != "" {
		status = StatusOffline
		if d.HostAddress != "" {
			addr = d.HostAddress
		} else {
			addr = fmt.Sprintf("%s:%d", d.IP, d.Port)
		}
	}
	name := d.Name
	if d.CustomName != "" {
		name = d.CustomName
	}
	return &DeviceView{
		ID:                              "gb:" + d.DeviceID,
		Name:                            name,
		AccessMode:                      AccessPassive,
		Protocol:                        ProtocolGB28181,
		Vendor:                          d.Manufacturer,
		Status:                          status,
		ChannelCount:                    d.ChannelCount,
		AddressText:                     addr,
		RawID:                           d.DeviceID,
		DBID:                            d.ID,
		StreamMode:                      d.StreamMode,
		SubscribeCycleForCatalog:        d.SubscribeCycleForCatalog,
		SubscribeCycleForMobilePosition: d.SubscribeCycleForMobilePosition,
		SubscribeCycleForAlarm:          d.SubscribeCycleForAlarm,
		// 国标操作/能力列：刷新、布防、订阅、统计、流传输…
		Capabilities: []string{
			"catalog", "transport", "subscribe", "stats", "guard", "play", "ptz", "alarm", "edit", "delete",
		},
		GB: &GBConfig{
			DeviceID:      d.DeviceID,
			Password:      d.Password,
			SdpIP:         d.SDPIP,
			MediaServerID: d.MediaServerID,
			Charset:       d.Charset,
		},
		CreateTime: d.CreateTime,
		UpdateTime: d.UpdateTime,
	}
}

func viewFromONVIF(d *domainonvif.Device) *DeviceView {
	status := StatusOffline
	if d.OnLine {
		status = StatusOnline
	}
	name := d.Name
	if d.CustomName != "" {
		name = d.CustomName
	}
	return &DeviceView{
		ID:           fmt.Sprintf("onvif:%d", d.ID),
		Name:         name,
		AccessMode:   AccessActive,
		Protocol:     ProtocolONVIF,
		Vendor:       d.Manufacturer,
		Status:       status,
		ChannelCount: 0,
		AddressText:  fmt.Sprintf("%s:%d", d.IP, d.Port),
		RawID:        strconv.FormatInt(d.ID, 10),
		DBID:         int(d.ID),
		// ONVIF：无流传输/订阅/统计；操作只有同步/通道/编辑/删除
		Capabilities: []string{"sync", "play", "ptz", "edit", "delete"},
		Onvif: &OnvifConfig{
			IP:       d.IP,
			Port:     d.Port,
			Username: d.Username,
			Password: d.Password,
		},
		CreateTime: d.CreateTime,
		UpdateTime: d.UpdateTime,
	}
}

func matchStatus(v *DeviceView, status string) bool {
	if status == "" {
		return true
	}
	return v.Status == status
}

// ParseID 解析统一设备 ID：gb:{deviceId} / onvif:{id}
func ParseID(id string) (protocol, raw string, err error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", "", fmt.Errorf("设备ID为空")
	}
	if i := strings.IndexByte(id, ':'); i > 0 {
		return id[:i], id[i+1:], nil
	}
	if _, e := strconv.ParseInt(id, 10, 64); e == nil {
		return ProtocolONVIF, id, nil
	}
	return ProtocolGB28181, id, nil
}
