package persistence

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	domainchannel "zero-web-server/internal/domain/channel"
	domaindevice "zero-web-server/internal/domain/device"
	"zero-web-server/internal/domain/shared"
	"zero-web-server/internal/infrastructure/persistence/model"
	"zero-web-server/pkg/idcode"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ===========================================================================
// GB28181 设备 / 通道持久化（统一模型）
//
// 设备身份       zws_device
// 接入参数(GB)   zws_access_binding.config_json
// 通道身份       zws_channel
// 通道扩展(GB)   zws_channel_ext（经纬度/PTZ/有音频 + gb_json 承载其余国标属性）
// 目录多挂载     zws_catalog（由 region/group 应用服务使用）
//
// domain 层仍是旧的扁平结构，边界处在此拼装/拆解。行政区划/业务分组关联
// 过渡期写入 channel_ext.gb_json（CivilCode/ParentID/BusinessGroupID），
// 保持既有查询语义；目录树的多挂载走 zws_catalog。
// ===========================================================================

// gbBindingConfig 是 GB 设备接入参数在 access_binding.config_json 中的结构。
type gbBindingConfig struct {
	IP                               string `json:"ip,omitempty"`
	Port                             int    `json:"port,omitempty"`
	Transport                        string `json:"transport,omitempty"`
	StreamMode                       string `json:"streamMode,omitempty"`
	Expires                          int    `json:"expires,omitempty"`
	HostAddress                      string `json:"hostAddress,omitempty"`
	Charset                          string `json:"charset,omitempty"`
	MediaServerID                    string `json:"mediaServerId,omitempty"`
	CustomName                       string `json:"customName,omitempty"`
	SDPIP                            string `json:"sdpIp,omitempty"`
	LocalIP                          string `json:"localIp,omitempty"`
	Password                         string `json:"password,omitempty"`
	HeartBeatInterval                int    `json:"heartBeatInterval,omitempty"`
	HeartBeatCount                   int    `json:"heartBeatCount,omitempty"`
	SubscribeCycleForCatalog         int    `json:"subscribeCycleForCatalog,omitempty"`
	SubscribeCycleForMobilePosition  int    `json:"subscribeCycleForMobilePosition,omitempty"`
	MobilePositionSubmissionInterval int    `json:"mobilePositionSubmissionInterval,omitempty"`
	SubscribeCycleForAlarm           int    `json:"subscribeCycleForAlarm,omitempty"`
	ServerID                         string `json:"serverId,omitempty"`
}

// gbChannelJSON 是 GB 通道的国标属性，落在 channel_ext.gb_json。
type gbChannelJSON struct {
	Manufacturer         string  `json:"manufacturer,omitempty"`
	Model                string  `json:"model,omitempty"`
	Parental             int     `json:"parental,omitempty"`
	ParentID             string  `json:"parentId,omitempty"`
	BusinessGroupID      string  `json:"businessGroupId,omitempty"`
	CivilCode            string  `json:"civilCode,omitempty"`
	SubCount             int     `json:"subCount,omitempty"`
	StreamIdentification string  `json:"streamIdentification,omitempty"`
	DataType             int     `json:"dataType,omitempty"`
	Owner                string  `json:"owner,omitempty"`
	Block                string  `json:"block,omitempty"`
	Address              string  `json:"address,omitempty"`
	GBName               string  `json:"gbName,omitempty"`
	GBStatus             string  `json:"gbStatus,omitempty"`
	GBLongitude          float64 `json:"gbLongitude,omitempty"`
	GBLatitude           float64 `json:"gbLatitude,omitempty"`
	GBPTZType            int     `json:"gbPtzType,omitempty"`
}

// ===========================================================================
// DeviceRepository
// ===========================================================================

type DeviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) GetByDeviceID(deviceID string) (*domaindevice.Device, error) {
	var m model.Device
	if err := r.db.Where("gb_code = ?", deviceID).First(&m).Error; err != nil {
		return nil, err
	}
	return r.hydrateDevice(&m)
}

func (r *DeviceRepository) GetByID(id int) (*domaindevice.Device, error) {
	var m model.Device
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return r.hydrateDevice(&m)
}

func (r *DeviceRepository) GetByInternalCode(code string) (*domaindevice.Device, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var m model.Device
	if err := r.db.Where("internal_code = ?", code).First(&m).Error; err != nil {
		return nil, err
	}
	return r.hydrateDevice(&m)
}

// hydrateDevice 拼装设备：身份(zws_device) + 接入参数(zws_access_binding.config_json)。
func (r *DeviceRepository) hydrateDevice(m *model.Device) (*domaindevice.Device, error) {
	d := &domaindevice.Device{
		ID: int(m.ID), InternalCode: m.InternalCode, DeviceID: m.GbCode,
		Name: m.Name, DeviceType: m.DeviceType, Manufacturer: m.Manufacturer, Model: m.Model, Firmware: m.Firmware,
		OnLine: m.Online, CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
	var b model.AccessBinding
	if err := r.db.Where("device_id = ?", m.ID).First(&b).Error; err == nil {
		var cfg gbBindingConfig
		if strings.TrimSpace(b.ConfigJSON) != "" {
			_ = json.Unmarshal([]byte(b.ConfigJSON), &cfg)
		}
		d.IP = cfg.IP
		d.Port = cfg.Port
		d.Transport = cfg.Transport
		d.StreamMode = cfg.StreamMode
		d.Expires = cfg.Expires
		d.HostAddress = cfg.HostAddress
		d.Charset = cfg.Charset
		d.MediaServerID = cfg.MediaServerID
		d.CustomName = cfg.CustomName
		d.SDPIP = cfg.SDPIP
		d.LocalIP = cfg.LocalIP
		d.Password = cfg.Password
		d.HeartBeatInterval = cfg.HeartBeatInterval
		d.HeartBeatCount = cfg.HeartBeatCount
		d.SubscribeCycleForCatalog = cfg.SubscribeCycleForCatalog
		d.SubscribeCycleForMobilePosition = cfg.SubscribeCycleForMobilePosition
		d.MobilePositionSubmissionInterval = cfg.MobilePositionSubmissionInterval
		d.SubscribeCycleForAlarm = cfg.SubscribeCycleForAlarm
		d.ServerID = cfg.ServerID
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return d, nil
}

func (r *DeviceRepository) Create(device *domaindevice.Device) error {
	if strings.TrimSpace(device.InternalCode) == "" {
		code, err := idcode.Device()
		if err != nil {
			return err
		}
		device.InternalCode = code
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	m := &model.Device{
		InternalCode: device.InternalCode, GbCode: device.DeviceID, Name: device.Name,
		DeviceType: device.DeviceType, Manufacturer: device.Manufacturer, Model: device.Model, Firmware: device.Firmware,
		Online: device.OnLine, Status: onlineStatus(device.OnLine),
		CreateTime: now, UpdateTime: now,
	}
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	device.ID = int(m.ID)
	device.CreateTime, device.UpdateTime = now, now
	return r.upsertBinding(m.ID, device, now)
}

func (r *DeviceRepository) Update(device *domaindevice.Device) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	m := &model.Device{
		ID: int64(device.ID), InternalCode: device.InternalCode, GbCode: device.DeviceID,
		Name: device.Name, DeviceType: device.DeviceType, Manufacturer: device.Manufacturer, Model: device.Model,
		Firmware: device.Firmware, Online: device.OnLine, Status: onlineStatus(device.OnLine),
		CreateTime: device.CreateTime, UpdateTime: now,
	}
	if err := r.db.Save(m).Error; err != nil {
		return err
	}
	return r.upsertBinding(m.ID, device, now)
}

func (r *DeviceRepository) UpdateOnline(deviceID string, online bool) error {
	return r.db.Model(&model.Device{}).Where("gb_code = ?", deviceID).
		Updates(map[string]any{"online": online, "status": onlineStatus(online), "update_time": nowTimeStr()}).Error
}

func (r *DeviceRepository) DeleteByDeviceID(deviceID string) error {
	// zws_access_binding / zws_channel 通过外键级联删除；此处删设备即可。
	return r.db.Where("gb_code = ?", deviceID).Delete(&model.Device{}).Error
}

// UpdateFields 仅更新给定列：身份列写 zws_device，接入参数列写 binding.config_json。
func (r *DeviceRepository) UpdateFields(deviceID string, fields map[string]interface{}) error {
	var dev model.Device
	if err := r.db.Where("gb_code = ?", deviceID).First(&dev).Error; err != nil {
		return err
	}
	deviceCols := map[string]any{}
	cfgUpdates := map[string]any{}
	for k, v := range fields {
		switch k {
		case "name", "manufacturer", "model", "firmware", "update_time", "on_line":
			deviceCols[k] = v
		case "ip", "port", "host_address", "expires", "transport", "local_ip", "password",
			"charset", "media_server_id", "custom_name", "sdp_ip", "server_id",
			"heart_beat_interval", "heart_beat_count", "subscribe_cycle_for_catalog",
			"subscribe_cycle_for_mobile_position", "mobile_position_submission_interval",
			"subscribe_cycle_for_alarm", "stream_mode":
			cfgUpdates[k] = v
		default:
			// 未知列忽略，避免写入不存在的物理列。
		}
	}
	if len(deviceCols) > 0 {
		if err := r.db.Model(&model.Device{}).Where("id = ?", dev.ID).Updates(deviceCols).Error; err != nil {
			return err
		}
	}
	if len(cfgUpdates) > 0 {
		return r.patchBindingConfig(dev.ID, cfgUpdates)
	}
	return nil
}

func (r *DeviceRepository) List(page, count int, query string, online *bool) ([]*domaindevice.Device, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Table("zws_device de").
		Select(`de.*, (SELECT COUNT(0) FROM zws_channel c WHERE c.device_id = de.id) AS channel_count`).
		Where("de.id IN (SELECT device_id FROM zws_access_binding WHERE protocol = ?)", "gb28181")
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("de.gb_code LIKE ? OR de.internal_code LIKE ? OR de.name LIKE ?", like, like, like)
	}
	if online != nil {
		q = q.Where("de.online = ?", *online)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type deviceRow struct {
		model.Device
		ChannelCount int `gorm:"column:channel_count"`
	}
	var rows []deviceRow
	if err := q.Order("de.id DESC").Offset((page - 1) * count).Limit(count).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	list := make([]*domaindevice.Device, 0, len(rows))
	for i := range rows {
		d, err := r.hydrateDevice(&rows[i].Device)
		if err != nil {
			return nil, 0, err
		}
		d.ChannelCount = rows[i].ChannelCount
		list = append(list, d)
	}
	return list, total, nil
}

func (r *DeviceRepository) ListOnline() ([]*domaindevice.Device, error) {
	var rows []model.Device
	if err := r.db.Where("online = ?", true).
		Where("id IN (SELECT device_id FROM zws_access_binding WHERE protocol = ?)", "gb28181").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*domaindevice.Device, 0, len(rows))
	for i := range rows {
		d, err := r.hydrateDevice(&rows[i])
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *DeviceRepository) upsertBinding(deviceID int64, d *domaindevice.Device, now string) error {
	cfg := gbBindingConfig{
		IP: d.IP, Port: d.Port, Transport: d.Transport, StreamMode: d.StreamMode,
		Expires: d.Expires, HostAddress: d.HostAddress, Charset: d.Charset,
		MediaServerID: d.MediaServerID, CustomName: d.CustomName, SDPIP: d.SDPIP,
		LocalIP: d.LocalIP, Password: d.Password, HeartBeatInterval: d.HeartBeatInterval,
		HeartBeatCount: d.HeartBeatCount, SubscribeCycleForCatalog: d.SubscribeCycleForCatalog,
		SubscribeCycleForMobilePosition: d.SubscribeCycleForMobilePosition,
		MobilePositionSubmissionInterval: d.MobilePositionSubmissionInterval,
		SubscribeCycleForAlarm:           d.SubscribeCycleForAlarm, ServerID: d.ServerID,
	}
	raw, _ := json.Marshal(cfg)
	var existing model.AccessBinding
	err := r.db.Where("device_id = ?", deviceID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(&model.AccessBinding{
			DeviceID: deviceID, Protocol: "gb28181", Mode: "passive",
			ExternalID: d.DeviceID, Endpoint: endpointOf(d.IP, d.Port),
			ConfigJSON: string(raw), Status: onlineStatus(d.OnLine),
			CreateTime: now, UpdateTime: now,
		}).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&model.AccessBinding{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"external_id": d.DeviceID, "endpoint": endpointOf(d.IP, d.Port),
		"config_json": string(raw), "status": onlineStatus(d.OnLine), "update_time": now,
	}).Error
}

func (r *DeviceRepository) patchBindingConfig(deviceID int64, updates map[string]any) error {
	var b model.AccessBinding
	if err := r.db.Where("device_id = ?", deviceID).First(&b).Error; err != nil {
		return err
	}
	var cfg gbBindingConfig
	if strings.TrimSpace(b.ConfigJSON) != "" {
		_ = json.Unmarshal([]byte(b.ConfigJSON), &cfg)
	}
	for k, v := range updates {
		switch k {
		case "ip":
			cfg.IP = fmt.Sprint(v)
		case "port":
			cfg.Port = toInt(v)
		case "host_address":
			cfg.HostAddress = fmt.Sprint(v)
		case "expires":
			cfg.Expires = toInt(v)
		case "transport":
			cfg.Transport = fmt.Sprint(v)
		case "local_ip":
			cfg.LocalIP = fmt.Sprint(v)
		case "password":
			cfg.Password = fmt.Sprint(v)
		case "charset":
			cfg.Charset = fmt.Sprint(v)
		case "media_server_id":
			cfg.MediaServerID = fmt.Sprint(v)
		case "custom_name":
			cfg.CustomName = fmt.Sprint(v)
		case "sdp_ip":
			cfg.SDPIP = fmt.Sprint(v)
		case "server_id":
			cfg.ServerID = fmt.Sprint(v)
		case "stream_mode":
			cfg.StreamMode = fmt.Sprint(v)
		case "heart_beat_interval":
			cfg.HeartBeatInterval = toInt(v)
		case "heart_beat_count":
			cfg.HeartBeatCount = toInt(v)
		case "subscribe_cycle_for_catalog":
			cfg.SubscribeCycleForCatalog = toInt(v)
		case "subscribe_cycle_for_mobile_position":
			cfg.SubscribeCycleForMobilePosition = toInt(v)
		case "mobile_position_submission_interval":
			cfg.MobilePositionSubmissionInterval = toInt(v)
		case "subscribe_cycle_for_alarm":
			cfg.SubscribeCycleForAlarm = toInt(v)
		}
	}
	raw, _ := json.Marshal(cfg)
	return r.db.Model(&model.AccessBinding{}).Where("id = ?", b.ID).
		Updates(map[string]any{"config_json": string(raw), "update_time": nowTimeStr()}).Error
}

func endpointOf(ip string, port int) string {
	if strings.TrimSpace(ip) == "" {
		return ""
	}
	if port > 0 {
		return fmt.Sprintf("%s:%d", ip, port)
	}
	return ip
}

func toInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case string:
		var n int
		fmt.Sscanf(x, "%d", &n)
		return n
	default:
		return 0
	}
}

func nowTimeStr() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// ===========================================================================
// ChannelRepository
// ===========================================================================

type ChannelRepository struct {
	db *gorm.DB
}

func NewChannelRepository(db *gorm.DB) *ChannelRepository {
	return &ChannelRepository{db: db}
}

var _ domainchannel.Repository = (*ChannelRepository)(nil)

func (r *ChannelRepository) GetOne(deviceID, channelDeviceID string) (*domainchannel.Channel, error) {
	var m model.Channel
	err := r.db.Where("device_id = (SELECT id FROM zws_device WHERE gb_code = ? LIMIT 1) AND (gb_code = ? OR external_key = ?)",
		deviceID, channelDeviceID, channelDeviceID).First(&m).Error
	if err != nil {
		return nil, err
	}
	return r.hydrateChannel(&m)
}

func (r *ChannelRepository) GetByID(id int) (*domainchannel.Channel, error) {
	var m model.Channel
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return r.hydrateChannel(&m)
}

func (r *ChannelRepository) GetByInternalCode(code string) (*domainchannel.Channel, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var m model.Channel
	if err := r.db.Where("internal_code = ?", code).First(&m).Error; err != nil {
		return nil, err
	}
	return r.hydrateChannel(&m)
}

// hydrateChannel 拼装通道：身份(zws_channel) + 扩展(zws_channel_ext) + 国标属性(ext.gb_json)。
func (r *ChannelRepository) hydrateChannel(m *model.Channel) (*domainchannel.Channel, error) {
	ch := &domainchannel.Channel{
		ID: int(m.ID), InternalCode: m.InternalCode, DataDeviceID: int(m.DeviceID),
		GBDeviceID: strings.TrimSpace(m.GbCode), Name: m.Name, Status: strings.TrimSpace(m.Status),
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
	var parentGb string
	_ = r.db.Table("zws_device").Select("gb_code").Where("id = ?", m.DeviceID).Scan(&parentGb).Error
	ch.DeviceID = parentGb

	var ext model.ChannelExt
	err := r.db.Where("channel_id = ?", m.ID).First(&ext).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	ch.Longitude = ext.Longitude
	ch.Latitude = ext.Latitude
	ch.PTZType = ext.PTZType
	ch.HasAudio = ext.HasAudio
	if strings.TrimSpace(ext.GBJSON) != "" {
		var g gbChannelJSON
		if json.Unmarshal([]byte(ext.GBJSON), &g) == nil {
			ch.Manufacturer = g.Manufacturer
			ch.Model = g.Model
			ch.Parental = g.Parental
			ch.ParentID = g.ParentID
			ch.BusinessGroupID = g.BusinessGroupID
			ch.CivilCode = g.CivilCode
			ch.SubCount = g.SubCount
			ch.StreamIdentification = g.StreamIdentification
			ch.DataType = g.DataType
			if strings.TrimSpace(g.GBName) != "" {
				ch.Name = g.GBName
			}
			if strings.TrimSpace(g.GBStatus) != "" {
				ch.Status = g.GBStatus
			}
			if g.GBLongitude != 0 {
				ch.Longitude = g.GBLongitude
			}
			if g.GBLatitude != 0 {
				ch.Latitude = g.GBLatitude
			}
			if g.GBPTZType > 0 {
				ch.PTZType = g.GBPTZType
			}
		}
	}
	return ch, nil
}

func (r *ChannelRepository) ListByDevice(deviceID string, page, count int, query string, online *bool) ([]*domainchannel.Channel, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.Channel{}).
		Where("device_id = (SELECT id FROM zws_device WHERE gb_code = ? LIMIT 1)", deviceID)
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("name LIKE ? OR gb_code LIKE ? OR internal_code LIKE ?", like, like, like)
	}
	if online != nil {
		if *online {
			q = q.Where("status = ?", "ON")
		} else {
			q = q.Where("status <> ?", "ON")
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Channel
	if err := q.Order("id ASC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	list := make([]*domainchannel.Channel, 0, len(rows))
	for i := range rows {
		ch, err := r.hydrateChannel(&rows[i])
		if err != nil {
			return nil, 0, err
		}
		list = append(list, ch)
	}
	return list, total, nil
}

func (r *ChannelRepository) ListAll(page, count int, query string) ([]*domainchannel.Channel, int64, error) {
	return r.listChannels(page, count, query, nil, nil, "")
}

func (r *ChannelRepository) ListCommon(page, count int, query string, channelType *int, online *bool, hasRecordPlan *bool) ([]*domainchannel.Channel, int64, error) {
	return r.listChannels(page, count, query, online, hasRecordPlan, "")
}

// listChannels 通用通道列表：query 过滤 + online + 是否有录像计划。
func (r *ChannelRepository) listChannels(page, count int, query string, online *bool, hasRecordPlan *bool, civilCode string) ([]*domainchannel.Channel, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Table("zws_channel c").Joins("LEFT JOIN zws_channel_ext e ON e.channel_id = c.id")
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("c.name LIKE ? OR c.gb_code LIKE ? OR c.internal_code LIKE ?", like, like, like)
	}
	if online != nil {
		if *online {
			q = q.Where("c.status = ?", "ON")
		} else {
			q = q.Where("c.status <> ?", "ON")
		}
	}
	if hasRecordPlan != nil {
		if *hasRecordPlan {
			q = q.Where("e.record_plan_id > 0")
		} else {
			q = q.Where("(e.record_plan_id = 0 OR e.record_plan_id IS NULL)")
		}
	}
	if civilCode != "" {
		q = q.Where("JSON_UNQUOTE(JSON_EXTRACT(e.gb_json, '$.civilCode')) = ?", civilCode)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Channel
	if err := q.Select("c.*").Order("c.id ASC").Offset((page - 1) * count).Limit(count).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	list := make([]*domainchannel.Channel, 0, len(rows))
	for i := range rows {
		ch, err := r.hydrateChannel(&rows[i])
		if err != nil {
			return nil, 0, err
		}
		list = append(list, ch)
	}
	return list, total, nil
}

func (r *ChannelRepository) ListByCivilCode(page, count int, query string, channelType *int, online *bool, civilCode string) ([]*domainchannel.Channel, int64, error) {
	return r.listChannels(page, count, query, online, nil, civilCode)
}

func (r *ChannelRepository) ListByGroupParent(page, count int, query string, channelType *int, online *bool, groupDeviceID string) ([]*domainchannel.Channel, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Table("zws_channel c").Joins("LEFT JOIN zws_channel_ext e ON e.channel_id = c.id")
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("c.name LIKE ? OR c.gb_code LIKE ? OR c.internal_code LIKE ?", like, like, like)
	}
	if online != nil {
		if *online {
			q = q.Where("c.status = ?", "ON")
		} else {
			q = q.Where("c.status <> ?", "ON")
		}
	}
	if groupDeviceID == "" {
		q = q.Where("(JSON_EXTRACT(e.gb_json, '$.parentId') IS NULL OR JSON_UNQUOTE(JSON_EXTRACT(e.gb_json, '$.parentId')) = '')")
	} else {
		q = q.Where("JSON_UNQUOTE(JSON_EXTRACT(e.gb_json, '$.parentId')) = ?", groupDeviceID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Channel
	if err := q.Select("c.*").Order("c.id ASC").Offset((page - 1) * count).Limit(count).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	list := make([]*domainchannel.Channel, 0, len(rows))
	for i := range rows {
		ch, err := r.hydrateChannel(&rows[i])
		if err != nil {
			return nil, 0, err
		}
		list = append(list, ch)
	}
	return list, total, nil
}

func (r *ChannelRepository) ResetByDevice(deviceID string, dataDeviceID int, channels []*domainchannel.Channel) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var dev model.Device
		if err := tx.Where("gb_code = ?", deviceID).First(&dev).Error; err != nil {
			return err
		}
		var existing []model.Channel
		_ = tx.Where("device_id = ?", dev.ID).Find(&existing).Error
		keep := make(map[string]string, len(existing))
		keepName := make(map[string]string, len(existing))
		for _, row := range existing {
			gb := strings.TrimSpace(row.GbCode)
			if gb == "" {
				continue
			}
			if strings.TrimSpace(row.InternalCode) != "" {
				keep[gb] = row.InternalCode
			}
			if strings.TrimSpace(row.Name) != "" {
				keepName[gb] = row.Name
			}
		}
		deviceName := dev.Name
		var cfg gbBindingConfig
		var b model.AccessBinding
		if err := tx.Where("device_id = ?", dev.ID).First(&b).Error; err == nil && strings.TrimSpace(b.ConfigJSON) != "" {
			_ = json.Unmarshal([]byte(b.ConfigJSON), &cfg)
			if strings.TrimSpace(cfg.CustomName) != "" {
				deviceName = cfg.CustomName
			}
		}
		if err := tx.Where("device_id = ?", dev.ID).Delete(&model.Channel{}).Error; err != nil {
			return err
		}
		if len(channels) == 0 {
			return nil
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		for i, ch := range channels {
			code := strings.TrimSpace(ch.InternalCode)
			if code == "" {
				if prev, ok := keep[strings.TrimSpace(ch.GBDeviceID)]; ok {
					code = prev
				} else {
					var err error
					code, err = idcode.Channel()
					if err != nil {
						return err
					}
				}
			}
			ch.InternalCode = code
			name := keepName[strings.TrimSpace(ch.GBDeviceID)]
			if strings.TrimSpace(name) == "" {
				name = shared.DefaultChannelName(deviceName, i+1)
			}
			m := &model.Channel{
				DeviceID: dev.ID, InternalCode: code, GbCode: ch.GBDeviceID,
				ExternalKey: ch.GBDeviceID, Kind: "video_sensor", Name: name,
				Online: ch.Status == "ON", Status: ch.Status, CreateTime: now, UpdateTime: now,
			}
			if err := tx.Create(m).Error; err != nil {
				return err
			}
			g := gbChannelJSON{
				Manufacturer: ch.Manufacturer, Model: ch.Model, Parental: ch.Parental,
				ParentID: ch.ParentID, BusinessGroupID: ch.BusinessGroupID, CivilCode: ch.CivilCode,
				SubCount: ch.SubCount, StreamIdentification: streamIdentificationOrDefault(ch.StreamIdentification),
				DataType: shared.ChannelDataTypeGB28181, GBName: name, GBStatus: ch.Status,
				GBLongitude: ch.Longitude, GBLatitude: ch.Latitude, GBPTZType: ch.PTZType,
			}
			gj, _ := json.Marshal(g)
			if err := tx.Create(&model.ChannelExt{
				ChannelID: m.ID, Longitude: ch.Longitude, Latitude: ch.Latitude,
				PTZType: ch.PTZType, HasAudio: ch.HasAudio, Enabled: true,
				GBJSON: string(gj), CreateTime: now, UpdateTime: now,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ChannelRepository) Update(ch *domainchannel.Channel) error {
	now := nowTimeStr()
	updates := map[string]any{
		"name": ch.Name, "gb_code": ch.GBDeviceID, "status": ch.Status,
		"online": ch.Status == "ON", "update_time": now,
	}
	if err := r.db.Model(&model.Channel{}).Where("id = ?", ch.ID).Updates(updates).Error; err != nil {
		return err
	}
	var ext model.ChannelExt
	err := r.db.Where("channel_id = ?", ch.ID).First(&ext).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		g := gbChannelJSON{
			Manufacturer: ch.Manufacturer, Model: ch.Model, Parental: ch.Parental,
			ParentID: ch.ParentID, BusinessGroupID: ch.BusinessGroupID, CivilCode: ch.CivilCode,
			SubCount: ch.SubCount, StreamIdentification: ch.StreamIdentification,
		}
		gj, _ := json.Marshal(g)
		return r.db.Create(&model.ChannelExt{
			ChannelID: int64(ch.ID), Longitude: ch.Longitude, Latitude: ch.Latitude,
			PTZType: ch.PTZType, HasAudio: ch.HasAudio, Enabled: true, GBJSON: string(gj),
			CreateTime: now, UpdateTime: now,
		}).Error
	}
	if err != nil {
		return err
	}
	var g gbChannelJSON
	if strings.TrimSpace(ext.GBJSON) != "" {
		_ = json.Unmarshal([]byte(ext.GBJSON), &g)
	}
	g.Manufacturer = ch.Manufacturer
	g.Model = ch.Model
	g.Parental = ch.Parental
	g.SubCount = ch.SubCount
	if strings.TrimSpace(ch.ParentID) != "" {
		g.ParentID = ch.ParentID
	}
	if strings.TrimSpace(ch.BusinessGroupID) != "" {
		g.BusinessGroupID = ch.BusinessGroupID
	}
	gj, _ := json.Marshal(g)
	return r.db.Model(&model.ChannelExt{}).Where("id = ?", ext.ID).Updates(map[string]any{
		"longitude": ch.Longitude, "latitude": ch.Latitude, "ptz_type": ch.PTZType,
		"has_audio": ch.HasAudio, "gb_json": string(gj), "update_time": now,
	}).Error
}

func (r *ChannelRepository) ListIDsByDevice(deviceID string) ([]int, error) {
	var ids []int
	err := r.db.Model(&model.Channel{}).
		Where("device_id = (SELECT id FROM zws_device WHERE gb_code = ? LIMIT 1)", deviceID).
		Pluck("id", &ids).Error
	return ids, err
}

func (r *ChannelRepository) ListByDataDeviceIDs(dataDeviceIDs []int) ([]*domainchannel.Channel, error) {
	if len(dataDeviceIDs) == 0 {
		return []*domainchannel.Channel{}, nil
	}
	var rows []model.Channel
	if err := r.db.Where("device_id IN ?", dataDeviceIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainchannel.Channel, 0, len(rows))
	for i := range rows {
		ch, err := r.hydrateChannel(&rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, ch)
	}
	return out, nil
}

func (r *ChannelRepository) DeleteByDevice(deviceID string) error {
	return r.db.Where("device_id = (SELECT id FROM zws_device WHERE gb_code = ? LIMIT 1)", deviceID).
		Delete(&model.Channel{}).Error
}

func (r *ChannelRepository) ChangeAudio(channelID int, audio bool) error {
	return r.db.Model(&model.ChannelExt{}).Where("channel_id = ?", channelID).
		Update("has_audio", audio).Error
}

func (r *ChannelRepository) UpdateStreamIdentification(id int, deviceID string, value string) error {
	q := r.db.Model(&model.Channel{})
	if id > 0 {
		q = q.Where("id = ?", id)
	} else if deviceID != "" {
		q = q.Where("device_id = (SELECT id FROM zws_device WHERE gb_code = ? LIMIT 1)", deviceID)
	} else {
		return nil
	}
	var rows []model.Channel
	if err := q.Select("id").Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := r.patchChannelGB(row.ID, func(g *gbChannelJSON) { g.StreamIdentification = value }); err != nil {
			return err
		}
	}
	return nil
}

// patchChannelGB 读改写 channel_ext.gb_json。
func (r *ChannelRepository) patchChannelGB(channelID int64, fn func(g *gbChannelJSON)) error {
	var ext model.ChannelExt
	err := r.db.Where("channel_id = ?", channelID).First(&ext).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		var g gbChannelJSON
		fn(&g)
		gj, _ := json.Marshal(g)
		return r.db.Create(&model.ChannelExt{ChannelID: channelID, Enabled: true, GBJSON: string(gj), CreateTime: nowTimeStr(), UpdateTime: nowTimeStr()}).Error
	}
	if err != nil {
		return err
	}
	var g gbChannelJSON
	if strings.TrimSpace(ext.GBJSON) != "" {
		_ = json.Unmarshal([]byte(ext.GBJSON), &g)
	}
	fn(&g)
	gj, _ := json.Marshal(g)
	return r.db.Model(&model.ChannelExt{}).Where("id = ?", ext.ID).
		Updates(map[string]any{"gb_json": string(gj), "update_time": nowTimeStr()}).Error
}

func streamIdentificationOrDefault(v string) string {
	if strings.TrimSpace(v) == "" {
		return "stream:0"
	}
	return strings.TrimSpace(v)
}

func (r *ChannelRepository) SetGBDeviceID(channelID int, gbDeviceID string) error {
	return r.db.Model(&model.Channel{}).Where("id = ?", channelID).
		Updates(map[string]any{"gb_code": gbDeviceID, "update_time": nowTimeStr()}).Error
}

func (r *ChannelRepository) CountCommonByIDs(channelIDs []int) (int64, error) {
	if len(channelIDs) == 0 {
		return 0, nil
	}
	var total int64
	err := r.db.Model(&model.Channel{}).Where("id IN ?", channelIDs).Count(&total).Error
	return total, err
}

func (r *ChannelRepository) CountCommonByDataDeviceIDs(dataDeviceIDs []int) (int64, error) {
	if len(dataDeviceIDs) == 0 {
		return 0, nil
	}
	var cnt int64
	err := r.db.Model(&model.Channel{}).Where("device_id IN ?", dataDeviceIDs).Count(&cnt).Error
	return cnt, err
}

// --- 行政区划/业务分组（过渡期写 channel_ext.gb_json）---

func (r *ChannelRepository) SetCivilCode(civilCode string, channelIDs []int) error {
	if len(channelIDs) == 0 {
		return fmt.Errorf("通道ID不可为空")
	}
	for _, id := range channelIDs {
		if err := r.patchChannelGB(int64(id), func(g *gbChannelJSON) { g.CivilCode = civilCode }); err != nil {
			return err
		}
	}
	return nil
}

func (r *ChannelRepository) ClearCivilCode(channelIDs []int) error {
	if len(channelIDs) == 0 {
		return fmt.Errorf("通道ID不可为空")
	}
	return r.SetCivilCode("", channelIDs)
}

func (r *ChannelRepository) SetGroup(parentID, businessGroup string, channelIDs []int) error {
	if len(channelIDs) == 0 {
		return fmt.Errorf("通道ID不可为空")
	}
	for _, id := range channelIDs {
		if err := r.patchChannelGB(int64(id), func(g *gbChannelJSON) {
			g.ParentID = parentID
			g.BusinessGroupID = businessGroup
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *ChannelRepository) ClearGroupParent(channelIDs []int) error {
	if len(channelIDs) == 0 {
		return fmt.Errorf("通道ID不可为空")
	}
	for _, id := range channelIDs {
		if err := r.patchChannelGB(int64(id), func(g *gbChannelJSON) {
			g.ParentID = ""
			g.BusinessGroupID = ""
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *ChannelRepository) SetCivilCodeByDataDeviceIDs(civilCode string, dataDeviceIDs []int) error {
	ids, err := r.channelIDsByDeviceIDs(dataDeviceIDs)
	if err != nil {
		return err
	}
	return r.SetCivilCode(civilCode, ids)
}

func (r *ChannelRepository) ClearCivilCodeByDataDeviceIDs(dataDeviceIDs []int) error {
	ids, err := r.channelIDsByDeviceIDs(dataDeviceIDs)
	if err != nil {
		return err
	}
	return r.ClearCivilCode(ids)
}

func (r *ChannelRepository) SetGroupByDataDeviceIDs(parentID, businessGroup string, dataDeviceIDs []int) error {
	ids, err := r.channelIDsByDeviceIDs(dataDeviceIDs)
	if err != nil {
		return err
	}
	return r.SetGroup(parentID, businessGroup, ids)
}

func (r *ChannelRepository) ClearGroupByDataDeviceIDs(dataDeviceIDs []int) error {
	ids, err := r.channelIDsByDeviceIDs(dataDeviceIDs)
	if err != nil {
		return err
	}
	return r.ClearGroupParent(ids)
}

func (r *ChannelRepository) channelIDsByDeviceIDs(dataDeviceIDs []int) ([]int, error) {
	if len(dataDeviceIDs) == 0 {
		return nil, nil
	}
	var ids []int
	err := r.db.Model(&model.Channel{}).Where("device_id IN ?", dataDeviceIDs).Pluck("id", &ids).Error
	return ids, err
}

// NextGBSeq 按 14 位前缀原子分配下一个 6 位序号（事务 + 行锁）。
func (r *ChannelRepository) NextGBSeq(prefix string) (int, error) {
	seq := 0
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var row model.GBCodeSeq
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("prefix = ?", prefix).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			seq = 1
			return tx.Create(&model.GBCodeSeq{Prefix: prefix, Next: 1}).Error
		}
		if err != nil {
			return err
		}
		seq = int(row.Next) + 1
		return tx.Model(&model.GBCodeSeq{}).Where("prefix = ?", prefix).Update("next", seq).Error
	})
	return seq, err
}
