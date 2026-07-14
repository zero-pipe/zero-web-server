package persistence

import (
	"fmt"
	"time"

	domainalarm "zero-web-server/internal/domain/alarm"
	domainplatform "zero-web-server/internal/domain/platform"
	domainposition "zero-web-server/internal/domain/position"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type AlarmRepository struct{ db *gorm.DB }

func NewAlarmRepository(db *gorm.DB) *AlarmRepository { return &AlarmRepository{db: db} }

func (r *AlarmRepository) Create(alarm *domainalarm.Alarm) error {
	m := &model.Alarm{
		ChannelID: alarm.ChannelID, Description: alarm.Description, SnapPath: alarm.SnapPath,
		RecordPath: alarm.RecordPath, Longitude: alarm.Longitude, Latitude: alarm.Latitude,
		AlarmType: alarm.AlarmType, AlarmTime: alarm.AlarmTime,
	}
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	alarm.ID = m.ID
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

type PositionRepository struct{ db *gorm.DB }

func NewPositionRepository(db *gorm.DB) *PositionRepository { return &PositionRepository{db: db} }

func (r *PositionRepository) Create(pos *domainposition.MobilePosition) error {
	m := toModelPosition(pos)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	pos.ID = m.ID
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
	var m model.Platform
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return toDomainPlatform(&m), nil
}

func (r *PlatformRepository) GetByServerGBID(serverGBID string) (*domainplatform.Platform, error) {
	var m model.Platform
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
	q := r.db.Model(&model.Platform{})
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("name LIKE ? OR server_gb_id LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.Platform
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
	p.ID = m.ID
	return nil
}

func (r *PlatformRepository) Update(p *domainplatform.Platform) error {
	return r.db.Save(toModelPlatform(p)).Error
}

func (r *PlatformRepository) Delete(id int) error {
	return r.db.Delete(&model.Platform{}, id).Error
}

func (r *PlatformRepository) UpdateStatus(id int, online bool) error {
	return r.db.Model(&model.Platform{}).Where("id = ?", id).Update("status", online).Error
}

func (r *PlatformRepository) ListEnabled() ([]*domainplatform.Platform, error) {
	var rows []model.Platform
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
	err := r.db.Model(&model.GBDeviceChannel{}).Where("device_id = ?", deviceID).Pluck("id", &ids).Error
	return ids, err
}

func (r *PlatformRepository) ListPlatformChannels(platformID int) ([]int, error) {
	var ids []int
	err := r.db.Model(&model.PlatformChannel{}).Where("platform_id = ?", platformID).
		Pluck("device_channel_id", &ids).Error
	return ids, err
}

func (r *PlatformRepository) AddPlatformChannel(platformID, channelID int) error {
	pc := model.PlatformChannel{PlatformID: platformID, DeviceChannelID: channelID}
	return r.db.Create(&pc).Error
}

func (r *PlatformRepository) RemovePlatformChannel(platformID, channelID int) error {
	return r.db.Where("platform_id = ? AND device_channel_id = ?", platformID, channelID).
		Delete(&model.PlatformChannel{}).Error
}

func (r *PlatformRepository) GetPlatformChannel(platformID, channelID int) (*model.PlatformChannel, error) {
	var row model.PlatformChannel
	err := r.db.Where("platform_id = ? AND device_channel_id = ?", platformID, channelID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *PlatformRepository) GetByCustomDeviceID(platformID int, customID string) (*model.PlatformChannel, error) {
	var row model.PlatformChannel
	err := r.db.Where("platform_id = ? AND custom_device_id = ?", platformID, customID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *PlatformRepository) UpdatePlatformChannelCustom(platformID, channelID int, customID, customName string) error {
	return r.db.Model(&model.PlatformChannel{}).
		Where("platform_id = ? AND device_channel_id = ?", platformID, channelID).
		Updates(map[string]any{
			"custom_device_id": customID,
			"custom_name":      customName,
		}).Error
}

func toDomainAlarm(m *model.Alarm) *domainalarm.Alarm {
	return &domainalarm.Alarm{
		ID: m.ID, ChannelID: m.ChannelID, Description: m.Description, SnapPath: m.SnapPath,
		RecordPath: m.RecordPath, Longitude: m.Longitude, Latitude: m.Latitude,
		AlarmType: m.AlarmType, AlarmTime: m.AlarmTime,
	}
}

func toModelPosition(p *domainposition.MobilePosition) *model.MobilePosition {
	return &model.MobilePosition{
		ID: p.ID, ChannelID: p.ChannelID, Timestamp: p.Timestamp,
		Longitude: p.Longitude, Latitude: p.Latitude, Altitude: p.Altitude,
		Speed: p.Speed, Direction: p.Direction, CreateTime: p.CreateTime,
	}
}

func toDomainPosition(m *model.MobilePosition) *domainposition.MobilePosition {
	return &domainposition.MobilePosition{
		ID: m.ID, ChannelID: m.ChannelID, Timestamp: m.Timestamp,
		Longitude: m.Longitude, Latitude: m.Latitude, Altitude: m.Altitude,
		Speed: m.Speed, Direction: m.Direction, CreateTime: m.CreateTime,
	}
}

func toModelPlatform(p *domainplatform.Platform) *model.Platform {
	return &model.Platform{
		ID: p.ID, Enable: p.Enable, Name: p.Name, ServerGBID: p.ServerGBID,
		ServerGBDomain: p.ServerGBDomain, ServerIP: p.ServerIP, ServerPort: p.ServerPort,
		DeviceGBID: p.DeviceGBID, DeviceIP: p.DeviceIP, DevicePort: p.DevicePort,
		Username: p.Username, Password: p.Password, Expires: p.Expires,
		KeepTimeout: p.KeepTimeout, Transport: p.Transport, Status: p.Status,
		AutoPushChannel: p.AutoPushChannel, ServerID: p.ServerID,
		CreateTime: p.CreateTime, UpdateTime: p.UpdateTime,
	}
}

func toDomainPlatform(m *model.Platform) *domainplatform.Platform {
	return &domainplatform.Platform{
		ID: m.ID, Enable: m.Enable, Name: m.Name, ServerGBID: m.ServerGBID,
		ServerGBDomain: m.ServerGBDomain, ServerIP: m.ServerIP, ServerPort: m.ServerPort,
		DeviceGBID: m.DeviceGBID, DeviceIP: m.DeviceIP, DevicePort: m.DevicePort,
		Username: m.Username, Password: m.Password, Expires: m.Expires,
		KeepTimeout: m.KeepTimeout, Transport: m.Transport, Status: m.Status,
		AutoPushChannel: m.AutoPushChannel, ServerID: m.ServerID,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func timeNow() string { return time.Now().Format("2006-01-02 15:04:05") }
