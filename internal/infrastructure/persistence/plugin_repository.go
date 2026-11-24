package persistence

import (
	"encoding/json"
	"strings"
	"time"

	domainplugin "zero-web-server/internal/domain/plugin"
	"zero-web-server/internal/domain/shared"
	"zero-web-server/internal/infrastructure/persistence/model"
	"zero-web-server/pkg/idcode"

	"gorm.io/gorm"
)

// PluginRepository 插件接入持久化。
// 插件设备 → zws_device + zws_access_binding(protocol=插件协议, plugin_instance_id=运行时ID,
// external_id=DeviceUID, config_json=DriverParams)；插件通道 → zws_channel + zws_channel_ext。
type PluginRepository struct{ db *gorm.DB }

func NewPluginRepository(db *gorm.DB) *PluginRepository { return &PluginRepository{db: db} }

var _ domainplugin.Repository = (*PluginRepository)(nil)

func (r *PluginRepository) UpsertPlugin(p *domainplugin.Plugin) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	protocols, _ := json.Marshal(p.Protocols)
	caps, _ := json.Marshal(p.Capabilities)
	transports, _ := json.Marshal(p.StreamTransports)
	var existing model.Plugin
	err := r.db.Where("id = ?", p.ID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		m := &model.Plugin{
			ID: p.ID, PluginID: p.PluginID, Vendor: p.Vendor, Version: p.Version,
			Protocols: string(protocols), Capabilities: string(caps), StreamTransports: string(transports),
			ConfigSchema: p.ConfigSchema, Endpoint: p.Endpoint, Online: true,
			LastSeen: time.Now().UnixMilli(), CreateTime: now, UpdateTime: now,
		}
		return r.db.Create(m).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&model.Plugin{}).Where("id = ?", p.ID).Updates(map[string]any{
		"plugin_id":         p.PluginID,
		"vendor":            p.Vendor,
		"version":           p.Version,
		"protocols":         string(protocols),
		"capabilities":      string(caps),
		"stream_transports": string(transports),
		"config_schema":     p.ConfigSchema,
		"endpoint":          p.Endpoint,
		"online":            true,
		"last_seen":         time.Now().UnixMilli(),
		"update_time":       now,
	}).Error
}

func (r *PluginRepository) UpdatePluginSeen(id string, ts int64, online bool) error {
	return r.db.Model(&model.Plugin{}).Where("id = ?", id).
		Updates(map[string]any{"last_seen": ts, "online": online}).Error
}

func (r *PluginRepository) MarkPluginOffline(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Plugin{}).Where("id = ?", id).Update("online", false).Error; err != nil {
			return err
		}
		// 该插件实例下的设备置离线
		return tx.Model(&model.Device{}).
			Where("id IN (SELECT device_id FROM zws_access_binding WHERE plugin_instance_id = ?)", id).
			Update("online", false).Error
	})
}

func (r *PluginRepository) DeletePlugin(id string) error {
	return r.db.Where("id = ?", id).Delete(&model.Plugin{}).Error
}

func (r *PluginRepository) ListPlugins() ([]*domainplugin.Plugin, error) {
	var rows []model.Plugin
	if err := r.db.Order("update_time DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainplugin.Plugin, 0, len(rows))
	for i := range rows {
		out = append(out, toDomainPlugin(&rows[i]))
	}
	return out, nil
}

func (r *PluginRepository) GetPlugin(id string) (*domainplugin.Plugin, error) {
	var row model.Plugin
	if err := r.db.Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return toDomainPlugin(&row), nil
}

func (r *PluginRepository) ResolvePlugin(protocol string) (*domainplugin.Plugin, error) {
	var rows []model.Plugin
	if err := r.db.Where("online = ?", true).Order("update_time DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	protocol = strings.TrimSpace(protocol)
	for i := range rows {
		for _, p := range decodeStrings(rows[i].Protocols) {
			if p == protocol {
				return toDomainPlugin(&rows[i]), nil
			}
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// pluginDeviceRow 便于 join 出绑定的插件信息。
type pluginBinding struct {
	Protocol         string `gorm:"column:protocol"`
	ExternalID       string `gorm:"column:external_id"`
	PluginInstanceID string `gorm:"column:plugin_instance_id"`
	ConfigJSON       string `gorm:"column:config_json"`
}

func (r *PluginRepository) CreateDevice(d *domainplugin.Device) error {
	if strings.TrimSpace(d.InternalCode) == "" {
		code, err := idcode.Device()
		if err != nil {
			return err
		}
		d.InternalCode = code
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	m := &model.Device{
		InternalCode: d.InternalCode, Name: d.Name, Manufacturer: d.Manufacturer,
		Model: d.Model, Firmware: d.Firmware, SerialNumber: d.SerialNumber,
		Online: d.Online, Enabled: d.Enabled, Status: onlineStatus(d.Online),
		CreateTime: now, UpdateTime: now,
	}
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	cfg, _ := json.Marshal(map[string]string{"driverParams": d.DriverParams})
	if err := r.db.Create(&model.AccessBinding{
		DeviceID: m.ID, Protocol: d.Protocol, Mode: "active",
		ExternalID: d.DeviceUID, PluginInstanceID: d.PluginID,
		ConfigJSON: string(cfg), Status: onlineStatus(d.Online), CreateTime: now, UpdateTime: now,
	}).Error; err != nil {
		return err
	}
	d.ID = int(m.ID)
	d.CreateTime, d.UpdateTime = now, now
	return nil
}

func (r *PluginRepository) UpdateDevice(d *domainplugin.Device) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	if err := r.db.Model(&model.Device{}).Where("id = ?", d.ID).Updates(map[string]any{
		"name": d.Name, "enabled": d.Enabled, "online": d.Online,
		"manufacturer": d.Manufacturer, "model": d.Model, "firmware": d.Firmware,
		"serial_number": d.SerialNumber, "status": onlineStatus(d.Online), "update_time": now,
	}).Error; err != nil {
		return err
	}
	cfg, _ := json.Marshal(map[string]string{"driverParams": d.DriverParams})
	return r.db.Model(&model.AccessBinding{}).Where("device_id = ?", d.ID).
		Updates(map[string]any{"config_json": string(cfg), "update_time": now}).Error
}

func (r *PluginRepository) DeleteDevice(id int) error {
	// 绑定/通道/扩展通过外键级联删除。
	return r.db.Delete(&model.Device{}, id).Error
}

func (r *PluginRepository) CountDevicesByPlugin(pluginID string) (int, error) {
	var ids []string
	if err := r.db.Model(&model.Plugin{}).Where("plugin_id = ?", pluginID).Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	var n int64
	err := r.db.Model(&model.AccessBinding{}).Where("plugin_instance_id IN ?", ids).Count(&n).Error
	return int(n), err
}

func (r *PluginRepository) ListDevices() ([]*domainplugin.Device, error) {
	var rows []model.Device
	if err := r.db.Where("id IN (SELECT device_id FROM zws_access_binding WHERE plugin_instance_id <> '')").
		Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	counts := map[int64]int{}
	var agg []struct {
		DeviceID int64 `gorm:"column:device_id"`
		N        int   `gorm:"column:n"`
	}
	_ = r.db.Table("zws_channel").Select("device_id, COUNT(*) AS n").Group("device_id").Scan(&agg).Error
	for _, c := range agg {
		counts[c.DeviceID] = c.N
	}
	out := make([]*domainplugin.Device, 0, len(rows))
	for i := range rows {
		d := r.hydratePluginDevice(&rows[i])
		d.ChannelCount = counts[rows[i].ID]
		out = append(out, d)
	}
	return out, nil
}

func (r *PluginRepository) GetDevice(id int) (*domainplugin.Device, error) {
	var row model.Device
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return r.hydratePluginDevice(&row), nil
}

func (r *PluginRepository) GetDeviceByUID(pluginID, deviceUID string) (*domainplugin.Device, error) {
	var b model.AccessBinding
	if err := r.db.Where("plugin_instance_id = ? AND external_id = ?", pluginID, deviceUID).First(&b).Error; err != nil {
		return nil, err
	}
	var row model.Device
	if err := r.db.First(&row, b.DeviceID).Error; err != nil {
		return nil, err
	}
	return r.hydratePluginDevice(&row), nil
}

func (r *PluginRepository) hydratePluginDevice(m *model.Device) *domainplugin.Device {
	d := &domainplugin.Device{
		ID: int(m.ID), InternalCode: m.InternalCode, Name: m.Name, Enabled: m.Enabled,
		Online: m.Online, Manufacturer: m.Manufacturer, Model: m.Model,
		Firmware: m.Firmware, SerialNumber: m.SerialNumber,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
	var b model.AccessBinding
	if err := r.db.Where("device_id = ?", m.ID).First(&b).Error; err == nil {
		d.Protocol = b.Protocol
		d.PluginID = b.PluginInstanceID
		d.DeviceUID = b.ExternalID
		if strings.TrimSpace(b.ConfigJSON) != "" {
			var cfg struct {
				DriverParams string `json:"driverParams"`
			}
			if json.Unmarshal([]byte(b.ConfigJSON), &cfg) == nil {
				d.DriverParams = cfg.DriverParams
			}
		}
	}
	return d
}

func (r *PluginRepository) ReplaceChannels(deviceDBID int, channels []*domainplugin.Channel) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing []model.Channel
		_ = tx.Where("device_id = ?", deviceDBID).Find(&existing).Error
		keep := map[string]string{}
		keepName := map[string]string{}
		for i := range existing {
			key := strings.TrimSpace(existing[i].ExternalKey)
			keep[key] = existing[i].InternalCode
			if strings.TrimSpace(existing[i].Name) != "" {
				keepName[key] = existing[i].Name
			}
		}
		var dev model.Device
		_ = tx.First(&dev, deviceDBID).Error
		if err := tx.Where("device_id = ?", deviceDBID).Delete(&model.Channel{}).Error; err != nil {
			return err
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		for i, ch := range channels {
			code := ch.InternalCode
			if code == "" {
				code = keep[ch.ChannelID]
			}
			if code == "" {
				var err error
				code, err = idcode.Channel()
				if err != nil {
					return err
				}
			}
			name := keepName[ch.ChannelID]
			if strings.TrimSpace(name) == "" {
				name = shared.DefaultChannelName(dev.Name, i+1)
			}
			ch.ID, ch.InternalCode, ch.DeviceDBID = 0, code, deviceDBID
			ch.CreateTime, ch.UpdateTime = now, now
			m := &model.Channel{
				DeviceID: int64(deviceDBID), InternalCode: code, ExternalKey: ch.ChannelID,
				Kind: "video_sensor", Name: name, Online: ch.Status == "ON", Status: ch.Status,
				CreateTime: now, UpdateTime: now,
			}
			if err := tx.Create(m).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.ChannelExt{
				ChannelID: m.ID, PTZType: ch.PTZType, HasAudio: ch.HasAudio,
				Enabled: true, CreateTime: now, UpdateTime: now,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PluginRepository) ListChannels(deviceDBID int) ([]*domainplugin.Channel, error) {
	var rows []model.Channel
	if err := r.db.Where("device_id = ?", deviceDBID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainplugin.Channel, 0, len(rows))
	for i := range rows {
		out = append(out, r.hydratePluginChannel(&rows[i]))
	}
	return out, nil
}

func (r *PluginRepository) GetChannel(deviceDBID int, channelID string) (*domainplugin.Channel, error) {
	var row model.Channel
	if err := r.db.Where("device_id = ? AND external_key = ?", deviceDBID, channelID).First(&row).Error; err != nil {
		return nil, err
	}
	return r.hydratePluginChannel(&row), nil
}

func (r *PluginRepository) hydratePluginChannel(m *model.Channel) *domainplugin.Channel {
	ch := &domainplugin.Channel{
		ID: int(m.ID), InternalCode: m.InternalCode, DeviceDBID: int(m.DeviceID),
		ChannelID: m.ExternalKey, Name: m.Name, Status: m.Status,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
	var ext model.ChannelExt
	if err := r.db.Where("channel_id = ?", m.ID).First(&ext).Error; err == nil {
		ch.PTZType = ext.PTZType
		ch.HasAudio = ext.HasAudio
	}
	return ch
}

func toDomainPlugin(m *model.Plugin) *domainplugin.Plugin {
	return &domainplugin.Plugin{
		ID: m.ID, PluginID: m.PluginID, Vendor: m.Vendor, Version: m.Version,
		Protocols: decodeStrings(m.Protocols), Capabilities: decodeStrings(m.Capabilities),
		StreamTransports: decodeStrings(m.StreamTransports),
		ConfigSchema:     m.ConfigSchema, Endpoint: m.Endpoint, Online: m.Online,
		LastSeen: m.LastSeen, CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func decodeStrings(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}
