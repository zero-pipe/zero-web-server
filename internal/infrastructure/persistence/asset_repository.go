package persistence

import (
	"fmt"
	"time"

	domainasset "zero-web-server/internal/domain/asset"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type AssetRepository struct{ db *gorm.DB }

func NewAssetRepository(db *gorm.DB) *AssetRepository { return &AssetRepository{db: db} }

var _ domainasset.Repository = (*AssetRepository)(nil)

func nowText() string { return time.Now().Format("2006-01-02 15:04:05") }

func (r *AssetRepository) CreateDevice(d *domainasset.PhysicalDevice) error {
	now := nowText()
	m := &model.PhysicalDevice{
		ID: d.ID, InternalCode: d.InternalCode, Name: d.Name, Vendor: d.Vendor,
		Model: d.Model, SerialNumber: d.SerialNumber, Enabled: d.Enabled, Online: d.Online,
		CreateTime: now, UpdateTime: now,
	}
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	d.CreateTime, d.UpdateTime = now, now
	return nil
}

func (r *AssetRepository) UpsertDevice(d *domainasset.PhysicalDevice) error {
	now := nowText()
	return r.db.Save(&model.PhysicalDevice{
		ID: d.ID, InternalCode: d.InternalCode, Name: d.Name, Vendor: d.Vendor,
		Model: d.Model, SerialNumber: d.SerialNumber, Enabled: d.Enabled, Online: d.Online,
		CreateTime: d.CreateTime, UpdateTime: now,
	}).Error
}

func (r *AssetRepository) GetDevice(id string) (*domainasset.PhysicalDevice, error) {
	var m model.PhysicalDevice
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainAssetDevice(&m), nil
}

func (r *AssetRepository) GetDeviceByInternalCode(code string) (*domainasset.PhysicalDevice, error) {
	var m model.PhysicalDevice
	if err := r.db.Where("internal_code = ?", code).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainAssetDevice(&m), nil
}

func (r *AssetRepository) ListDevices(page, count int, query, protocol, status string) ([]*domainasset.DeviceSummary, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Table("zws_physical_device d").
		Joins("LEFT JOIN zws_access_binding b ON b.device_id = d.id")
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("d.name LIKE ? OR d.internal_code LIKE ? OR d.serial_number LIKE ?", like, like, like)
	}
	if protocol != "" {
		q = q.Where("b.protocol = ?", protocol)
	}
	switch status {
	case "online":
		q = q.Where("d.online = ?", true)
	case "offline", "pending":
		q = q.Where("d.online = ?", false)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		model.PhysicalDevice
		BID              string `gorm:"column:b_id"`
		Protocol         string `gorm:"column:b_protocol"`
		Mode             string `gorm:"column:b_mode"`
		ExternalID       string `gorm:"column:b_external_id"`
		Endpoint         string `gorm:"column:b_endpoint"`
		PluginInstanceID string `gorm:"column:b_plugin_instance_id"`
		ConfigJSON       string `gorm:"column:b_config_json"`
		Status           string `gorm:"column:b_status"`
		LastSeen         int64  `gorm:"column:b_last_seen"`
		LegacyDeviceID   int    `gorm:"column:b_legacy_device_id"`
		BCreateTime      string `gorm:"column:b_create_time"`
		BUpdateTime      string `gorm:"column:b_update_time"`
	}
	var rows []row
	err := q.Select(`d.*, b.id AS b_id, b.protocol AS b_protocol, b.mode AS b_mode,
		b.external_id AS b_external_id, b.endpoint AS b_endpoint,
		b.plugin_instance_id AS b_plugin_instance_id, b.config_json AS b_config_json,
		b.status AS b_status, b.last_seen AS b_last_seen, b.legacy_device_id AS b_legacy_device_id,
		b.create_time AS b_create_time, b.update_time AS b_update_time`).
		Order("d.update_time DESC").
		Offset((page - 1) * count).Limit(count).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	counts := map[string]int{}
	var res []struct {
		DeviceID string `gorm:"column:device_id"`
		N        int    `gorm:"column:n"`
	}
	_ = r.db.Table("zws_resource").Select("device_id, COUNT(*) AS n").Group("device_id").Scan(&res).Error
	for _, c := range res {
		counts[c.DeviceID] = c.N
	}
	out := make([]*domainasset.DeviceSummary, 0, len(rows))
	for i := range rows {
		b := &domainasset.AccessBinding{
			ID: rows[i].BID, DeviceID: rows[i].ID, Protocol: rows[i].Protocol, Mode: rows[i].Mode,
			ExternalID: rows[i].ExternalID, Endpoint: rows[i].Endpoint,
			PluginInstanceID: rows[i].PluginInstanceID, ConfigJSON: rows[i].ConfigJSON,
			Status: rows[i].Status, LastSeen: rows[i].LastSeen, LegacyDeviceID: rows[i].LegacyDeviceID,
			CreateTime: rows[i].BCreateTime, UpdateTime: rows[i].BUpdateTime,
		}
		if b.ID == "" {
			b = nil
		}
		out = append(out, &domainasset.DeviceSummary{
			Device:        toDomainAssetDevice(&rows[i].PhysicalDevice),
			Binding:       b,
			ResourceCount: counts[rows[i].ID],
		})
	}
	return out, total, nil
}

func (r *AssetRepository) CreateBinding(b *domainasset.AccessBinding) error {
	now := nowText()
	if err := r.db.Create(&model.AccessBinding{
		ID: b.ID, DeviceID: b.DeviceID, Protocol: b.Protocol, Mode: b.Mode,
		ExternalID: b.ExternalID, Endpoint: b.Endpoint, PluginInstanceID: b.PluginInstanceID,
		ConfigJSON: b.ConfigJSON, Status: b.Status, LastSeen: b.LastSeen,
		LegacyDeviceID: b.LegacyDeviceID, CreateTime: now, UpdateTime: now,
	}).Error; err != nil {
		return err
	}
	b.CreateTime, b.UpdateTime = now, now
	return nil
}

func (r *AssetRepository) UpsertBinding(b *domainasset.AccessBinding) error {
	now := nowText()
	return r.db.Save(&model.AccessBinding{
		ID: b.ID, DeviceID: b.DeviceID, Protocol: b.Protocol, Mode: b.Mode,
		ExternalID: b.ExternalID, Endpoint: b.Endpoint, PluginInstanceID: b.PluginInstanceID,
		ConfigJSON: b.ConfigJSON, Status: b.Status, LastSeen: b.LastSeen,
		LegacyDeviceID: b.LegacyDeviceID, CreateTime: b.CreateTime, UpdateTime: now,
	}).Error
}

func (r *AssetRepository) GetBindingByDevice(deviceID string) (*domainasset.AccessBinding, error) {
	var m model.AccessBinding
	if err := r.db.Where("device_id = ?", deviceID).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainAssetBinding(&m), nil
}

func (r *AssetRepository) UpsertResource(res *domainasset.Resource) error {
	now := nowText()
	return r.db.Save(&model.AssetResource{
		ID: res.ID, DeviceID: res.DeviceID, BindingID: res.BindingID, Kind: res.Kind,
		ExternalKey: res.ExternalKey, Name: res.Name, Status: res.Status,
		LegacyChannelID: res.LegacyChannelID, AttrsJSON: res.AttrsJSON,
		CreateTime: res.CreateTime, UpdateTime: now,
	}).Error
}

func (r *AssetRepository) ListResources(deviceID string) ([]*domainasset.Resource, error) {
	var rows []model.AssetResource
	if err := r.db.Where("device_id = ?", deviceID).Order("kind, name").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainasset.Resource, 0, len(rows))
	for i := range rows {
		out = append(out, toDomainAssetResource(&rows[i]))
	}
	return out, nil
}

func (r *AssetRepository) GetResource(id string) (*domainasset.Resource, error) {
	var m model.AssetResource
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainAssetResource(&m), nil
}

func (r *AssetRepository) UpsertProfile(p *domainasset.StreamProfile) error {
	now := nowText()
	return r.db.Save(&model.AssetStreamProfile{
		ID: p.ID, ResourceID: p.ResourceID, Role: p.Role, Name: p.Name,
		SelectorJSON: p.SelectorJSON, Codec: p.Codec, Resolution: p.Resolution,
		IsDefault: p.IsDefault, CreateTime: p.CreateTime, UpdateTime: now,
	}).Error
}

func (r *AssetRepository) ListProfiles(resourceID string) ([]*domainasset.StreamProfile, error) {
	var rows []model.AssetStreamProfile
	if err := r.db.Where("resource_id = ?", resourceID).Order("is_default DESC, role").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainasset.StreamProfile, 0, len(rows))
	for i := range rows {
		out = append(out, toDomainAssetProfile(&rows[i]))
	}
	return out, nil
}

func (r *AssetRepository) GetProfile(id string) (*domainasset.StreamProfile, error) {
	var m model.AssetStreamProfile
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainAssetProfile(&m), nil
}

func (r *AssetRepository) ReplaceCapabilities(scopeType, scopeID string, caps []*domainasset.Capability) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("scope_type = ? AND scope_id = ?", scopeType, scopeID).Delete(&model.AssetCapability{}).Error; err != nil {
			return err
		}
		now := nowText()
		for _, c := range caps {
			if err := tx.Create(&model.AssetCapability{
				ID: c.ID, ScopeType: c.ScopeType, ScopeID: c.ScopeID, Operation: c.Operation,
				Supported: c.Supported, Configured: c.Configured, Available: c.Available,
				ParamsJSON: c.ParamsJSON, Source: c.Source, UpdateTime: now,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *AssetRepository) ListCapabilities(scopeType, scopeID string) ([]*domainasset.Capability, error) {
	var rows []model.AssetCapability
	if err := r.db.Where("scope_type = ? AND scope_id = ?", scopeType, scopeID).Order("operation").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainasset.Capability, 0, len(rows))
	for i := range rows {
		out = append(out, &domainasset.Capability{
			ID: rows[i].ID, ScopeType: rows[i].ScopeType, ScopeID: rows[i].ScopeID,
			Operation: rows[i].Operation, Supported: rows[i].Supported,
			Configured: rows[i].Configured, Available: rows[i].Available,
			ParamsJSON: rows[i].ParamsJSON, Source: rows[i].Source, UpdateTime: rows[i].UpdateTime,
		})
	}
	return out, nil
}

func (r *AssetRepository) UpsertEventSource(e *domainasset.EventSource) error {
	return r.db.Save(&model.AssetEventSource{
		ID: e.ID, DeviceID: e.DeviceID, ResourceID: e.ResourceID, Kind: e.Kind,
		ExternalKey: e.ExternalKey, Enabled: e.Enabled, ConfigJSON: e.ConfigJSON,
	}).Error
}

func (r *AssetRepository) ListEventSources(deviceID string) ([]*domainasset.EventSource, error) {
	var rows []model.AssetEventSource
	if err := r.db.Where("device_id = ?", deviceID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainasset.EventSource, 0, len(rows))
	for i := range rows {
		out = append(out, &domainasset.EventSource{
			ID: rows[i].ID, DeviceID: rows[i].DeviceID, ResourceID: rows[i].ResourceID,
			Kind: rows[i].Kind, ExternalKey: rows[i].ExternalKey,
			Enabled: rows[i].Enabled, ConfigJSON: rows[i].ConfigJSON,
		})
	}
	return out, nil
}

func (r *AssetRepository) GetLegacyRef(table, legacyID string) (string, string, error) {
	var m model.UnifiedRef
	if err := r.db.Where("legacy_table = ? AND legacy_id = ?", table, legacyID).First(&m).Error; err != nil {
		return "", "", err
	}
	return m.EntityType, m.UnifiedID, nil
}

func (r *AssetRepository) UpsertLegacyRef(table, legacyID, entityType, unifiedID string) error {
	var m model.UnifiedRef
	err := r.db.Where("legacy_table = ? AND legacy_id = ?", table, legacyID).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(&model.UnifiedRef{
			LegacyTable: table, LegacyID: legacyID, EntityType: entityType, UnifiedID: unifiedID,
		}).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&model.UnifiedRef{}).Where("id = ?", m.ID).
		Updates(map[string]any{"entity_type": entityType, "unified_id": unifiedID}).Error
}

func toDomainAssetDevice(m *model.PhysicalDevice) *domainasset.PhysicalDevice {
	return &domainasset.PhysicalDevice{
		ID: m.ID, InternalCode: m.InternalCode, Name: m.Name, Vendor: m.Vendor,
		Model: m.Model, SerialNumber: m.SerialNumber, Enabled: m.Enabled, Online: m.Online,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func toDomainAssetBinding(m *model.AccessBinding) *domainasset.AccessBinding {
	return &domainasset.AccessBinding{
		ID: m.ID, DeviceID: m.DeviceID, Protocol: m.Protocol, Mode: m.Mode,
		ExternalID: m.ExternalID, Endpoint: m.Endpoint, PluginInstanceID: m.PluginInstanceID,
		ConfigJSON: m.ConfigJSON, Status: m.Status, LastSeen: m.LastSeen,
		LegacyDeviceID: m.LegacyDeviceID, CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func toDomainAssetResource(m *model.AssetResource) *domainasset.Resource {
	return &domainasset.Resource{
		ID: m.ID, DeviceID: m.DeviceID, BindingID: m.BindingID, Kind: m.Kind,
		ExternalKey: m.ExternalKey, Name: m.Name, Status: m.Status,
		LegacyChannelID: m.LegacyChannelID, AttrsJSON: m.AttrsJSON,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func toDomainAssetProfile(m *model.AssetStreamProfile) *domainasset.StreamProfile {
	return &domainasset.StreamProfile{
		ID: m.ID, ResourceID: m.ResourceID, Role: m.Role, Name: m.Name,
		SelectorJSON: m.SelectorJSON, Codec: m.Codec, Resolution: m.Resolution,
		IsDefault: m.IsDefault, CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}
