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
	// 同一统一对象可能因旧表重建产生多条映射（旧 ID + 新 ID）；
	// 只要还有一条映射对应的 legacy 行存在，就不能删除统一对象。
	liveUnified := map[string]bool{}
	for i := range refs {
		r := &refs[i]
		if set, tracked := sets[r.LegacyTable]; tracked && set[r.LegacyID] {
			liveUnified[r.UnifiedID] = true
		}
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
		// 孤儿：按实体类型级联删除统一资产（仍有其它存活映射时只删映射）。
		if !liveUnified[r.UnifiedID] {
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
			chUID, err := p.resourceLegacyID("zws_device_channel", strconv.Itoa(ch.ID), ch.InternalCode)
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
				"longitude": ch.GBLongitude, "latitude": ch.GBLatitude, "mapLevel": ch.MapLevel,
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
			chUID, err := p.resourceLegacyID("zws_onvif_channel", strconv.FormatInt(ch.ID, 10), ch.InternalCode)
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
	// 每插件声明能力缓存：从 zws_plugin.capabilities 读取（JSON 数组），nil=未知。
	capCache := map[string]map[string]bool{}
	custCache := map[string][]string{}
	for i := range devices {
		dev := &devices[i]
		declared := p.pluginCapsCached(capCache, dev.PluginID)
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
		if err := p.upsertCapabilitiesWith("device", uid, "plugin", dev.Online, nil, func(op string) bool {
			return pluginOpSupported(declared, op)
		}); err != nil {
			return err
		}
		// 自定义能力：登记到设备作用域能力表（Source=custom），供 UI 展示与运行时熔断。
		for _, ccID := range p.pluginCustomCapIDs(custCache, dev.PluginID) {
			if err := p.db.Create(&model.AssetCapability{
				ID: stableID("cap", "device", uid, ccID), ScopeType: "device", ScopeID: uid,
				Operation: ccID, Supported: true, Configured: true, Available: dev.Online,
				Source: "custom", UpdateTime: nowText(),
			}).Error; err != nil {
				return err
			}
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
			chUID, err := p.resourceLegacyID("zws_plugin_channel", strconv.Itoa(ch.ID), ch.InternalCode)
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
			if err := p.upsertCapabilitiesWith("resource", chUID, "plugin", dev.Online, channelCaps(ch.PTZType, ch.HasAudio), func(op string) bool {
				return pluginOpSupported(declared, op)
			}); err != nil {
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
			ID: stableID("binding", deviceID), DeviceID: deviceID, Protocol: protocol, Mode: mode,
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
	attrs = mergeResourceAttrs(existing.AttrsJSON, attrs)
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
	return p.upsertCapabilitiesWith(scopeType, scopeID, protocol, online, extra, nil)
}

// upsertCapabilitiesWith 按 supportedOf 决定每个操作是否"声明支持"（声明驱动）。
// supportedOf 为 nil 时全部视为支持（保持 GB/ONVIF 旧行为）。
// 声明不支持的操作显式写 Supported=false，使调用前即可拒绝（不再靠运行时失败兜底）。
func (p *AssetProjector) upsertCapabilitiesWith(scopeType, scopeID, protocol string, online bool, extra []string, supportedOf func(op string) bool) error {
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
		supported := true
		if supportedOf != nil {
			supported = supportedOf(op)
		}
		if err := p.db.Create(&model.AssetCapability{
			ID: stableID("cap", scopeType, scopeID, op), ScopeType: scopeType, ScopeID: scopeID,
			Operation: op, Supported: supported, Configured: supported, Available: online && supported,
			Source: "backfill", UpdateTime: now,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

// pluginCapsCached 读插件实例声明的能力集合（zws_plugin.capabilities，JSON 数组）。
// 返回 nil 表示未知（无插件行或能力为空），调用方据此维持宽松（legacy）行为。
func (p *AssetProjector) pluginCapsCached(cache map[string]map[string]bool, pluginInstanceID string) map[string]bool {
	key := strings.TrimSpace(pluginInstanceID)
	if key == "" {
		return nil
	}
	if v, ok := cache[key]; ok {
		return v
	}
	var raw string
	_ = p.db.Model(&model.Plugin{}).Where("id = ?", key).Pluck("capabilities", &raw).Error
	var set map[string]bool
	if strings.TrimSpace(raw) != "" {
		var caps []string
		if json.Unmarshal([]byte(raw), &caps) == nil && len(caps) > 0 {
			set = map[string]bool{}
			for _, c := range caps {
				set[strings.ToLower(strings.TrimSpace(c))] = true
			}
		}
	}
	cache[key] = set
	return set
}

// pluginCustomCapIDs 读插件实例自定义能力 id 列表：
// zws_plugin(plugin_id,version) → zws_plugin_package.manifest_json.custom_capabilities[].id。
// 无安装包（手工部署）或解析失败时返回空。
func (p *AssetProjector) pluginCustomCapIDs(cache map[string][]string, pluginInstanceID string) []string {
	key := strings.TrimSpace(pluginInstanceID)
	if key == "" {
		return nil
	}
	if v, ok := cache[key]; ok {
		return v
	}
	var row struct {
		PluginID string
		Version  string
	}
	_ = p.db.Model(&model.Plugin{}).Select("plugin_id, version").Where("id = ?", key).Scan(&row).Error
	var ids []string
	if strings.TrimSpace(row.PluginID) != "" {
		var mj string
		_ = p.db.Model(&model.PluginPackage{}).
			Where("plugin_id = ? AND version = ?", row.PluginID, row.Version).
			Pluck("manifest_json", &mj).Error
		if strings.TrimSpace(mj) != "" {
			var mf struct {
				CustomCapabilities []struct {
					ID string `json:"id"`
				} `json:"custom_capabilities"`
			}
			if json.Unmarshal([]byte(mj), &mf) == nil {
				for _, cc := range mf.CustomCapabilities {
					if id := strings.TrimSpace(cc.ID); id != "" {
						ids = append(ids, id)
					}
				}
			}
		}
	}
	cache[key] = ids
	return ids
}

// pluginOpSupported 把插件清单能力（live/ptz/playback/arm…）映射到平台能力操作。
// 决定该操作是否声明支持。declared==nil 时维持宽松（存量插件未上报能力）。
func pluginOpSupported(declared map[string]bool, op string) bool {
	if declared == nil {
		return true
	}
	switch op {
	case "play":
		return declared["live"]
	case "playback":
		return declared["playback"] || declared["record_query"]
	case "ptz":
		return declared["ptz"]
	case "snapshot":
		return declared["snapshot"]
	case "alarm", "subscribe":
		return declared["arm"] || declared["disarm"]
	case "audio":
		return declared["audio"]
	case "config":
		return declared["config"]
	case "sync":
		return true
	default:
		return true
	}
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
		// 映射丢失时使用确定性 ID，避免同一 legacy 记录在重复投影中生成不同 UUID。
		return stableID("legacy", table, legacyID), nil
	}
	if err != nil {
		return "", err
	}
	return ref.UnifiedID, nil
}

// resourceLegacyID 优先按旧 ID 映射；旧表重建导致 ID 变化时，用通道内码找回原统一资源，
// 避免协议同步把用户改名/编辑数据投影成一条新资源。
func (p *AssetProjector) resourceLegacyID(table, legacyID, internalCode string) (string, error) {
	var ref model.UnifiedRef
	err := p.db.Where("legacy_table = ? AND legacy_id = ?", table, legacyID).First(&ref).Error
	if err == nil {
		return ref.UnifiedID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return "", err
	}
	code := strings.TrimSpace(internalCode)
	if code != "" {
		var res model.AssetResource
		if e := p.db.Where("internal_code = ?", code).First(&res).Error; e == nil {
			_ = p.upsertRef(table, legacyID, "resource", res.ID)
			return res.ID, nil
		}
	}
	return stableID("legacy", table, legacyID), nil
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

// mergeResourceAttrs 保留用户在统一资产上编辑的字段，协议同步只更新协议字段。
func mergeResourceAttrs(existing, incoming string) string {
	if strings.TrimSpace(existing) == "" {
		return incoming
	}
	var oldAttrs, newAttrs map[string]any
	if json.Unmarshal([]byte(existing), &oldAttrs) != nil || json.Unmarshal([]byte(incoming), &newAttrs) != nil {
		return incoming
	}
	if edited, _ := oldAttrs["userEdited"].(bool); !edited {
		return incoming
	}
	for _, key := range []string{"longitude", "latitude", "mapLevel", "tags", "enabled"} {
		if v, ok := oldAttrs[key]; ok {
			newAttrs[key] = v
		}
	}
	newAttrs["userEdited"] = true
	raw, err := json.Marshal(newAttrs)
	if err != nil {
		return incoming
	}
	return string(raw)
}

func protocolDeviceOps(protocol string) []string {
	switch protocol {
	case "gb28181":
		return []string{"play", "playback", "ptz", "snapshot", "alarm", "subscribe", "guard", "transport"}
	case "onvif":
		return []string{"play", "ptz", "snapshot", "sync"}
	case "plugin":
		return []string{"play", "playback", "ptz", "snapshot", "alarm", "subscribe", "sync"}
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
