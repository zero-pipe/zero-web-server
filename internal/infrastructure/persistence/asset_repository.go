package persistence

import (
	"encoding/json"
	"fmt"
	"strings"
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
	case "offline":
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

func (r *AssetRepository) GetResourceByInternalCode(code string) (*domainasset.Resource, error) {
	var m model.AssetResource
	if err := r.db.Where("internal_code = ?", code).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainAssetResource(&m), nil
}

func (r *AssetRepository) UpdateDeviceFields(id string, patch domainasset.DevicePatch) error {
	fields := map[string]any{"update_time": nowText()}
	if patch.Name != nil {
		fields["name"] = strings.TrimSpace(*patch.Name)
	}
	if patch.Vendor != nil {
		fields["vendor"] = strings.TrimSpace(*patch.Vendor)
	}
	if patch.Model != nil {
		fields["model"] = strings.TrimSpace(*patch.Model)
	}
	if patch.SerialNumber != nil {
		fields["serial_number"] = strings.TrimSpace(*patch.SerialNumber)
	}
	if patch.Enabled != nil {
		fields["enabled"] = *patch.Enabled
	}
	return r.db.Model(&model.PhysicalDevice{}).Where("id = ?", id).Updates(fields).Error
}

func (r *AssetRepository) UpdateResourceFields(id string, patch domainasset.ResourcePatch) error {
	var m model.AssetResource
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		return err
	}
	attrs := map[string]any{}
	if strings.TrimSpace(m.AttrsJSON) != "" {
		_ = json.Unmarshal([]byte(m.AttrsJSON), &attrs)
	}
	edited := false
	if patch.Longitude != nil {
		attrs["longitude"] = *patch.Longitude
		edited = true
	}
	if patch.Latitude != nil {
		attrs["latitude"] = *patch.Latitude
		edited = true
	}
	if patch.MapLevel != nil {
		attrs["mapLevel"] = *patch.MapLevel
		edited = true
	}
	if patch.PTZType != nil {
		attrs["ptzType"] = *patch.PTZType
		edited = true
	}
	if patch.HasAudio != nil {
		attrs["hasAudio"] = *patch.HasAudio
		edited = true
	}
	if patch.Tags != nil {
		attrs["tags"] = strings.TrimSpace(*patch.Tags)
		edited = true
	}
	if patch.Enabled != nil {
		attrs["enabled"] = *patch.Enabled
		edited = true
	}
	if edited {
		attrs["userEdited"] = true
	}
	raw, err := json.Marshal(attrs)
	if err != nil {
		return err
	}
	updates := map[string]any{"attrs_json": string(raw), "update_time": nowText()}
	if patch.Name != nil {
		updates["name"] = strings.TrimSpace(*patch.Name)
	}
	return r.db.Model(&model.AssetResource{}).Where("id = ?", id).Updates(updates).Error
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

func (r *AssetRepository) ListChannels(page, count int, query, protocol, deviceInternalCode string) ([]*domainasset.ChannelView, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Table("zws_resource r").
		Joins("JOIN zws_physical_device d ON d.id = r.device_id").
		Joins("LEFT JOIN zws_access_binding b ON b.device_id = d.id")
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("r.name LIKE ? OR r.internal_code LIKE ? OR r.external_key LIKE ? OR d.name LIKE ?",
			like, like, like, like)
	}
	if protocol != "" {
		q = q.Where("b.protocol = ?", protocol)
	}
	if deviceInternalCode != "" {
		q = q.Where("d.internal_code = ?", deviceInternalCode)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type row struct {
		ResourceID         string  `gorm:"column:resource_id"`
		InternalCode       string  `gorm:"column:internal_code"`
		DeviceID           string  `gorm:"column:device_id"`
		DeviceInternalCode string  `gorm:"column:device_internal_code"`
		DeviceName         string  `gorm:"column:device_name"`
		Protocol           string  `gorm:"column:protocol"`
		Name               string  `gorm:"column:name"`
		Kind               string  `gorm:"column:kind"`
		ExternalKey        string  `gorm:"column:external_key"`
		Status             string  `gorm:"column:status"`
		Online             bool    `gorm:"column:online"`
		PTZType            int     `gorm:"column:ptz_type"`
		HasAudio           bool    `gorm:"column:has_audio"`
		Longitude          float64 `gorm:"column:longitude"`
		Latitude           float64 `gorm:"column:latitude"`
		MapLevel           int     `gorm:"column:map_level"`
		Tags               string  `gorm:"column:tags"`
		Enabled            int     `gorm:"column:enabled"`
	}
	var rows []row
	err := q.Select(`r.id AS resource_id, r.internal_code, r.device_id, d.internal_code AS device_internal_code,
			d.name AS device_name, b.protocol, r.name, r.kind, r.external_key, r.status, d.online,
			COALESCE(CAST(JSON_EXTRACT(r.attrs_json, '$.ptzType') AS SIGNED), 0) AS ptz_type,
			COALESCE(CAST(JSON_EXTRACT(r.attrs_json, '$.hasAudio') AS SIGNED), 0) AS has_audio,
			COALESCE(CAST(JSON_EXTRACT(r.attrs_json, '$.longitude') AS DOUBLE), 0) AS longitude,
			COALESCE(CAST(JSON_EXTRACT(r.attrs_json, '$.latitude') AS DOUBLE), 0) AS latitude,
			COALESCE(CAST(JSON_EXTRACT(r.attrs_json, '$.mapLevel') AS SIGNED), 0) AS map_level,
			COALESCE(JSON_UNQUOTE(JSON_EXTRACT(r.attrs_json, '$.tags')), '') AS tags,
			COALESCE(CAST(JSON_EXTRACT(r.attrs_json, '$.enabled') AS SIGNED), 1) AS enabled`).
		Order("d.internal_code ASC, r.id ASC").
		Offset((page - 1) * count).Limit(count).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]*domainasset.ChannelView, 0, len(rows))
	for i := range rows {
		x := &rows[i]
		view := &domainasset.ChannelView{
			ResourceID: x.ResourceID, InternalCode: x.InternalCode, DeviceID: x.DeviceID,
			DeviceInternalCode: x.DeviceInternalCode, DeviceName: x.DeviceName, Protocol: x.Protocol,
			Name: x.Name, Kind: x.Kind, ExternalKey: x.ExternalKey, Status: x.Status,
			Online: x.Online, PTZType: x.PTZType, HasAudio: x.HasAudio,
			Longitude: x.Longitude, Latitude: x.Latitude, MapLevel: x.MapLevel,
			Tags: x.Tags, Enabled: x.Enabled == 1,
		}
		if strings.EqualFold(x.Protocol, "gb28181") {
			view.GBCode = x.ExternalKey
		}
		out = append(out, view)
	}
	return out, total, nil
}

func toDomainAssetResource(m *model.AssetResource) *domainasset.Resource {
	res := &domainasset.Resource{
		ID: m.ID, InternalCode: m.InternalCode, DeviceID: m.DeviceID, BindingID: m.BindingID, Kind: m.Kind,
		ExternalKey: m.ExternalKey, Name: m.Name, Status: m.Status,
		LegacyChannelID: m.LegacyChannelID, AttrsJSON: m.AttrsJSON,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
		Enabled: true,
	}
	var attrs map[string]any
	if json.Unmarshal([]byte(m.AttrsJSON), &attrs) == nil {
		if v, ok := attrs["longitude"].(float64); ok {
			res.Longitude = v
		}
		if v, ok := attrs["latitude"].(float64); ok {
			res.Latitude = v
		}
		if v, ok := attrs["mapLevel"].(float64); ok {
			res.MapLevel = int(v)
		}
		if v, ok := attrs["ptzType"].(float64); ok {
			res.PTZType = int(v)
		}
		if v, ok := attrs["hasAudio"].(bool); ok {
			res.HasAudio = v
		}
		if v, ok := attrs["tags"].(string); ok {
			res.Tags = v
		}
		if v, ok := attrs["enabled"].(bool); ok {
			res.Enabled = v
		}
	}
	return res
}

func toDomainAssetProfile(m *model.AssetStreamProfile) *domainasset.StreamProfile {
	return &domainasset.StreamProfile{
		ID: m.ID, ResourceID: m.ResourceID, Role: m.Role, Name: m.Name,
		SelectorJSON: m.SelectorJSON, Codec: m.Codec, Resolution: m.Resolution,
		IsDefault: m.IsDefault, CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

// SetCapabilityAvailable 运行时更新能力可用性。
func (r *AssetRepository) SetCapabilityAvailable(scopeType, scopeID, operation string, available bool) error {
	return r.db.Model(&model.AssetCapability{}).
		Where("scope_type = ? AND scope_id = ? AND operation = ?", scopeType, scopeID, operation).
		Updates(map[string]any{"available": available, "update_time": nowText()}).Error
}
