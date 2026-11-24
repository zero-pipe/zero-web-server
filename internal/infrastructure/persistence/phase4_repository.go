package persistence

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domainalarm "zero-web-server/internal/domain/alarm"
	domainplatform "zero-web-server/internal/domain/platform"
	domainposition "zero-web-server/internal/domain/position"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AlarmRepository struct{ db *gorm.DB }

func NewAlarmRepository(db *gorm.DB) *AlarmRepository { return &AlarmRepository{db: db} }

func (r *AlarmRepository) Create(alarm *domainalarm.Alarm) error {
	m := &model.Alarm{
		EventID: alarm.EventID, ChannelID: int64(alarm.ChannelID), Description: alarm.Description,
		AlarmType: alarm.AlarmType, AlarmCode: alarm.AlarmCode, AlarmTime: alarm.AlarmTime,
		DeviceInternalCode: alarm.DeviceInternalCode, ChannelInternalCode: alarm.ChannelInternalCode,
		Longitude: alarm.Longitude, Latitude: alarm.Latitude, BBoxJSON: alarm.BBoxJSON,
		SnapPath: alarm.SnapPath, RecordPath: alarm.RecordPath,
		Severity: alarm.Severity, Status: alarm.Status, HandedBy: alarm.HandledBy,
		HandledAt: alarm.HandledAt, Remark: alarm.Remark,
		CreateTime: time.Now().Format("2006-01-02 15:04:05"),
	}
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	alarm.ID = int(m.ID)
	return nil
}

func (r *AlarmRepository) GetByID(id int) (*domainalarm.Alarm, error) {
	var m model.Alarm
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return toDomainAlarm(&m), nil
}

func (r *AlarmRepository) List(page, count int, alarmType *int, beginTime, endTime int64) ([]*domainalarm.Alarm, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.Alarm{})
	if alarmType != nil {
		q = q.Where("alarm_type = ?", *alarmType)
	}
	if beginTime > 0 {
		q = q.Where("alarm_time >= ?", beginTime)
	}
	if endTime > 0 {
		q = q.Where("alarm_time <= ?", endTime)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Alarm
	if err := q.Order("alarm_time DESC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	list := make([]*domainalarm.Alarm, 0, len(rows))
	for i := range rows {
		list = append(list, toDomainAlarm(&rows[i]))
	}
	return list, total, nil
}

func (r *AlarmRepository) Delete(ids []int) error {
	return r.db.Where("id IN ?", ids).Delete(&model.Alarm{}).Error
}

func (r *AlarmRepository) Clear(alarmType *int, beginTime, endTime int64) error {
	q := r.db.Where("1=1")
	if alarmType != nil {
		q = q.Where("alarm_type = ?", *alarmType)
	}
	if beginTime > 0 {
		q = q.Where("alarm_time >= ?", beginTime)
	}
	if endTime > 0 {
		q = q.Where("alarm_time <= ?", endTime)
	}
	return q.Delete(&model.Alarm{}).Error
}

func (r *AlarmRepository) UpdateStatus(id int, status, handledBy, remark string, handledAt int64) error {
	return r.db.Model(&model.Alarm{}).Where("id = ?", id).Updates(map[string]any{
		"status": status, "handed_by": handledBy, "remark": remark, "handled_at": handledAt,
	}).Error
}

type PositionRepository struct{ db *gorm.DB }

func NewPositionRepository(db *gorm.DB) *PositionRepository { return &PositionRepository{db: db} }

func (r *PositionRepository) Create(pos *domainposition.MobilePosition) error {
	m := toModelPosition(pos)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	pos.ID = int(m.ID)
	return nil
}

func (r *PositionRepository) BatchCreate(list []*domainposition.MobilePosition) error {
	if len(list) == 0 {
		return nil
	}
	models := make([]model.MobilePosition, 0, len(list))
	for _, p := range list {
		models = append(models, *toModelPosition(p))
	}
	return r.db.Create(&models).Error
}

func (r *PositionRepository) ListByChannel(channelID int, start, end int64) ([]*domainposition.MobilePosition, error) {
	q := r.db.Where("channel_id = ?", channelID)
	if start > 0 {
		q = q.Where("timestamp >= ?", start)
	}
	if end > 0 {
		q = q.Where("timestamp <= ?", end)
	}
	var rows []model.MobilePosition
	if err := q.Order("timestamp DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*domainposition.MobilePosition, 0, len(rows))
	for i := range rows {
		list = append(list, toDomainPosition(&rows[i]))
	}
	return list, nil
}

func (r *PositionRepository) Latest(channelID int) (*domainposition.MobilePosition, error) {
	var m model.MobilePosition
	if err := r.db.Where("channel_id = ?", channelID).Order("timestamp DESC").First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainPosition(&m), nil
}

type PlatformRepository struct{ db *gorm.DB }

func NewPlatformRepository(db *gorm.DB) *PlatformRepository { return &PlatformRepository{db: db} }

func (r *PlatformRepository) GetByID(id int) (*domainplatform.Platform, error) {
	var m model.UpperPlatform
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return toDomainPlatform(&m), nil
}

func (r *PlatformRepository) GetByServerGBID(serverGBID string) (*domainplatform.Platform, error) {
	var m model.UpperPlatform
	if err := r.db.Where("server_gb_id = ?", serverGBID).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainPlatform(&m), nil
}

func (r *PlatformRepository) List(page, count int, query string) ([]*domainplatform.Platform, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.UpperPlatform{})
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("name LIKE ? OR server_gb_id LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.UpperPlatform
	if err := q.Order("id DESC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	list := make([]*domainplatform.Platform, 0, len(rows))
	for i := range rows {
		list = append(list, toDomainPlatform(&rows[i]))
	}
	return list, total, nil
}

func (r *PlatformRepository) Create(p *domainplatform.Platform) error {
	m := toModelPlatform(p)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	p.ID = int(m.ID)
	return nil
}

func (r *PlatformRepository) Update(p *domainplatform.Platform) error {
	return r.db.Save(toModelPlatform(p)).Error
}

func (r *PlatformRepository) Delete(id int) error {
	return r.db.Delete(&model.UpperPlatform{}, id).Error
}

func (r *PlatformRepository) UpdateStatus(id int, online bool) error {
	return r.db.Model(&model.UpperPlatform{}).Where("id = ?", id).Update("status", online).Error
}

func (r *PlatformRepository) ListEnabled() ([]*domainplatform.Platform, error) {
	var rows []model.UpperPlatform
	if err := r.db.Where("enable = ?", true).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*domainplatform.Platform, 0, len(rows))
	for i := range rows {
		list = append(list, toDomainPlatform(&rows[i]))
	}
	return list, nil
}

func (r *PlatformRepository) ListChannelIDsByDevice(deviceID string) ([]int, error) {
	var ids []int
	err := r.db.Table("zws_channel").
		Where("device_id = (SELECT id FROM zws_device WHERE gb_code = ? LIMIT 1)", deviceID).
		Pluck("id", &ids).Error
	return ids, err
}

func (r *PlatformRepository) ListPlatformChannels(platformID int) ([]int, error) {
	var ids []int
	err := r.db.Model(&model.PlatformChannel{}).Where("platform_id = ?", platformID).
		Pluck("channel_id", &ids).Error
	return ids, err
}

func (r *PlatformRepository) AddPlatformChannel(platformID, channelID int) error {
	pc := model.PlatformChannel{PlatformID: int64(platformID), ChannelID: int64(channelID),
		CreateTime: nowTimeStr(), UpdateTime: nowTimeStr()}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&pc).Error
}

func (r *PlatformRepository) RemovePlatformChannel(platformID, channelID int) error {
	return r.db.Where("platform_id = ? AND channel_id = ?", platformID, channelID).
		Delete(&model.PlatformChannel{}).Error
}

func (r *PlatformRepository) GetPlatformChannel(platformID, channelID int) (*domainplatform.Channel, error) {
	var row model.PlatformChannel
	err := r.db.Where("platform_id = ? AND channel_id = ?", platformID, channelID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return toDomainPlatformChannel(&row), nil
}

func (r *PlatformRepository) GetByCustomDeviceID(platformID int, customID string) (*domainplatform.Channel, error) {
	var row model.PlatformChannel
	err := r.db.Where("platform_id = ? AND custom_gb_code = ?", platformID, customID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return toDomainPlatformChannel(&row), nil
}

func (r *PlatformRepository) UpdatePlatformChannelCustom(platformID, channelID int, customID, customName string) error {
	overrides := map[string]any{}
	var row model.PlatformChannel
	if err := r.db.Where("platform_id = ? AND channel_id = ?", platformID, channelID).First(&row).Error; err == nil {
		if strings.TrimSpace(row.OverridesJSON) != "" {
			_ = json.Unmarshal([]byte(row.OverridesJSON), &overrides)
		}
	}
	if strings.TrimSpace(customName) != "" {
		overrides["customName"] = customName
	} else {
		delete(overrides, "customName")
	}
	raw, _ := json.Marshal(overrides)
	return r.db.Model(&model.PlatformChannel{}).
		Where("platform_id = ? AND channel_id = ?", platformID, channelID).
		Updates(map[string]any{
			"custom_gb_code": customID, "overrides_json": string(raw), "update_time": nowTimeStr(),
		}).Error
}

func toDomainPlatformChannel(m *model.PlatformChannel) *domainplatform.Channel {
	ch := &domainplatform.Channel{
		ID: int(m.ID), PlatformID: int(m.PlatformID), DeviceChannelID: int(m.ChannelID),
		CustomDeviceID: m.CustomGBCode,
	}
	if strings.TrimSpace(m.OverridesJSON) != "" {
		var ov map[string]any
		if json.Unmarshal([]byte(m.OverridesJSON), &ov) == nil {
			if v, ok := ov["customName"].(string); ok {
				ch.CustomName = v
			}
		}
	}
	return ch
}

func toDomainAlarm(m *model.Alarm) *domainalarm.Alarm {
	return &domainalarm.Alarm{
		ID: int(m.ID), ChannelID: int(m.ChannelID), Description: m.Description, SnapPath: m.SnapPath,
		RecordPath: m.RecordPath, Longitude: m.Longitude, Latitude: m.Latitude,
		AlarmType: m.AlarmType, AlarmCode: m.AlarmCode, AlarmTime: m.AlarmTime,
		EventID: m.EventID, DeviceInternalCode: m.DeviceInternalCode,
		ChannelInternalCode: m.ChannelInternalCode, BBoxJSON: m.BBoxJSON,
		Severity: m.Severity, Status: m.Status, HandledBy: m.HandedBy,
		HandledAt: m.HandledAt, Remark: m.Remark,
	}
}

func toModelPosition(p *domainposition.MobilePosition) *model.MobilePosition {
	return &model.MobilePosition{
		ID: int64(p.ID), ChannelID: int64(p.ChannelID), Timestamp: p.Timestamp,
		Longitude: p.Longitude, Latitude: p.Latitude, Altitude: p.Altitude,
		Speed: p.Speed, Direction: p.Direction, CreateTime: p.CreateTime,
	}
}

func toDomainPosition(m *model.MobilePosition) *domainposition.MobilePosition {
	return &domainposition.MobilePosition{
		ID: int(m.ID), ChannelID: int(m.ChannelID), Timestamp: m.Timestamp,
		Longitude: m.Longitude, Latitude: m.Latitude, Altitude: m.Altitude,
		Speed: m.Speed, Direction: m.Direction, CreateTime: m.CreateTime,
	}
}

func toModelPlatform(p *domainplatform.Platform) *model.UpperPlatform {
	return &model.UpperPlatform{
		ID: int64(p.ID), Enable: p.Enable, Name: p.Name, ServerGBID: p.ServerGBID,
		ServerGBDomain: p.ServerGBDomain, ServerIP: p.ServerIP, ServerPort: p.ServerPort,
		DeviceGBID: p.DeviceGBID, DeviceIP: p.DeviceIP, DevicePort: p.DevicePort,
		Username: p.Username, Password: p.Password, Expires: p.Expires,
		KeepTimeout: p.KeepTimeout, Transport: p.Transport, Status: p.Status,
		AutoPushChannel: p.AutoPushChannel, ServerID: p.ServerID,
		CreateTime: p.CreateTime, UpdateTime: p.UpdateTime,
	}
}

func toDomainPlatform(m *model.UpperPlatform) *domainplatform.Platform {
	return &domainplatform.Platform{
		ID: int(m.ID), Enable: m.Enable, Name: m.Name, ServerGBID: m.ServerGBID,
		ServerGBDomain: m.ServerGBDomain, ServerIP: m.ServerIP, ServerPort: m.ServerPort,
		DeviceGBID: m.DeviceGBID, DeviceIP: m.DeviceIP, DevicePort: m.DevicePort,
		Username: m.Username, Password: m.Password, Expires: m.Expires,
		KeepTimeout: m.KeepTimeout, Transport: m.Transport, Status: m.Status,
		AutoPushChannel: m.AutoPushChannel, ServerID: m.ServerID,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func timeNow() string { return time.Now().Format("2006-01-02 15:04:05") }

func (r *PlatformRepository) ListCatalogSnapshot(platformID int) ([]*domainplatform.CatalogSnapshot, error) {
	var rows []model.PlatformCatalogSnapshot
	if err := r.db.Where("platform_id = ?", platformID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainplatform.CatalogSnapshot, 0, len(rows))
	for i := range rows {
		out = append(out, &domainplatform.CatalogSnapshot{
			PlatformID: int(rows[i].PlatformID), DeviceID: rows[i].DeviceID, Name: rows[i].Name,
			ParentID: rows[i].ParentID, ItemType: rows[i].ItemType, Status: rows[i].Status,
			Hash: rows[i].Hash,
		})
	}
	return out, nil
}

func (r *PlatformRepository) ReplaceCatalogSnapshot(platformID int, list []*domainplatform.CatalogSnapshot) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("platform_id = ?", platformID).Delete(&model.PlatformCatalogSnapshot{}).Error; err != nil {
			return err
		}
		if len(list) == 0 {
			return nil
		}
		rows := make([]model.PlatformCatalogSnapshot, 0, len(list))
		for _, e := range list {
			rows = append(rows, model.PlatformCatalogSnapshot{
				PlatformID: int64(platformID), DeviceID: e.DeviceID, Name: e.Name,
				ParentID: e.ParentID, ItemType: e.ItemType, Status: e.Status, Hash: e.Hash,
			})
		}
		return tx.Create(&rows).Error
	})
}

func (r *PlatformRepository) ListGroupShares(platformID int) ([]string, error) {
	var rows []model.PlatformGroupShare
	if err := r.db.Where("platform_id = ?", platformID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		var code string
		_ = r.db.Table("zws_group").Select("code").Where("id = ?", row.GroupID).Scan(&code).Error
		if strings.TrimSpace(code) != "" {
			out = append(out, code)
		}
	}
	return out, nil
}

// AddGroupShares 入参为业务分组 deviceID（=zws_group.code），解析为 group.id 后落表。
func (r *PlatformRepository) AddGroupShares(platformID int, groupDeviceIDs []string) error {
	if len(groupDeviceIDs) == 0 {
		return nil
	}
	rows := make([]model.PlatformGroupShare, 0, len(groupDeviceIDs))
	for _, id := range groupDeviceIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		var gid int64
		_ = r.db.Table("zws_group").Select("id").Where("code = ?", id).Scan(&gid).Error
		if gid == 0 {
			continue
		}
		rows = append(rows, model.PlatformGroupShare{PlatformID: int64(platformID), GroupID: gid})
	}
	if len(rows) == 0 {
		return nil
	}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func (r *PlatformRepository) RemoveGroupShares(platformID int, groupDeviceIDs []string) error {
	if len(groupDeviceIDs) == 0 {
		return nil
	}
	gids := make([]int64, 0, len(groupDeviceIDs))
	for _, id := range groupDeviceIDs {
		var gid int64
		_ = r.db.Table("zws_group").Select("id").Where("code = ?", strings.TrimSpace(id)).Scan(&gid).Error
		if gid > 0 {
			gids = append(gids, gid)
		}
	}
	if len(gids) == 0 {
		return nil
	}
	return r.db.Where("platform_id = ? AND group_id IN ?", platformID, gids).
		Delete(&model.PlatformGroupShare{}).Error
}
