package persistence

import (
	"fmt"
	"time"

	domainevent "zero-web-server/internal/domain/event"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

// EventRepository 事件持久化。
// 新 schema：zws_event 用 BIGINT 自增主键，业务唯一标识为 event_id；
// 设备/通道外键为 BIGINT（由内码解析）。对外（domain）仍用字符串 ID，边界处互转。
type EventRepository struct{ db *gorm.DB }

func NewEventRepository(db *gorm.DB) *EventRepository { return &EventRepository{db: db} }

var _ domainevent.Repository = (*EventRepository)(nil)

func (r *EventRepository) Create(e *domainevent.Event) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	if e.EventID == "" {
		e.EventID = newBusinessID("E")
	}
	m := &model.Event{
		EventID: e.EventID, EventType: e.EventType, Severity: e.Severity, Status: e.Status,
		OccurredAt: e.OccurredAt, ReceivedAt: e.ReceivedAt,
		DeviceInternalCode: e.DeviceInternalCode, ChannelInternalCode: e.ChannelInternalCode,
		BBoxJSON: e.BBoxJSON, ImageURL: firstNonEmptyStr(e.ImageURL, e.ImageKey),
		RawJSON: e.RawJSON, CreateTime: now,
	}
	if devID, err := deviceIDByInternalCode(r.db, e.DeviceInternalCode); err != nil {
		return fmt.Errorf("resolve event device: %w", err)
	} else {
		m.DeviceID = devID
	}
	if chID, err := channelIDByInternalCode(r.db, e.ChannelInternalCode); err == nil && chID > 0 {
		m.ChannelID = &chID
	}
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	e.ID = m.EventID
	e.CreateTime = now
	return nil
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
		AlarmID     int    `gorm:"column:alarm_id"`
	}
	err := r.db.Table("zws_event e").
		Select("e.*, d.name AS device_name, c.name AS channel_name, COALESCE(a.id, 0) AS alarm_id").
		Joins("LEFT JOIN zws_device d ON d.id = e.device_id").
		Joins("LEFT JOIN zws_channel c ON c.id = e.channel_id").
		Joins("LEFT JOIN zws_alarm a ON a.event_id = e.event_id").
		Where("e.event_id = ?", id).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.EventID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	ev := toDomainEvent(&row.Event, row.DeviceName, row.ChannelName)
	ev.AlarmID = row.AlarmID
	return ev, nil
}

func (r *EventRepository) List(page, count int, f domainevent.ListFilter) ([]*domainevent.Event, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 20
	}
	q := r.db.Table("zws_event e").
		Joins("LEFT JOIN zws_device d ON d.id = e.device_id").
		Joins("LEFT JOIN zws_channel c ON c.id = e.channel_id").
		Joins("LEFT JOIN zws_alarm a ON a.event_id = e.event_id")
	if f.DeviceInternalCode != "" {
		q = q.Where("e.device_internal_code = ?", f.DeviceInternalCode)
	}
	if f.DeviceID != "" {
		q = q.Where("e.device_internal_code = ?", f.DeviceID)
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
		AlarmID     int    `gorm:"column:alarm_id"`
	}
	err := q.Select("e.*, d.name AS device_name, c.name AS channel_name, COALESCE(a.id, 0) AS alarm_id").
		Order("e.occurred_at DESC, e.id DESC").
		Offset((page - 1) * count).Limit(count).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]*domainevent.Event, 0, len(rows))
	for i := range rows {
		ev := toDomainEvent(&rows[i].Event, rows[i].DeviceName, rows[i].ChannelName)
		ev.AlarmID = rows[i].AlarmID
		out = append(out, ev)
	}
	return out, total, nil
}

// UpdateAlarmID 新 schema 报警关联落在 zws_alarm.event_id，事件侧无需写回，保持接口兼容为空实现。
func (r *EventRepository) UpdateAlarmID(id string, alarmID int) error { return nil }

func (r *EventRepository) DeleteBefore(before int64) (int64, error) {
	res := r.db.Where("occurred_at < ?", before).Delete(&model.Event{})
	return res.RowsAffected, res.Error
}

func (r *EventRepository) DeleteRange(begin, end int64, all bool) (int64, error) {
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
	devIDStr := ""
	if m.DeviceID > 0 {
		devIDStr = formatID(m.DeviceID)
	}
	return &domainevent.Event{
		ID: m.EventID, EventID: m.EventID, DeviceName: deviceName, ChannelName: channelName,
		DeviceID: devIDStr, DeviceInternalCode: m.DeviceInternalCode,
		ChannelInternalCode: m.ChannelInternalCode,
		EventType:           m.EventType, Severity: m.Severity,
		OccurredAt: m.OccurredAt, ReceivedAt: m.ReceivedAt,
		ImageURL: m.ImageURL, BBoxJSON: m.BBoxJSON, RawJSON: m.RawJSON, Status: m.Status,
		CreateTime: m.CreateTime,
	}
}
