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

// AssetRepository 统一资产仓储。
// 统一 schema 下：
//   设备 = zws_device（身份稳定）+ zws_access_binding（接入参数/协议）
//   通道 = zws_channel（身份/状态）+ zws_channel_ext（经纬度/PTZ/有音频/启用/标签）
//   码流 = zws_stream_profile；能力 = zws_capability；事件源 = zws_event_source
// 对外（domain asset）字符串 ID = 库内 BIGINT 主键的十进制字符串。
type AssetRepository struct{ db *gorm.DB }

func NewAssetRepository(db *gorm.DB) *AssetRepository { return &AssetRepository{db: db} }

var _ domainasset.Repository = (*AssetRepository)(nil)

func nowText() string { return time.Now().Format("2006-01-02 15:04:05") }

// --- Device ---

func (r *AssetRepository) CreateDevice(d *domainasset.PhysicalDevice) error {
	now := nowText()
	m := &model.Device{
		InternalCode: d.InternalCode, Name: d.Name, Manufacturer: d.Vendor,
		Model: d.Model, SerialNumber: d.SerialNumber, Enabled: d.Enabled, Online: d.Online,
		Status: onlineStatus(d.Online), CreateTime: now, UpdateTime: now,
	}
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	d.ID = formatID(m.ID)
	d.CreateTime, d.UpdateTime = now, now
	return nil
}

func (r *AssetRepository) UpsertDevice(d *domainasset.PhysicalDevice) error {
	if id, ok := parseID(d.ID); ok && id > 0 {
		return r.db.Model(&model.Device{}).Where("id = ?", id).Updates(map[string]any{
			"internal_code": d.InternalCode, "name": d.Name, "manufacturer": d.Vendor,
			"model": d.Model, "serial_number": d.SerialNumber, "enabled": d.Enabled,
			"online": d.Online, "status": onlineStatus(d.Online), "update_time": nowText(),
		}).Error
	}
	return r.CreateDevice(d)
}

func (r *AssetRepository) GetDevice(id string) (*domainasset.PhysicalDevice, error) {
	pk, ok := parseID(id)
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	var m model.Device
	if err := r.db.First(&m, pk).Error; err != nil {
		return nil, err
	}
	return toDomainAssetDevice(&m), nil
}

func (r *AssetRepository) GetDeviceByInternalCode(code string) (*domainasset.PhysicalDevice, error) {
	var m model.Device
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
	q := r.db.Table("zws_device d").
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
		model.Device
		BID              int64  `gorm:"column:b_id"`
		Protocol         string `gorm:"column:b_protocol"`
		Mode             string `gorm:"column:b_mode"`
		ExternalID       string `gorm:"column:b_external_id"`
		Endpoint         string `gorm:"column:b_endpoint"`
		PluginInstanceID string `gorm:"column:b_plugin_instance_id"`
		ConfigJSON       string `gorm:"column:b_config_json"`
		BStatus          string `gorm:"column:b_status"`
		LastSeen         int64  `gorm:"column:b_last_seen"`
		BCreateTime      string `gorm:"column:b_create_time"`
		BUpdateTime      string `gorm:"column:b_update_time"`
	}
	var rows []row
	err := q.Select(`d.*, COALESCE(b.id,0) AS b_id, b.protocol AS b_protocol, b.mode AS b_mode,
		b.external_id AS b_external_id, b.endpoint AS b_endpoint,
		b.plugin_instance_id AS b_plugin_instance_id, b.config_json AS b_config_json,
		b.status AS b_status, b.last_seen AS b_last_seen,
		b.create_time AS b_create_time, b.update_time AS b_update_time`).
		Order("d.update_time DESC").
		Offset((page - 1) * count).Limit(count).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	counts := map[int64]int{}
	var res []struct {
		DeviceID int64 `gorm:"column:device_id"`
		N        int   `gorm:"column:n"`
	}
	_ = r.db.Table("zws_channel").Select("device_id, COUNT(*) AS n").Group("device_id").Scan(&res).Error
	for _, c := range res {
		counts[c.DeviceID] = c.N
	}
	out := make([]*domainasset.DeviceSummary, 0, len(rows))
	for i := range rows {
		var b *domainasset.AccessBinding
		if rows[i].BID > 0 {
			b = &domainasset.AccessBinding{
				ID: formatID(rows[i].BID), DeviceID: formatID(rows[i].ID),
				Protocol: rows[i].Protocol, Mode: rows[i].Mode,
				ExternalID: rows[i].ExternalID, Endpoint: rows[i].Endpoint,
				PluginInstanceID: rows[i].PluginInstanceID, ConfigJSON: rows[i].ConfigJSON,
				Status: rows[i].BStatus, LastSeen: rows[i].LastSeen,
				CreateTime: rows[i].BCreateTime, UpdateTime: rows[i].BUpdateTime,
			}
		}
		out = append(out, &domainasset.DeviceSummary{
			Device:        toDomainAssetDevice(&rows[i].Device),
			Binding:       b,
			ResourceCount: counts[rows[i].ID],
		})
	}
	return out, total, nil
}

func (r *AssetRepository) UpdateDeviceFields(id string, patch domainasset.DevicePatch) error {
	pk, ok := parseID(id)
	if !ok {
		return gorm.ErrRecordNotFound
	}
	fields := map[string]any{"update_time": nowText()}
	if patch.Name != nil {
		fields["name"] = strings.TrimSpace(*patch.Name)
	}
	if patch.Vendor != nil {
		fields["manufacturer"] = strings.TrimSpace(*patch.Vendor)
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
	return r.db.Model(&model.Device{}).Where("id = ?", pk).Updates(fields).Error
}

// --- Binding ---

func (r *AssetRepository) CreateBinding(b *domainasset.AccessBinding) error {
	now := nowText()
	devID, ok := parseID(b.DeviceID)
	if !ok {
		return fmt.Errorf("invalid device id: %s", b.DeviceID)
	}
	m := &model.AccessBinding{
		DeviceID: devID, Protocol: b.Protocol, Mode: b.Mode,
		ExternalID: b.ExternalID, Endpoint: b.Endpoint, PluginInstanceID: b.PluginInstanceID,
		ConfigJSON: b.ConfigJSON, Status: b.Status, LastSeen: b.LastSeen,
		CreateTime: now, UpdateTime: now,
	}
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	b.ID = formatID(m.ID)
	b.CreateTime, b.UpdateTime = now, now
	return nil
}

func (r *AssetRepository) UpsertBinding(b *domainasset.AccessBinding) error {
	devID, ok := parseID(b.DeviceID)
	if !ok {
		return fmt.Errorf("invalid device id: %s", b.DeviceID)
	}
	var existing model.AccessBinding
	err := r.db.Where("device_id = ?", devID).First(&existing).Error
	now := nowText()
	if err == gorm.ErrRecordNotFound {
		return r.CreateBinding(b)
	}
	if err != nil {
		return err
	}
	if err := r.db.Model(&model.AccessBinding{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"protocol": b.Protocol, "mode": b.Mode, "external_id": b.ExternalID,
		"endpoint": b.Endpoint, "plugin_instance_id": b.PluginInstanceID,
		"config_json": b.ConfigJSON, "status": b.Status, "last_seen": b.LastSeen,
		"update_time": now,
	}).Error; err != nil {
		return err
	}
	b.ID = formatID(existing.ID)
	return nil
}

func (r *AssetRepository) GetBindingByDevice(deviceID string) (*domainasset.AccessBinding, error) {
	devID, ok := parseID(deviceID)
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	var m model.AccessBinding
	if err := r.db.Where("device_id = ?", devID).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainAssetBinding(&m), nil
}

// --- Resource (= Channel) ---

func (r *AssetRepository) UpsertResource(res *domainasset.Resource) error {
	now := nowText()
	devID, ok := parseID(res.DeviceID)
	if !ok {
		return fmt.Errorf("invalid device id: %s", res.DeviceID)
	}
	var existing model.Channel
	var err error
	if pk, has := parseID(res.ID); has && pk > 0 {
		err = r.db.First(&existing, pk).Error
	} else {
		err = r.db.Where("device_id = ? AND external_key = ?", devID, res.ExternalKey).First(&existing).Error
	}
	if err == gorm.ErrRecordNotFound {
		m := &model.Channel{
			DeviceID: devID, InternalCode: res.InternalCode, ExternalKey: res.ExternalKey,
			Kind: kindOrDefault(res.Kind), Name: res.Name, Status: res.Status, Online: res.Status == "ON",
			CreateTime: now, UpdateTime: now,
		}
		if err := r.db.Create(m).Error; err != nil {
			return err
		}
		res.ID = formatID(m.ID)
		return r.upsertChannelExt(m.ID, res, now)
	}
	if err != nil {
		return err
	}
	if err := r.db.Model(&model.Channel{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"internal_code": firstNonEmptyStr(res.InternalCode, existing.InternalCode),
		"external_key":  firstNonEmptyStr(res.ExternalKey, existing.ExternalKey),
		"kind":          kindOrDefault(res.Kind), "name": res.Name, "status": res.Status,
		"online": res.Status == "ON", "update_time": now,
	}).Error; err != nil {
		return err
	}
	res.ID = formatID(existing.ID)
	return r.upsertChannelExt(existing.ID, res, now)
}

func (r *AssetRepository) upsertChannelExt(channelID int64, res *domainasset.Resource, now string) error {
	var ext model.ChannelExt
	err := r.db.Where("channel_id = ?", channelID).First(&ext).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(&model.ChannelExt{
			ChannelID: channelID, Longitude: res.Longitude, Latitude: res.Latitude,
			MapLevel: res.MapLevel, PTZType: res.PTZType, HasAudio: res.HasAudio,
			Enabled: res.Enabled, Tags: res.Tags, ExtraJSON: res.AttrsJSON,
			CreateTime: now, UpdateTime: now,
		}).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&model.ChannelExt{}).Where("id = ?", ext.ID).Updates(map[string]any{
		"longitude": res.Longitude, "latitude": res.Latitude, "map_level": res.MapLevel,
		"ptz_type": res.PTZType, "has_audio": res.HasAudio, "enabled": res.Enabled,
		"tags": res.Tags, "extra_json": res.AttrsJSON, "update_time": now,
	}).Error
}

func (r *AssetRepository) ListResources(deviceID string) ([]*domainasset.Resource, error) {
	devID, ok := parseID(deviceID)
	if !ok {
		return []*domainasset.Resource{}, nil
	}
	var rows []model.Channel
	if err := r.db.Where("device_id = ?", devID).Order("kind, name").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainasset.Resource, 0, len(rows))
	for i := range rows {
		out = append(out, r.hydrateAssetResource(&rows[i]))
	}
	return out, nil
}

func (r *AssetRepository) GetResource(id string) (*domainasset.Resource, error) {
	pk, ok := parseID(id)
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	var m model.Channel
	if err := r.db.First(&m, pk).Error; err != nil {
		return nil, err
	}
	return r.hydrateAssetResource(&m), nil
}

func (r *AssetRepository) GetResourceByInternalCode(code string) (*domainasset.Resource, error) {
	var m model.Channel
	if err := r.db.Where("internal_code = ?", code).First(&m).Error; err != nil {
		return nil, err
	}
	return r.hydrateAssetResource(&m), nil
}

func (r *AssetRepository) hydrateAssetResource(m *model.Channel) *domainasset.Resource {
	res := &domainasset.Resource{
		ID: formatID(m.ID), InternalCode: m.InternalCode, DeviceID: formatID(m.DeviceID),
		Kind: m.Kind, ExternalKey: m.ExternalKey, Name: m.Name, Status: m.Status,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime, Enabled: true,
	}
	var b model.AccessBinding
	if err := r.db.Where("device_id = ?", m.DeviceID).First(&b).Error; err == nil {
		res.BindingID = formatID(b.ID)
	}
	var ext model.ChannelExt
	if err := r.db.Where("channel_id = ?", m.ID).First(&ext).Error; err == nil {
		res.Longitude = ext.Longitude
		res.Latitude = ext.Latitude
		res.MapLevel = ext.MapLevel
		res.PTZType = ext.PTZType
		res.HasAudio = ext.HasAudio
		res.Enabled = ext.Enabled
		res.Tags = ext.Tags
		res.AttrsJSON = ext.ExtraJSON
	}
	return res
}

func (r *AssetRepository) UpdateResourceFields(id string, patch domainasset.ResourcePatch) error {
	pk, ok := parseID(id)
	if !ok {
		return gorm.ErrRecordNotFound
	}
	var m model.Channel
	if err := r.db.First(&m, pk).Error; err != nil {
		return err
	}
	if patch.Name != nil {
		if err := r.db.Model(&model.Channel{}).Where("id = ?", pk).
			Updates(map[string]any{"name": strings.TrimSpace(*patch.Name), "update_time": nowText()}).Error; err != nil {
			return err
		}
	}
	var ext model.ChannelExt
	err := r.db.Where("channel_id = ?", pk).First(&ext).Error
	if err == gorm.ErrRecordNotFound {
		ext = model.ChannelExt{ChannelID: pk, Enabled: true, CreateTime: nowText(), UpdateTime: nowText()}
		if err := r.db.Create(&ext).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	updates := map[string]any{"update_time": nowText()}
	if patch.Longitude != nil {
		updates["longitude"] = *patch.Longitude
	}
	if patch.Latitude != nil {
		updates["latitude"] = *patch.Latitude
	}
	if patch.MapLevel != nil {
		updates["map_level"] = *patch.MapLevel
	}
	if patch.PTZType != nil {
		updates["ptz_type"] = *patch.PTZType
	}
	if patch.HasAudio != nil {
		updates["has_audio"] = *patch.HasAudio
	}
	if patch.Tags != nil {
		updates["tags"] = strings.TrimSpace(*patch.Tags)
	}
	if patch.Enabled != nil {
		updates["enabled"] = *patch.Enabled
	}
	// 非国标通道的国标编码/属性落 gb_json
	if patch.GBCode != nil || patch.GBFields != nil {
		var g gbChannelJSON
		if strings.TrimSpace(ext.GBJSON) != "" {
			_ = json.Unmarshal([]byte(ext.GBJSON), &g)
		}
		if patch.GBCode != nil {
			g.GBName = strings.TrimSpace(*patch.GBCode)
		}
		// GBFields 覆盖到 gb_json 的 CivilCode/ParentID 等常见键
		for k, v := range patch.GBFields {
			if s, ok := v.(string); ok {
				switch k {
				case "civilCode":
					g.CivilCode = s
				case "parentId":
					g.ParentID = s
				case "businessGroupId":
					g.BusinessGroupID = s
				case "manufacturer":
					g.Manufacturer = s
				case "model":
					g.Model = s
				}
			}
		}
		gj, _ := json.Marshal(g)
		updates["gb_json"] = string(gj)
	}
	return r.db.Model(&model.ChannelExt{}).Where("id = ?", ext.ID).Updates(updates).Error
}

func (r *AssetRepository) ListChannels(page, count int, query, protocol, deviceInternalCode string) ([]*domainasset.ChannelView, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Table("zws_channel c").
		Joins("JOIN zws_device d ON d.id = c.device_id").
		Joins("LEFT JOIN zws_access_binding b ON b.device_id = d.id")
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("c.name LIKE ? OR c.internal_code LIKE ? OR c.gb_code LIKE ? OR d.name LIKE ?",
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
		ChannelID          int64   `gorm:"column:channel_id"`
		InternalCode       string  `gorm:"column:internal_code"`
		DeviceID           int64   `gorm:"column:device_id"`
		DeviceInternalCode string  `gorm:"column:device_internal_code"`
		DeviceName         string  `gorm:"column:device_name"`
		Protocol           string  `gorm:"column:protocol"`
		Name               string  `gorm:"column:name"`
		Kind               string  `gorm:"column:kind"`
		ExternalKey        string  `gorm:"column:external_key"`
		GbCode             string  `gorm:"column:gb_code"`
		Status             string  `gorm:"column:status"`
		Online             bool    `gorm:"column:online"`
		PTZType            int     `gorm:"column:ptz_type"`
		HasAudio           bool    `gorm:"column:has_audio"`
		Longitude          float64 `gorm:"column:longitude"`
		Latitude           float64 `gorm:"column:latitude"`
		MapLevel           int     `gorm:"column:map_level"`
		Tags               string  `gorm:"column:tags"`
		Enabled            bool    `gorm:"column:enabled"`
		GBJSON             string  `gorm:"column:gb_json"`
	}
	var rows []row
	err := q.Select(`c.id AS channel_id, c.internal_code, c.device_id, d.internal_code AS device_internal_code,
			d.name AS device_name, b.protocol, c.name, c.kind, c.external_key, c.gb_code, c.status, c.online,
			COALESCE(e.ptz_type,0) AS ptz_type, COALESCE(e.has_audio,0) AS has_audio,
			COALESCE(e.longitude,0) AS longitude, COALESCE(e.latitude,0) AS latitude,
			COALESCE(e.map_level,0) AS map_level, COALESCE(e.tags,'') AS tags,
			COALESCE(e.enabled,1) AS enabled, COALESCE(e.gb_json,'') AS gb_json`).
		Joins("LEFT JOIN zws_channel_ext e ON e.channel_id = c.id").
		Order("d.internal_code ASC, c.id ASC").
		Offset((page - 1) * count).Limit(count).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]*domainasset.ChannelView, 0, len(rows))
	for i := range rows {
		x := &rows[i]
		view := &domainasset.ChannelView{
			ResourceID: formatID(x.ChannelID), InternalCode: x.InternalCode, DeviceID: formatID(x.DeviceID),
			DeviceInternalCode: x.DeviceInternalCode, DeviceName: x.DeviceName, Protocol: x.Protocol,
			Name: x.Name, Kind: x.Kind, ExternalKey: x.ExternalKey, GBCode: x.GbCode, Status: x.Status,
			Online: x.Online, PTZType: x.PTZType, HasAudio: x.HasAudio,
			Longitude: x.Longitude, Latitude: x.Latitude, MapLevel: x.MapLevel,
			Tags: x.Tags, Enabled: x.Enabled,
		}
		if strings.TrimSpace(x.GBJSON) != "" {
			var g map[string]any
			if json.Unmarshal([]byte(x.GBJSON), &g) == nil {
				view.GB = g
			}
		}
		out = append(out, view)
	}
	return out, total, nil
}

// --- Profile ---

func (r *AssetRepository) UpsertProfile(p *domainasset.StreamProfile) error {
	now := nowText()
	chID, ok := parseID(p.ResourceID)
	if !ok {
		return fmt.Errorf("invalid resource id: %s", p.ResourceID)
	}
	var existing model.StreamProfile
	err := r.db.Where("channel_id = ? AND role = ?", chID, p.Role).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		m := &model.StreamProfile{
			ChannelID: chID, Role: p.Role, Name: p.Name, SelectorJSON: p.SelectorJSON,
			Codec: p.Codec, Resolution: p.Resolution, IsDefault: p.IsDefault,
			CreateTime: now, UpdateTime: now,
		}
		if err := r.db.Create(m).Error; err != nil {
			return err
		}
		p.ID = formatID(m.ID)
		return nil
	}
	if err != nil {
		return err
	}
	if err := r.db.Model(&model.StreamProfile{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"name": p.Name, "selector_json": p.SelectorJSON, "codec": p.Codec,
		"resolution": p.Resolution, "is_default": p.IsDefault, "update_time": now,
	}).Error; err != nil {
		return err
	}
	p.ID = formatID(existing.ID)
	return nil
}

func (r *AssetRepository) ListProfiles(resourceID string) ([]*domainasset.StreamProfile, error) {
	chID, ok := parseID(resourceID)
	if !ok {
		return []*domainasset.StreamProfile{}, nil
	}
	var rows []model.StreamProfile
	if err := r.db.Where("channel_id = ?", chID).Order("is_default DESC, role").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainasset.StreamProfile, 0, len(rows))
	for i := range rows {
		out = append(out, toDomainAssetProfile(&rows[i]))
	}
	return out, nil
}

func (r *AssetRepository) GetProfile(id string) (*domainasset.StreamProfile, error) {
	pk, ok := parseID(id)
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	var m model.StreamProfile
	if err := r.db.First(&m, pk).Error; err != nil {
		return nil, err
	}
	return toDomainAssetProfile(&m), nil
}

// --- Capability ---

func (r *AssetRepository) ReplaceCapabilities(scopeType, scopeID string, caps []*domainasset.Capability) error {
	chID, ok := parseID(scopeID)
	if !ok {
		return fmt.Errorf("invalid scope id: %s", scopeID)
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("channel_id = ?", chID).Delete(&model.Capability{}).Error; err != nil {
			return err
		}
		now := nowText()
		for _, c := range caps {
			if err := tx.Create(&model.Capability{
				ChannelID: chID, Operation: c.Operation, Supported: c.Supported,
				Configured: c.Configured, Available: c.Available, ParamsJSON: c.ParamsJSON,
				Source: c.Source, UpdateTime: now,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *AssetRepository) ListCapabilities(scopeType, scopeID string) ([]*domainasset.Capability, error) {
	chID, ok := parseID(scopeID)
	if !ok {
		return []*domainasset.Capability{}, nil
	}
	var rows []model.Capability
	if err := r.db.Where("channel_id = ?", chID).Order("operation").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainasset.Capability, 0, len(rows))
	for i := range rows {
		out = append(out, &domainasset.Capability{
			ID: formatID(rows[i].ID), ScopeType: scopeType, ScopeID: scopeID,
			Operation: rows[i].Operation, Supported: rows[i].Supported,
			Configured: rows[i].Configured, Available: rows[i].Available,
			ParamsJSON: rows[i].ParamsJSON, Source: rows[i].Source, UpdateTime: rows[i].UpdateTime,
		})
	}
	return out, nil
}

func (r *AssetRepository) SetCapabilityAvailable(scopeType, scopeID, operation string, available bool) error {
	chID, ok := parseID(scopeID)
	if !ok {
		return nil
	}
	return r.db.Model(&model.Capability{}).
		Where("channel_id = ? AND operation = ?", chID, operation).
		Updates(map[string]any{"available": available, "update_time": nowText()}).Error
}

// --- EventSource ---

func (r *AssetRepository) UpsertEventSource(e *domainasset.EventSource) error {
	devID, ok := parseID(e.DeviceID)
	if !ok {
		return fmt.Errorf("invalid device id: %s", e.DeviceID)
	}
	var chID *int64
	if id, ok := parseID(e.ResourceID); ok && id > 0 {
		chID = &id
	}
	var existing model.EventSource
	q := r.db.Where("device_id = ?", devID)
	if chID != nil {
		q = q.Where("channel_id = ?", *chID)
	} else {
		q = q.Where("channel_id IS NULL")
	}
	err := q.Where("kind = ?", e.Kind).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		m := &model.EventSource{
			DeviceID: devID, ChannelID: chID, Kind: e.Kind,
			ExternalKey: e.ExternalKey, Enabled: e.Enabled, ConfigJSON: e.ConfigJSON,
		}
		if err := r.db.Create(m).Error; err != nil {
			return err
		}
		e.ID = formatID(m.ID)
		return nil
	}
	if err != nil {
		return err
	}
	if err := r.db.Model(&model.EventSource{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"external_key": e.ExternalKey, "enabled": e.Enabled, "config_json": e.ConfigJSON,
	}).Error; err != nil {
		return err
	}
	e.ID = formatID(existing.ID)
	return nil
}

func (r *AssetRepository) ListEventSources(deviceID string) ([]*domainasset.EventSource, error) {
	devID, ok := parseID(deviceID)
	if !ok {
		return []*domainasset.EventSource{}, nil
	}
	var rows []model.EventSource
	if err := r.db.Where("device_id = ?", devID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainasset.EventSource, 0, len(rows))
	for i := range rows {
		es := &domainasset.EventSource{
			ID: formatID(rows[i].ID), DeviceID: deviceID, Kind: rows[i].Kind,
			ExternalKey: rows[i].ExternalKey, Enabled: rows[i].Enabled, ConfigJSON: rows[i].ConfigJSON,
		}
		if rows[i].ChannelID != nil {
			es.ResourceID = formatID(*rows[i].ChannelID)
		}
		out = append(out, es)
	}
	return out, nil
}

// --- helpers ---

func toDomainAssetDevice(m *model.Device) *domainasset.PhysicalDevice {
	return &domainasset.PhysicalDevice{
		ID: formatID(m.ID), InternalCode: m.InternalCode, Name: m.Name, Vendor: m.Manufacturer,
		Model: m.Model, SerialNumber: m.SerialNumber, Enabled: m.Enabled, Online: m.Online,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func toDomainAssetBinding(m *model.AccessBinding) *domainasset.AccessBinding {
	return &domainasset.AccessBinding{
		ID: formatID(m.ID), DeviceID: formatID(m.DeviceID), Protocol: m.Protocol, Mode: m.Mode,
		ExternalID: m.ExternalID, Endpoint: m.Endpoint, PluginInstanceID: m.PluginInstanceID,
		ConfigJSON: m.ConfigJSON, Status: m.Status, LastSeen: m.LastSeen,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func toDomainAssetProfile(m *model.StreamProfile) *domainasset.StreamProfile {
	return &domainasset.StreamProfile{
		ID: formatID(m.ID), ResourceID: formatID(m.ChannelID), Role: m.Role, Name: m.Name,
		SelectorJSON: m.SelectorJSON, Codec: m.Codec, Resolution: m.Resolution,
		IsDefault: m.IsDefault, CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func kindOrDefault(k string) string {
	if strings.TrimSpace(k) == "" {
		return "video_sensor"
	}
	return k
}
