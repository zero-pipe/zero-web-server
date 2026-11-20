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
		if err := r.db.Create(m).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if err := r.db.Model(&model.Plugin{}).Where("id = ?", p.ID).Updates(map[string]any{
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
		}).Error; err != nil {
			return err
		}
	}
	return nil
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
		return tx.Model(&model.PluginDevice{}).Where("plugin_id = ?", id).Update("online", false).Error
	})
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

func (r *PluginRepository) CreateDevice(d *domainplugin.Device) error {
	if strings.TrimSpace(d.InternalCode) == "" {
		code, err := idcode.Device()
		if err != nil {
			return err
		}
		d.InternalCode = code
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	m := &model.PluginDevice{
		InternalCode: d.InternalCode, PluginID: d.PluginID, Protocol: d.Protocol,
		DeviceUID: d.DeviceUID, Name: d.Name, Enabled: true, Online: d.Online,
		DriverParams: d.DriverParams, Manufacturer: d.Manufacturer, Model: d.Model,
		Firmware: d.Firmware, SerialNumber: d.SerialNumber,
		CreateTime: now, UpdateTime: now,
	}
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	d.ID = m.ID
	d.CreateTime, d.UpdateTime = now, now
	return nil
}

func (r *PluginRepository) UpdateDevice(d *domainplugin.Device) error {
	return r.db.Model(&model.PluginDevice{}).Where("id = ?", d.ID).Updates(map[string]any{
		"name":          d.Name,
		"enabled":       d.Enabled,
		"online":        d.Online,
		"driver_params": d.DriverParams,
		"manufacturer":  d.Manufacturer,
		"model":         d.Model,
		"firmware":      d.Firmware,
		"serial_number": d.SerialNumber,
		"update_time":   time.Now().Format("2006-01-02 15:04:05"),
	}).Error
}

func (r *PluginRepository) DeleteDevice(id int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("device_db_id = ?", id).Delete(&model.PluginChannel{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.PluginDevice{}, id).Error
	})
}

// CountDevicesByPlugin 统计某插件（按清单 plugin_id，如 hikvision-ipc）接入的设备数。
// 设备表存的是运行时实例 ID（agentId-pluginId），故按运行时实例反查。
func (r *PluginRepository) CountDevicesByPlugin(pluginID string) (int, error) {
	var ids []string
	if err := r.db.Model(&model.Plugin{}).Where("plugin_id = ?", pluginID).Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	var n int64
	if err := r.db.Model(&model.PluginDevice{}).Where("plugin_id IN ?", ids).Count(&n).Error; err != nil {
		return 0, err
	}
	return int(n), nil
}

func (r *PluginRepository) ListDevices() ([]*domainplugin.Device, error) {
	var rows []model.PluginDevice
	if err := r.db.Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	counts := map[int]int{}
	var chs []model.PluginChannel
	_ = r.db.Find(&chs).Error
	for i := range chs {
		counts[chs[i].DeviceDBID]++
	}
	out := make([]*domainplugin.Device, 0, len(rows))
	for i := range rows {
		d := toDomainPluginDevice(&rows[i])
		d.ChannelCount = counts[d.ID]
		out = append(out, d)
	}
	return out, nil
}

func (r *PluginRepository) GetDevice(id int) (*domainplugin.Device, error) {
	var row model.PluginDevice
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return toDomainPluginDevice(&row), nil
}

func (r *PluginRepository) GetDeviceByUID(pluginID, deviceUID string) (*domainplugin.Device, error) {
	var row model.PluginDevice
	if err := r.db.Where("plugin_id = ? AND device_uid = ?", pluginID, deviceUID).First(&row).Error; err != nil {
		return nil, err
	}
	return toDomainPluginDevice(&row), nil
}

func (r *PluginRepository) ReplaceChannels(deviceDBID int, channels []*domainplugin.Channel) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing []model.PluginChannel
		_ = tx.Where("device_db_id = ?", deviceDBID).Find(&existing).Error
		keep := map[string]string{}
		keepName := map[string]string{}
		for i := range existing {
			keep[existing[i].ChannelID] = existing[i].InternalCode
			if strings.TrimSpace(existing[i].Name) != "" {
				keepName[existing[i].ChannelID] = existing[i].Name
			}
		}
		var dev model.PluginDevice
		_ = tx.First(&dev, deviceDBID).Error
		if err := tx.Where("device_db_id = ?", deviceDBID).Delete(&model.PluginChannel{}).Error; err != nil {
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
			if err := tx.Create(&model.PluginChannel{
				InternalCode: code, DeviceDBID: deviceDBID, ChannelID: ch.ChannelID,
				Name: name, Status: ch.Status, PTZType: ch.PTZType, HasAudio: ch.HasAudio,
				CreateTime: now, UpdateTime: now,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PluginRepository) ListChannels(deviceDBID int) ([]*domainplugin.Channel, error) {
	var rows []model.PluginChannel
	if err := r.db.Where("device_db_id = ?", deviceDBID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainplugin.Channel, 0, len(rows))
	for i := range rows {
		out = append(out, toDomainPluginChannel(&rows[i]))
	}
	return out, nil
}

func (r *PluginRepository) GetChannel(deviceDBID int, channelID string) (*domainplugin.Channel, error) {
	var row model.PluginChannel
	if err := r.db.Where("device_db_id = ? AND channel_id = ?", deviceDBID, channelID).First(&row).Error; err != nil {
		return nil, err
	}
	return toDomainPluginChannel(&row), nil
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

func toDomainPluginDevice(m *model.PluginDevice) *domainplugin.Device {
	return &domainplugin.Device{
		ID: m.ID, InternalCode: m.InternalCode, PluginID: m.PluginID, Protocol: m.Protocol,
		DeviceUID: m.DeviceUID, Name: m.Name, Enabled: m.Enabled, Online: m.Online,
		DriverParams: m.DriverParams, Manufacturer: m.Manufacturer, Model: m.Model,
		Firmware: m.Firmware, SerialNumber: m.SerialNumber,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func toDomainPluginChannel(m *model.PluginChannel) *domainplugin.Channel {
	return &domainplugin.Channel{
		ID: m.ID, InternalCode: m.InternalCode, DeviceDBID: m.DeviceDBID,
		ChannelID: m.ChannelID, Name: m.Name, Status: m.Status,
		PTZType: m.PTZType, HasAudio: m.HasAudio,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
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
