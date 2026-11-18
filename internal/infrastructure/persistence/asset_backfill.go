package persistence

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	domainasset "zero-web-server/internal/domain/asset"
	"zero-web-server/internal/infrastructure/persistence/model"
	"zero-web-server/pkg/idcode"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AssetProjector 提供统一资产投影能力。
type AssetProjector struct{ db *gorm.DB }

func NewAssetProjector(db *gorm.DB) *AssetProjector { return &AssetProjector{db: db} }

// Backfill 将 GB / ONVIF / 插件旧数据幂等投影到统一资产模型。
func (p *AssetProjector) Backfill() error {
	if err := p.backfillGB(); err != nil {
		return err
	}
	if err := p.backfillONVIF(); err != nil {
		return err
	}
	if err := p.backfillPlugin(); err != nil {
		return err
	}
	// 投影只做 upsert；旧表中已删除、但统一资产残留的行在此清理。
	return p.Prune()
}

// Prune 回收孤儿：legacy 行已不存在、但统一资产仍存在的，按 zws_unified_ref 映射级联删除。
// 幂等、可反复调用（启动 Backfill 与删除设备后都会调用）。
func (p *AssetProjector) Prune() error {
	sets := map[string]map[string]bool{}
	for _, t := range []struct {
		table string
		model any
	}{
		{"zws_device", &model.GBDevice{}},
		{"zws_device_channel", &model.GBDeviceChannel{}},
		{"zws_onvif_device", &model.OnvifDevice{}},
		{"zws_onvif_channel", &model.OnvifChannel{}},
		{"zws_plugin_device", &model.PluginDevice{}},
		{"zws_plugin_channel", &model.PluginChannel{}},
	} {
		set, err := p.idSet(t.model)
		if err != nil {
			return err
		}
		sets[t.table] = set
	}

	var refs []model.UnifiedRef
	if err := p.db.Find(&refs).Error; err != nil {
		return err
	}
	for i := range refs {
		r := &refs[i]
		set, tracked := sets[r.LegacyTable]
		if !tracked {
			continue
		}
		if set[r.LegacyID] {
			continue // legacy 仍存在
		}
		// 孤儿：按实体类型级联删除统一资产
		switch r.EntityType {
		case "physical_device":
			if err := p.deleteDeviceTree(r.UnifiedID); err != nil {
				return err
			}
		case "resource":
			if err := p.deleteResourceTree(r.UnifiedID); err != nil {
				return err
			}
		}
		if err := p.db.Where("legacy_table = ? AND legacy_id = ?", r.LegacyTable, r.LegacyID).
			Delete(&model.UnifiedRef{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (p *AssetProjector) idSet(m any) (map[string]bool, error) {
	var ids []int64
	if err := p.db.Model(m).Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[strconv.FormatInt(id, 10)] = true
	}
	return set, nil
}

func (p *AssetProjector) deleteDeviceTree(deviceUID string) error {
	if deviceUID == "" {
		return nil
	}
	var resIDs []string
	if err := p.db.Model(&model.AssetResource{}).Where("device_id = ?", deviceUID).
		Pluck("id", &resIDs).Error; err != nil {
		return err
	}
	if len(resIDs) > 0 {
		if err := p.db.Where("resource_id IN ?", resIDs).Delete(&model.AssetStreamProfile{}).Error; err != nil {
			return err
		}
		if err := p.db.Where("scope_type = ? AND scope_id IN ?", "resource", resIDs).
			Delete(&model.AssetCapability{}).Error; err != nil {
			return err
		}
	}
	steps := []func() *gorm.DB{
		func() *gorm.DB { return p.db.Where("device_id = ?", deviceUID).Delete(&model.AssetResource{}) },
		func() *gorm.DB { return p.db.Where("device_id = ?", deviceUID).Delete(&model.AccessBinding{}) },
		func() *gorm.DB {
			return p.db.Where("scope_type = ? AND scope_id = ?", "device", deviceUID).Delete(&model.AssetCapability{})
		},
		func() *gorm.DB { return p.db.Where("device_id = ?", deviceUID).Delete(&model.AssetEventSource{}) },
		func() *gorm.DB { return p.db.Where("id = ?", deviceUID).Delete(&model.PhysicalDevice{}) },
	}
	for _, fn := range steps {
		if err := fn().Error; err != nil {
			return err
		}
	}
	return nil
}

func (p *AssetProjector) deleteResourceTree(resourceUID string) error {
	if resourceUID == "" {
		return nil
	}
	if err := p.db.Where("resource_id = ?", resourceUID).Delete(&model.AssetStreamProfile{}).Error; err != nil {
		return err
	}
	if err := p.db.Where("scope_type = ? AND scope_id = ?", "resource", resourceUID).
		Delete(&model.AssetCapability{}).Error; err != nil {
		return err
	}
	return p.db.Where("id = ?", resourceUID).Delete(&model.AssetResource{}).Error
}

func (p *AssetProjector) backfillGB() error {
	var devices []model.GBDevice
	if err := p.db.Find(&devices).Error; err != nil {
		return err
	}
	for i := range devices {
		dev := &devices[i]
		uid, err := p.legacyID("zws_device", strconv.Itoa(dev.ID))
		if err != nil {
			return err
		}
		internal := strings.TrimSpace(dev.InternalCode)
		if internal == "" {
			internal, _ = idcode.Device()
		}
		cfg, _ := json.Marshal(map[string]any{
			"deviceId": dev.DeviceID, "transport": dev.Transport, "streamMode": dev.StreamMode,
			"sdpIp": dev.SDPIP, "mediaServerId": dev.MediaServerID, "charset": dev.Charset,
		})
		if err := p.upsertDevice(uid, internal, dev.Name, dev.Manufacturer, dev.Model, "", dev.OnLine); err != nil {
			return err
		}
		status := onlineStatus(dev.OnLine)
		if err := p.upsertBinding(uid, "gb28181", "passive", dev.DeviceID, "", "", string(cfg), status, dev.ID); err != nil {
			return err
		}
		if err := p.upsertRef("zws_device", strconv.Itoa(dev.ID), "physical_device", uid); err != nil {
			return err
		}
		if err := p.upsertCapabilities("device", uid, "gb28181", dev.OnLine, nil); err != nil {
			return err
		}
		// GB 报警事件源
		evID := stableID("event", uid, "alarm")
		enabled := dev.SubscribeCycleForAlarm > 0
		if err := p.upsertEvent(evID, uid, "", "alarm", dev.DeviceID, enabled); err != nil {
			return err
		}

		var channels []model.GBDeviceChannel
		if err := p.db.Where("device_id = ? AND data_type = ?", dev.DeviceID, 1).Find(&channels).Error; err != nil {
			return err
		}
		for ci := range channels {
			ch := &channels[ci]
			chUID, err := p.legacyID("zws_device_channel", strconv.Itoa(ch.ID))
			if err != nil {
				return err
			}
			binding, _ := p.getBinding(uid)
			bindingID := ""
			if binding != nil {
				bindingID = binding.ID
			}
			attrs, _ := json.Marshal(map[string]any{
				"gbDeviceId": ch.GBDeviceID, "ptzType": ch.PTZType, "hasAudio": ch.HasAudio,
			})
			if err := p.upsertResource(chUID, ch.InternalCode, uid, bindingID, domainasset.ResourceVideoSensor,
				ch.GBDeviceID, firstNonEmpty(ch.GBName, ch.Name), firstNonEmpty(ch.GBStatus, ch.Status, "OFF"),
				ch.ID, string(attrs)); err != nil {
				return err
			}
			if err := p.upsertRef("zws_device_channel", strconv.Itoa(ch.ID), "resource", chUID); err != nil {
				return err
			}
			mainID := stableID("profile", chUID, "main")
			if err := p.upsertProfile(mainID, chUID, domainasset.ProfileMain, "主码流",
				`{"stream":"main"}`, "", "", true); err != nil {
				return err
			}
			if err := p.upsertCapabilities("resource", chUID, "gb28181", dev.OnLine, channelCaps(ch.PTZType, ch.HasAudio)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *AssetProjector) backfillONVIF() error {
	var devices []model.OnvifDevice
	if err := p.db.Find(&devices).Error; err != nil {
		return err
	}
	for i := range devices {
		dev := &devices[i]
		uid, err := p.legacyID("zws_onvif_device", strconv.FormatInt(dev.ID, 10))
		if err != nil {
			return err
		}
		internal := strings.TrimSpace(dev.InternalCode)
		if internal == "" {
			internal, _ = idcode.Device()
		}
		cfg, _ := json.Marshal(map[string]any{
			"ip": dev.IP, "port": dev.Port, "username": dev.Username,
			"deviceUri": dev.DeviceURI, "mediaUri": dev.MediaURI, "ptzUri": dev.PTZURI,
			"gbCode": dev.GbCode, "mediaServerId": dev.MediaServerID,
		})
		if err := p.upsertDevice(uid, internal, dev.Name, dev.Manufacturer, dev.Model, dev.SerialNumber, dev.OnLine); err != nil {
			return err
		}
		endpoint := fmt.Sprintf("%s:%d", dev.IP, dev.Port)
		if err := p.upsertBinding(uid, "onvif", "active", endpoint, endpoint, "", string(cfg), onlineStatus(dev.OnLine), int(dev.ID)); err != nil {
			return err
		}
		if err := p.upsertRef("zws_onvif_device", strconv.FormatInt(dev.ID, 10), "physical_device", uid); err != nil {
			return err
		}
		if err := p.upsertCapabilities("device", uid, "onvif", dev.OnLine, nil); err != nil {
			return err
		}

		var channels []model.OnvifChannel
		if err := p.db.Where("device_id = ?", dev.ID).Find(&channels).Error; err != nil {
			return err
		}
		binding, _ := p.getBinding(uid)
		bindingID := ""
		if binding != nil {
			bindingID = binding.ID
		}
		for ci := range channels {
			ch := &channels[ci]
			chUID, err := p.legacyID("zws_onvif_channel", strconv.FormatInt(ch.ID, 10))
			if err != nil {
				return err
			}
			external := firstNonEmpty(ch.VideoSource, ch.ProfileToken, strconv.FormatInt(ch.ID, 10))
			attrs, _ := json.Marshal(map[string]any{
				"profileToken": ch.ProfileToken, "videoSource": ch.VideoSource,
				"resolution": ch.Resolution, "codec": ch.Codec, "hasPtz": ch.HasPTZ,
			})
			if err := p.upsertResource(chUID, ch.InternalCode, uid, bindingID, domainasset.ResourceVideoSensor,
				external, ch.Name, firstNonEmpty(ch.Status, "OFF"), int(ch.ID), string(attrs)); err != nil {
				return err
			}
			if err := p.upsertRef("zws_onvif_channel", strconv.FormatInt(ch.ID, 10), "resource", chUID); err != nil {
				return err
			}
			if err := p.upsertONVIFProfiles(chUID, ch); err != nil {
				return err
			}
			ptzType := 0
			if ch.HasPTZ {
				ptzType = 1
			}
			if err := p.upsertCapabilities("resource", chUID, "onvif", dev.OnLine, channelCaps(ptzType, ch.HasAudio)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *AssetProjector) backfillPlugin() error {
	var devices []model.PluginDevice
	if err := p.db.Find(&devices).Error; err != nil {
		return err
	}
	for i := range devices {
		dev := &devices[i]
		uid, err := p.legacyID("zws_plugin_device", strconv.Itoa(dev.ID))
		if err != nil {
			return err
		}
		internal := strings.TrimSpace(dev.InternalCode)
		if internal == "" {
			internal, _ = idcode.Device()
		}
		if err := p.upsertDevice(uid, internal, dev.Name, dev.Manufacturer, dev.Model, dev.SerialNumber, dev.Online); err != nil {
			return err
		}
		if err := p.upsertBinding(uid, "plugin", "active", dev.DeviceUID, "", dev.PluginID,
			dev.DriverParams, onlineStatus(dev.Online), dev.ID); err != nil {
			return err
		}
		if err := p.upsertRef("zws_plugin_device", strconv.Itoa(dev.ID), "physical_device", uid); err != nil {
			return err
		}
		if err := p.upsertCapabilities("device", uid, "plugin", dev.Online, nil); err != nil {
			return err
		}
		var channels []model.PluginChannel
		if err := p.db.Where("device_db_id = ?", dev.ID).Find(&channels).Error; err != nil {
			return err
		}
		binding, _ := p.getBinding(uid)
		bindingID := ""
		if binding != nil {
			bindingID = binding.ID
		}
		for ci := range channels {
			ch := &channels[ci]
			chUID, err := p.legacyID("zws_plugin_channel", strconv.Itoa(ch.ID))
			if err != nil {
				return err
			}
			attrs, _ := json.Marshal(map[string]any{"channelId": ch.ChannelID, "ptzType": ch.PTZType, "hasAudio": ch.HasAudio})
			if err := p.upsertResource(chUID, ch.InternalCode, uid, bindingID, domainasset.ResourceVideoSensor,
				ch.ChannelID, ch.Name, firstNonEmpty(ch.Status, "OFF"), ch.ID, string(attrs)); err != nil {
				return err
			}
			if err := p.upsertRef("zws_plugin_channel", strconv.Itoa(ch.ID), "resource", chUID); err != nil {
				return err
			}
			mainID := stableID("profile", chUID, "main")
			if err := p.upsertProfile(mainID, chUID, domainasset.ProfileMain, "主码流",
				`{"channel":"main"}`, "", "", true); err != nil {
				return err
			}
			if err := p.upsertCapabilities("resource", chUID, "plugin", dev.Online, channelCaps(ch.PTZType, ch.HasAudio)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *AssetProjector) upsertONVIFProfiles(resourceUID string, ch *model.OnvifChannel) error {
	type profile struct {
		ProfileToken string `json:"profileToken"`
		Label        string `json:"label"`
		Resolution   string `json:"resolution"`
		Codec        string `json:"codec"`
	}
	items := []profile{}
	if strings.TrimSpace(ch.ProfilesJSON) != "" {
		_ = json.Unmarshal([]byte(ch.ProfilesJSON), &items)
	}
	if len(items) == 0 {
		items = append(items, profile{ProfileToken: ch.ProfileToken, Resolution: ch.Resolution, Codec: ch.Codec})
	}
	for i, it := range items {
		if strings.TrimSpace(it.ProfileToken) == "" {
			continue
		}
		role := inferProfileRole(it.Label, i)
		selector, _ := json.Marshal(map[string]any{"profileToken": it.ProfileToken})
		pid := stableID("profile", resourceUID, it.ProfileToken)
		isDefault := i == 0
		if err := p.upsertProfile(pid, resourceUID, role, firstNonEmpty(it.Label, role), string(selector),
			it.Codec, it.Resolution, isDefault); err != nil {
			return err
		}
	}
	return nil
}

func (p *AssetProjector) upsertDevice(id, internal, name, vendor, modelName, serial string, online bool) error {
	var existing model.PhysicalDevice
	err := p.db.Where("id = ?", id).First(&existing).Error
	now := nowText()
	if err == gorm.ErrRecordNotFound {
		return p.db.Create(&model.PhysicalDevice{
			ID: id, InternalCode: internal, Name: name, Vendor: vendor, Model: modelName,
			SerialNumber: serial, Enabled: true, Online: online, CreateTime: now, UpdateTime: now,
		}).Error
	}
	if err != nil {
		return err
	}
	// 保留用户改名：仅在名称为空时回填。
	nameUpdate := existing.Name
	if strings.TrimSpace(nameUpdate) == "" {
		nameUpdate = name
	}
	return p.db.Model(&model.PhysicalDevice{}).Where("id = ?", id).Updates(map[string]any{
		"name": nameUpdate, "vendor": firstNonEmpty(existing.Vendor, vendor),
		"model":         firstNonEmpty(existing.Model, modelName),
		"serial_number": firstNonEmpty(existing.SerialNumber, serial),
		"online":        online, "update_time": now,
	}).Error
}

func (p *AssetProjector) upsertBinding(deviceID, protocol, mode, externalID, endpoint, pluginInstance, cfg, status string, legacyDeviceID int) error {
	var existing model.AccessBinding
	err := p.db.Where("device_id = ?", deviceID).First(&existing).Error
	now := nowText()
	if err == gorm.ErrRecordNotFound {
		return p.db.Create(&model.AccessBinding{
			ID: uuid.NewString(), DeviceID: deviceID, Protocol: protocol, Mode: mode,
			ExternalID: externalID, Endpoint: endpoint, PluginInstanceID: pluginInstance,
			ConfigJSON: cfg, Status: status, LegacyDeviceID: legacyDeviceID,
			CreateTime: now, UpdateTime: now,
		}).Error
	}
	if err != nil {
		return err
	}
	return p.db.Model(&model.AccessBinding{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"protocol": protocol, "mode": mode, "external_id": externalID, "endpoint": endpoint,
		"plugin_instance_id": pluginInstance, "config_json": cfg, "status": status,
		"legacy_device_id": legacyDeviceID, "update_time": now,
	}).Error
}

func (p *AssetProjector) upsertResource(id, internalCode, deviceID, bindingID, kind, externalKey, name, status string, legacyChannelID int, attrs string) error {
	var existing model.AssetResource
	err := p.db.Where("id = ?", id).First(&existing).Error
	now := nowText()
	// 通道内码以 legacy 通道表为准（它按协议侧通道稳定生成/保留）；
	// legacy 为空时才沿用统一资产已有内码，最后兜底新生成，保证两边一致。
	code := strings.TrimSpace(internalCode)
	if code == "" {
		if strings.TrimSpace(existing.InternalCode) != "" {
			code = existing.InternalCode
		} else {
			code, err = idcode.Channel()
			if err != nil {
				return err
			}
		}
	}
	if err == gorm.ErrRecordNotFound {
		return p.db.Create(&model.AssetResource{
			ID: id, InternalCode: code, DeviceID: deviceID, BindingID: bindingID, Kind: kind,
			ExternalKey: externalKey, Name: name, Status: status,
			LegacyChannelID: legacyChannelID, AttrsJSON: attrs,
			CreateTime: now, UpdateTime: now,
		}).Error
	}
	if err != nil {
		return err
	}
	nameUpdate := existing.Name
	if strings.TrimSpace(nameUpdate) == "" {
		nameUpdate = name
	}
	return p.db.Model(&model.AssetResource{}).Where("id = ?", id).Updates(map[string]any{
		"internal_code": code,
		"binding_id":    bindingID, "external_key": externalKey, "name": nameUpdate,
		"status": status, "legacy_channel_id": legacyChannelID, "attrs_json": attrs,
		"update_time": now,
	}).Error
}

func (p *AssetProjector) upsertProfile(id, resourceID, role, name, selector, codec, resolution string, isDefault bool) error {
	var existing model.AssetStreamProfile
	err := p.db.Where("id = ?", id).First(&existing).Error
	now := nowText()
	if err == gorm.ErrRecordNotFound {
		return p.db.Create(&model.AssetStreamProfile{
			ID: id, ResourceID: resourceID, Role: role, Name: name,
			SelectorJSON: selector, Codec: codec, Resolution: resolution,
			IsDefault: isDefault, CreateTime: now, UpdateTime: now,
		}).Error
	}
	if err != nil {
		return err
	}
	return p.db.Model(&model.AssetStreamProfile{}).Where("id = ?", id).Updates(map[string]any{
		"role": role, "name": name, "selector_json": selector, "codec": codec,
		"resolution": resolution, "is_default": isDefault, "update_time": now,
	}).Error
}

func (p *AssetProjector) upsertCapabilities(scopeType, scopeID, protocol string, online bool, extra []string) error {
	ops := protocolDeviceOps(protocol)
	ops = append(ops, extra...)
	now := nowText()
	_ = p.db.Where("scope_type = ? AND scope_id = ?", scopeType, scopeID).Delete(&model.AssetCapability{}).Error
	seen := map[string]bool{}
	for _, op := range ops {
		if seen[op] {
			continue
		}
		seen[op] = true
		if err := p.db.Create(&model.AssetCapability{
			ID: stableID("cap", scopeType, scopeID, op), ScopeType: scopeType, ScopeID: scopeID,
			Operation: op, Supported: true, Configured: true, Available: online,
			Source: "backfill", UpdateTime: now,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (p *AssetProjector) upsertEvent(id, deviceID, resourceID, kind, external string, enabled bool) error {
	return p.db.Save(&model.AssetEventSource{
		ID: id, DeviceID: deviceID, ResourceID: resourceID, Kind: kind,
		ExternalKey: external, Enabled: enabled,
	}).Error
}

func (p *AssetProjector) legacyID(table, legacyID string) (string, error) {
	var ref model.UnifiedRef
	err := p.db.Where("legacy_table = ? AND legacy_id = ?", table, legacyID).First(&ref).Error
	if err == gorm.ErrRecordNotFound {
		return uuid.NewString(), nil
	}
	if err != nil {
		return "", err
	}
	return ref.UnifiedID, nil
}

func (p *AssetProjector) upsertRef(table, legacyID, entityType, unifiedID string) error {
	var ref model.UnifiedRef
	err := p.db.Where("legacy_table = ? AND legacy_id = ?", table, legacyID).First(&ref).Error
	if err == gorm.ErrRecordNotFound {
		return p.db.Create(&model.UnifiedRef{
			LegacyTable: table, LegacyID: legacyID, EntityType: entityType, UnifiedID: unifiedID,
		}).Error
	}
	if err != nil {
		return err
	}
	return p.db.Model(&model.UnifiedRef{}).Where("id = ?", ref.ID).
		Updates(map[string]any{"entity_type": entityType, "unified_id": unifiedID}).Error
}

func (p *AssetProjector) getBinding(deviceID string) (*model.AccessBinding, error) {
	var b model.AccessBinding
	if err := p.db.Where("device_id = ?", deviceID).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func stableID(parts ...string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(strings.Join(parts, "|"))).String()
}

func protocolDeviceOps(protocol string) []string {
	switch protocol {
	case "gb28181":
		return []string{"play", "playback", "ptz", "snapshot", "alarm", "subscribe", "guard", "transport"}
	case "onvif":
		return []string{"play", "ptz", "snapshot", "sync"}
	case "plugin":
		return []string{"play", "ptz", "sync"}
	default:
		return []string{"play"}
	}
}

func channelCaps(ptzType int, hasAudio bool) []string {
	ops := []string{"play", "snapshot"}
	if ptzType > 0 {
		ops = append(ops, "ptz")
	}
	if hasAudio {
		ops = append(ops, "audio")
	}
	return ops
}

func inferProfileRole(label string, index int) string {
	l := strings.ToLower(label)
	switch {
	case strings.Contains(l, "main") || strings.Contains(label, "主"):
		return domainasset.ProfileMain
	case strings.Contains(l, "sub") || strings.Contains(label, "子"):
		return domainasset.ProfileSub
	case strings.Contains(l, "third") || strings.Contains(label, "三"):
		return domainasset.ProfileThird
	}
	if index == 0 {
		return domainasset.ProfileMain
	}
	return domainasset.ProfileCustom
}

func onlineStatus(online bool) string {
	if online {
		return "online"
	}
	return "offline"
}
