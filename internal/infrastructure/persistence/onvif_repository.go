package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domainonvif "zero-web-server/internal/domain/onvif"
	"zero-web-server/internal/infrastructure/persistence/model"
	"zero-web-server/pkg/idcode"

	"gorm.io/gorm"
)

// OnvifDeviceRepository / OnvifChannelRepository：ONVIF 接入折叠进统一设备/通道表。
//   设备 → zws_device + zws_access_binding(protocol='onvif', config_json=用户名/URIs/发现模式等)
//   通道 → zws_channel + zws_channel_ext(config_json 承载 ONVIF profile 细节) + zws_stream_profile
// onvif.device 的 ID/deviceID 即统一 zws_device.id（BIGINT），保持与设备页一致。

type onvifDeviceConfig struct {
	Username       string `json:"username,omitempty"`
	Password       string `json:"password,omitempty"`
	HardwareID     string `json:"hardwareId,omitempty"`
	DeviceURI      string `json:"deviceUri,omitempty"`
	MediaURI       string `json:"mediaUri,omitempty"`
	PTZURI         string `json:"ptzUri,omitempty"`
	EventSubscribe bool   `json:"eventSubscribe"`
	DiscoveryMode  int    `json:"discoveryMode"`
	CustomName     string `json:"customName,omitempty"`
	ServerID       string `json:"serverId,omitempty"`
}

type onvifChannelConfig struct {
	ProfileToken  string                      `json:"profileToken,omitempty"`
	VideoSource   string                      `json:"videoSource,omitempty"`
	EncoderToken  string                      `json:"encoderToken,omitempty"`
	Resolution    string                      `json:"resolution,omitempty"`
	Codec         string                      `json:"codec,omitempty"`
	ConfigCodec   string                      `json:"configCodec,omitempty"`
	StreamChannel string                      `json:"streamChannel,omitempty"`
	StreamType    string                      `json:"streamType,omitempty"`
	HasPTZ        bool                        `json:"hasPtz"`
	StreamURI     string                      `json:"streamUri,omitempty"`
	Profiles      []domainonvif.StreamProfile `json:"profiles,omitempty"`
}

type OnvifDeviceRepository struct {
	db *gorm.DB
}

func NewOnvifDeviceRepository(db *gorm.DB) *OnvifDeviceRepository {
	return &OnvifDeviceRepository{db: db}
}

func (r *OnvifDeviceRepository) Create(ctx context.Context, device *domainonvif.Device) error {
	if strings.TrimSpace(device.InternalCode) == "" {
		code, err := idcode.Device()
		if err != nil {
			return err
		}
		device.InternalCode = code
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	m := &model.Device{
		InternalCode: device.InternalCode, GbCode: device.GbCode, Name: device.Name,
		DeviceType: device.DeviceType, Manufacturer: device.Manufacturer, Model: device.Model, Firmware: device.Firmware,
		SerialNumber: device.SerialNumber, Online: device.OnLine, Enabled: true,
		Status: onlineStatus(device.OnLine), CreateTime: now, UpdateTime: now,
	}
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	cfg, _ := json.Marshal(onvifDeviceConfig{
		Username: device.Username, Password: device.Password, HardwareID: device.HardwareID,
		DeviceURI: device.DeviceURI, MediaURI: device.MediaURI, PTZURI: device.PTZURI,
		EventSubscribe: device.EventSubscribe, DiscoveryMode: device.DiscoveryMode,
		CustomName: device.CustomName, ServerID: device.ServerID,
	})
	if err := r.db.WithContext(ctx).Create(&model.AccessBinding{
		DeviceID: m.ID, Protocol: "onvif", Mode: "active",
		ExternalID: deviceURIOrEndpoint(device), Endpoint: endpointOf(device.IP, device.Port),
		ConfigJSON: string(cfg), Status: onlineStatus(device.OnLine), CreateTime: now, UpdateTime: now,
	}).Error; err != nil {
		return err
	}
	device.ID = m.ID
	device.CreateTime, device.UpdateTime = now, now
	return nil
}

func (r *OnvifDeviceRepository) Update(ctx context.Context, device *domainonvif.Device) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	if err := r.db.WithContext(ctx).Model(&model.Device{}).Where("id = ?", device.ID).Updates(map[string]any{
		"gb_code": device.GbCode, "name": device.Name, "manufacturer": device.Manufacturer,
		"model": device.Model, "firmware": device.Firmware, "serial_number": device.SerialNumber,
		"online": device.OnLine, "status": onlineStatus(device.OnLine), "update_time": now,
	}).Error; err != nil {
		return err
	}
	cfg, _ := json.Marshal(onvifDeviceConfig{
		Username: device.Username, Password: device.Password, HardwareID: device.HardwareID,
		DeviceURI: device.DeviceURI, MediaURI: device.MediaURI, PTZURI: device.PTZURI,
		EventSubscribe: device.EventSubscribe, DiscoveryMode: device.DiscoveryMode,
		CustomName: device.CustomName, ServerID: device.ServerID,
	})
	return r.db.WithContext(ctx).Model(&model.AccessBinding{}).Where("device_id = ?", device.ID).
		Updates(map[string]any{"config_json": string(cfg), "endpoint": endpointOf(device.IP, device.Port), "update_time": now}).Error
}

func (r *OnvifDeviceRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Device{}, id).Error
}

func (r *OnvifDeviceRepository) GetByID(ctx context.Context, id int64) (*domainonvif.Device, error) {
	var m model.Device
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return r.hydrateOnvifDevice(&m), nil
}

func (r *OnvifDeviceRepository) List(ctx context.Context, page, count int, keyword string) ([]*domainonvif.Device, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.WithContext(ctx).Model(&model.Device{}).
		Where("id IN (SELECT device_id FROM zws_access_binding WHERE protocol = 'onvif')")
	if keyword != "" {
		like := fmt.Sprintf("%%%s%%", keyword)
		q = q.Where("name LIKE ? OR internal_code LIKE ? OR gb_code LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Device
	if err := q.Order("id DESC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	devices := make([]*domainonvif.Device, 0, len(rows))
	for i := range rows {
		devices = append(devices, r.hydrateOnvifDevice(&rows[i]))
	}
	return devices, total, nil
}

func (r *OnvifDeviceRepository) ExistsByIPPort(ctx context.Context, ip string, port int) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AccessBinding{}).
		Where("protocol = 'onvif' AND endpoint = ?", endpointOf(ip, port)).Count(&count).Error
	return count > 0, err
}

func (r *OnvifDeviceRepository) UpdateOnlineStatus(ctx context.Context, id int64, online bool) error {
	return r.db.WithContext(ctx).Model(&model.Device{}).Where("id = ?", id).
		Updates(map[string]any{"online": online, "status": onlineStatus(online), "update_time": nowStr()}).Error
}

func (r *OnvifDeviceRepository) hydrateOnvifDevice(m *model.Device) *domainonvif.Device {
	d := &domainonvif.Device{
		ID: m.ID, InternalCode: m.InternalCode, GbCode: m.GbCode, Name: m.Name,
		DeviceType: m.DeviceType, Manufacturer: m.Manufacturer, Model: m.Model, Firmware: m.Firmware,
		SerialNumber: m.SerialNumber, OnLine: m.Online,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
	var b model.AccessBinding
	if err := r.db.Where("device_id = ?", m.ID).First(&b).Error; err == nil {
		var cfg onvifDeviceConfig
		if strings.TrimSpace(b.ConfigJSON) != "" {
			_ = json.Unmarshal([]byte(b.ConfigJSON), &cfg)
		}
		d.Username = cfg.Username
		d.Password = cfg.Password
		d.HardwareID = cfg.HardwareID
		d.DeviceURI = cfg.DeviceURI
		d.MediaURI = cfg.MediaURI
		d.PTZURI = cfg.PTZURI
		d.EventSubscribe = cfg.EventSubscribe
		d.DiscoveryMode = cfg.DiscoveryMode
		d.CustomName = cfg.CustomName
		d.ServerID = cfg.ServerID
		if b.Endpoint != "" {
			ip, port := splitEndpoint(b.Endpoint)
			d.IP, d.Port = ip, port
		}
	}
	return d
}

func deviceURIOrEndpoint(d *domainonvif.Device) string {
	if strings.TrimSpace(d.DeviceURI) != "" {
		return d.DeviceURI
	}
	return endpointOf(d.IP, d.Port)
}

func splitEndpoint(ep string) (string, int) {
	ep = strings.TrimSpace(ep)
	if ep == "" {
		return "", 0
	}
	i := strings.LastIndex(ep, ":")
	if i < 0 {
		return ep, 0
	}
	var port int
	fmt.Sscanf(ep[i+1:], "%d", &port)
	return ep[:i], port
}

// ===========================================================================
// OnvifChannelRepository
// ===========================================================================

type OnvifChannelRepository struct {
	db *gorm.DB
}

func NewOnvifChannelRepository(db *gorm.DB) *OnvifChannelRepository {
	return &OnvifChannelRepository{db: db}
}

func (r *OnvifChannelRepository) DeleteByDeviceID(ctx context.Context, deviceID int64) error {
	return r.db.WithContext(ctx).Where("device_id = ?", deviceID).Delete(&model.Channel{}).Error
}

func (r *OnvifChannelRepository) BatchCreate(ctx context.Context, channels []*domainonvif.Channel) error {
	if len(channels) == 0 {
		return nil
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, ch := range channels {
		if strings.TrimSpace(ch.InternalCode) == "" {
			code, err := idcode.Channel()
			if err != nil {
				return err
			}
			ch.InternalCode = code
		}
		m := &model.Channel{
			DeviceID: ch.DeviceID, InternalCode: ch.InternalCode, GbCode: ch.GbCode,
			ExternalKey: ch.ProfileToken, Kind: "video_sensor", Name: ch.Name,
			Online: ch.Status == "ON", Status: ch.Status, CreateTime: now, UpdateTime: now,
		}
		if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
			return err
		}
		ch.ID = m.ID
		ch.CreateTime, ch.UpdateTime = now, now
		cfg, _ := json.Marshal(onvifChannelConfig{
			ProfileToken: ch.ProfileToken, VideoSource: ch.VideoSource, EncoderToken: ch.EncoderToken,
			Resolution: ch.Resolution, Codec: ch.Codec, ConfigCodec: ch.ConfigCodec,
			StreamChannel: ch.StreamChannel, StreamType: ch.StreamType, HasPTZ: ch.HasPTZ,
			StreamURI: ch.StreamURI, Profiles: ch.StreamProfiles,
		})
		if err := r.db.WithContext(ctx).Create(&model.ChannelExt{
			ChannelID: m.ID, HasAudio: ch.HasAudio, Enabled: true, ExtraJSON: string(cfg),
			CreateTime: now, UpdateTime: now,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *OnvifChannelRepository) Update(ctx context.Context, channel *domainonvif.Channel) error {
	if channel == nil || channel.ID == 0 {
		return fmt.Errorf("invalid channel")
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	if err := r.db.WithContext(ctx).Model(&model.Channel{}).Where("id = ?", channel.ID).Updates(map[string]any{
		"name": channel.Name, "gb_code": channel.GbCode, "external_key": channel.ProfileToken,
		"status": channel.Status, "online": channel.Status == "ON", "update_time": now,
	}).Error; err != nil {
		return err
	}
	cfg, _ := json.Marshal(onvifChannelConfig{
		ProfileToken: channel.ProfileToken, VideoSource: channel.VideoSource, EncoderToken: channel.EncoderToken,
		Resolution: channel.Resolution, Codec: channel.Codec, ConfigCodec: channel.ConfigCodec,
		StreamChannel: channel.StreamChannel, StreamType: channel.StreamType, HasPTZ: channel.HasPTZ,
		StreamURI: channel.StreamURI, Profiles: channel.StreamProfiles,
	})
	return r.db.WithContext(ctx).Model(&model.ChannelExt{}).Where("channel_id = ?", channel.ID).
		Updates(map[string]any{"has_audio": channel.HasAudio, "extra_json": string(cfg), "update_time": now}).Error
}

func (r *OnvifChannelRepository) ListByDeviceID(ctx context.Context, deviceID int64) ([]*domainonvif.Channel, error) {
	var rows []model.Channel
	if err := r.db.WithContext(ctx).Where("device_id = ?", deviceID).Find(&rows).Error; err != nil {
		return nil, err
	}
	channels := make([]*domainonvif.Channel, 0, len(rows))
	for i := range rows {
		channels = append(channels, r.hydrateOnvifChannel(&rows[i]))
	}
	return channels, nil
}

func (r *OnvifChannelRepository) GetByID(ctx context.Context, id int64) (*domainonvif.Channel, error) {
	var m model.Channel
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return r.hydrateOnvifChannel(&m), nil
}

func (r *OnvifChannelRepository) List(ctx context.Context, page, count int, deviceID int64) ([]*domainonvif.Channel, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.WithContext(ctx).Model(&model.Channel{})
	if deviceID > 0 {
		q = q.Where("device_id = ?", deviceID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Channel
	if err := q.Order("id DESC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	channels := make([]*domainonvif.Channel, 0, len(rows))
	for i := range rows {
		channels = append(channels, r.hydrateOnvifChannel(&rows[i]))
	}
	return channels, total, nil
}

func (r *OnvifChannelRepository) hydrateOnvifChannel(m *model.Channel) *domainonvif.Channel {
	ch := &domainonvif.Channel{
		ID: m.ID, InternalCode: m.InternalCode, GbCode: m.GbCode, DeviceID: m.DeviceID,
		ProfileToken: m.ExternalKey, Name: m.Name, Status: m.Status,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
	var ext model.ChannelExt
	if err := r.db.Where("channel_id = ?", m.ID).First(&ext).Error; err == nil {
		ch.HasAudio = ext.HasAudio
		if strings.TrimSpace(ext.ExtraJSON) != "" {
			var cfg onvifChannelConfig
			if json.Unmarshal([]byte(ext.ExtraJSON), &cfg) == nil {
				ch.ProfileToken = firstNonEmptyStr(cfg.ProfileToken, ch.ProfileToken)
				ch.VideoSource = cfg.VideoSource
				ch.EncoderToken = cfg.EncoderToken
				ch.Resolution = cfg.Resolution
				ch.Codec = cfg.Codec
				ch.ConfigCodec = cfg.ConfigCodec
				ch.StreamChannel = cfg.StreamChannel
				ch.StreamType = cfg.StreamType
				ch.HasPTZ = cfg.HasPTZ
				ch.StreamURI = cfg.StreamURI
				ch.StreamProfiles = cfg.Profiles
			}
		}
	}
	return ch
}

func nowStr() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
