package persistence

import (
	"fmt"
	"time"

	domainevent "zero-web-server/internal/domain/event"
	"zero-web-server/internal/infrastructure/persistence/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventRepository struct{ db *gorm.DB }

func NewEventRepository(db *gorm.DB) *EventRepository { return &EventRepository{db: db} }

var _ domainevent.Repository = (*EventRepository)(nil)

func (r *EventRepository) Create(e *domainevent.Event) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	return r.db.Create(&model.Event{
		ID: e.ID, EventID: e.EventID, SchemaVersion: e.SchemaVersion, SourceID: e.SourceID,
		DeviceID: e.DeviceID, BindingID: e.BindingID, ResourceID: e.ResourceID,
		DeviceInternalCode: e.DeviceInternalCode, DeviceUID: e.DeviceUID,
		ChannelExternalKey: e.ChannelExternalKey, ChannelInternalCode: e.ChannelInternalCode,
		EventType: e.EventType, Severity: e.Severity,
		OccurredAt: e.OccurredAt, ReceivedAt: e.ReceivedAt,
		ImageKey: e.ImageKey, ImageURL: e.ImageURL, ImageExpiresAt: e.ImageExpiresAt,
		BBoxJSON: e.BBoxJSON, RawJSON: e.RawJSON, Status: e.Status, AlarmID: e.AlarmID,
		CreateTime: now,
	}).Error
}

func (r *EventRepository) Exists(eventID string) (bool, error) {
	var n int64
	if err := r.db.Model(&model.Event{}).Where("event_id = ?", eventID).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *EventRepository) Get(id string) (*domainevent.Event, error) {
	var row struct {
		model.Event
		DeviceName  string `gorm:"column:device_name"`
		ChannelName string `gorm:"column:channel_name"`
	}
	err := r.db.Table("zws_event e").
		Select("e.*, d.name AS device_name, r.name AS channel_name").
		Joins("LEFT JOIN zws_physical_device d ON d.id = e.device_id").
		Joins("LEFT JOIN zws_resource r ON r.id = e.resource_id").
		Where("e.id = ?", id).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return toDomainEvent(&row.Event, row.DeviceName, row.ChannelName), nil
}

func (r *EventRepository) List(page, count int, f domainevent.ListFilter) ([]*domainevent.Event, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 20
	}
	q := r.db.Table("zws_event e").
		Joins("LEFT JOIN zws_physical_device d ON d.id = e.device_id").
		Joins("LEFT JOIN zws_resource r ON r.id = e.resource_id")
	if f.DeviceID != "" {
		q = q.Where("e.device_id = ?", f.DeviceID)
	}
	if f.DeviceInternalCode != "" {
		q = q.Where("e.device_internal_code = ?", f.DeviceInternalCode)
	}
	if f.DeviceName != "" {
		q = q.Where("d.name LIKE ?", "%"+f.DeviceName+"%")
	}
	if f.EventType != "" {
		q = q.Where("e.event_type = ?", f.EventType)
	}
	if f.Status != "" {
		q = q.Where("e.status = ?", f.Status)
	}
	if f.Begin > 0 {
		q = q.Where("e.occurred_at >= ?", f.Begin)
	}
	if f.End > 0 {
		q = q.Where("e.occurred_at <= ?", f.End)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		model.Event
		DeviceName  string `gorm:"column:device_name"`
		ChannelName string `gorm:"column:channel_name"`
	}
	err := q.Select("e.*, d.name AS device_name, r.name AS channel_name").
		Order("e.occurred_at DESC, e.id DESC").
		Offset((page - 1) * count).Limit(count).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]*domainevent.Event, 0, len(rows))
	for i := range rows {
		out = append(out, toDomainEvent(&rows[i].Event, rows[i].DeviceName, rows[i].ChannelName))
	}
	return out, total, nil
}

func (r *EventRepository) UpdateAlarmID(id string, alarmID int) error {
	return r.db.Model(&model.Event{}).Where("id = ?", id).Update("alarm_id", alarmID).Error
}

func (r *EventRepository) DeleteBefore(before int64) (int64, error) {
	res := r.db.Where("occurred_at < ?", before).Delete(&model.Event{})
	return res.RowsAffected, res.Error
}

func (r *EventRepository) DeleteRange(begin, end int64, all bool) (int64, error) {
	// Where("1=1") 既避免 GORM 全表更新保护，也保证 all=true 时能删除全部。
	q := r.db.Where("1=1")
	if !all {
		if begin <= 0 && end <= 0 {
			return 0, fmt.Errorf("请指定时间范围或选择全部删除")
		}
		if begin > 0 {
			q = q.Where("occurred_at >= ?", begin)
		}
		if end > 0 {
			q = q.Where("occurred_at <= ?", end)
		}
	}
	res := q.Delete(&model.Event{})
	return res.RowsAffected, res.Error
}

func toDomainEvent(m *model.Event, deviceName, channelName string) *domainevent.Event {
	return &domainevent.Event{
		ID: m.ID, EventID: m.EventID, SchemaVersion: m.SchemaVersion, SourceID: m.SourceID,
		DeviceID: m.DeviceID, DeviceName: deviceName, ChannelName: channelName,
		BindingID: m.BindingID, ResourceID: m.ResourceID,
		DeviceInternalCode: m.DeviceInternalCode, DeviceUID: m.DeviceUID,
		ChannelExternalKey: m.ChannelExternalKey, ChannelInternalCode: m.ChannelInternalCode,
		EventType: m.EventType, Severity: m.Severity,
		OccurredAt: m.OccurredAt, ReceivedAt: m.ReceivedAt,
		ImageKey: m.ImageKey, ImageURL: m.ImageURL, ImageExpiresAt: m.ImageExpiresAt,
		BBoxJSON: m.BBoxJSON, RawJSON: m.RawJSON, Status: m.Status, AlarmID: m.AlarmID,
		CreateTime: m.CreateTime,
	}
}
