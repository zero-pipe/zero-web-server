package persistence

import (
	"fmt"
	"time"

	domaince "zero-web-server/internal/domain/customevent"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

// CustomEventRepository 自定义事件持久化。
// 新 schema：BIGINT 自增主键；对外字符串 ID 优先用业务 event_id，回退为 BIGINT 字符串。
type CustomEventRepository struct{ db *gorm.DB }

func NewCustomEventRepository(db *gorm.DB) *CustomEventRepository {
	return &CustomEventRepository{db: db}
}

var _ domaince.Repository = (*CustomEventRepository)(nil)

func customEventCode(m *model.CustomEvent) string {
	if m.EventID != "" {
		return m.EventID
	}
	return formatID(m.ID)
}

func (r *CustomEventRepository) Create(e *domaince.CustomEvent) error {
	if e.EventID == "" {
		e.EventID = newBusinessID("CE")
	}
	if e.CreateTime == "" {
		e.CreateTime = time.Now().Format("2006-01-02 15:04:05")
	}
	m := &model.CustomEvent{
		EventID: e.EventID,
		DeviceInternalCode: e.DeviceInternalCode, ChannelInternalCode: e.ChannelInternalCode,
		DeviceID: e.DeviceID, ResourceID: e.ResourceID,
		MajorType: e.MajorType, MinorType: e.MinorType,
		OccurAt: e.OccurAt, Level: e.Level,
		ImageKey: e.ImageKey, ImageURL: e.ImageURL, RawJSON: e.RawJSON,
		CreateTime: e.CreateTime,
	}
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	e.ID = customEventCode(m)
	return nil
}

func (r *CustomEventRepository) Exists(eventID string) (bool, error) {
	var n int64
	if err := r.db.Model(&model.CustomEvent{}).Where("event_id = ?", eventID).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *CustomEventRepository) Get(id string) (*domaince.CustomEvent, error) {
	var m model.CustomEvent
	q := r.db.Where("event_id = ?", id)
	if pk, ok := parseID(id); ok {
		q = r.db.Where("event_id = ? OR id = ?", id, pk)
	}
	if err := q.First(&m).Error; err != nil {
		return nil, err
	}
	return customEventFromModel(&m), nil
}

func (r *CustomEventRepository) List(page, count int, f domaince.ListFilter) ([]*domaince.CustomEvent, int64, error) {
	if page < 1 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.CustomEvent{})
	if f.DeviceInternalCode != "" {
		q = q.Where("device_internal_code = ?", f.DeviceInternalCode)
	}
	if f.ChannelInternalCode != "" {
		q = q.Where("channel_internal_code = ?", f.ChannelInternalCode)
	}
	if f.MajorType != "" {
		q = q.Where("major_type = ?", f.MajorType)
	}
	if f.MinorType != "" {
		q = q.Where("minor_type = ?", f.MinorType)
	}
	if f.Begin > 0 {
		q = q.Where("occur_at >= ?", f.Begin)
	}
	if f.End > 0 {
		q = q.Where("occur_at <= ?", f.End)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.CustomEvent
	if err := q.Order("occur_at DESC, id DESC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*domaince.CustomEvent, 0, len(rows))
	for i := range rows {
		out = append(out, customEventFromModel(&rows[i]))
	}
	return out, total, nil
}

func (r *CustomEventRepository) DistinctTypes(deviceInternalCode string) ([]domaince.TypeCount, error) {
	q := r.db.Model(&model.CustomEvent{}).
		Select("major_type, minor_type, count(*) as count")
	if deviceInternalCode != "" {
		q = q.Where("device_internal_code = ?", deviceInternalCode)
	}
	var rows []domaince.TypeCount
	if err := q.Group("major_type, minor_type").Order("major_type, minor_type").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *CustomEventRepository) DeleteRange(begin, end int64, all bool) (int64, error) {
	q := r.db.Where("1=1")
	if !all {
		if begin <= 0 && end <= 0 {
			return 0, fmt.Errorf("请指定时间范围或选择全部删除")
		}
		if begin > 0 {
			q = q.Where("occur_at >= ?", begin)
		}
		if end > 0 {
			q = q.Where("occur_at <= ?", end)
		}
	}
	res := q.Delete(&model.CustomEvent{})
	return res.RowsAffected, res.Error
}

func customEventFromModel(m *model.CustomEvent) *domaince.CustomEvent {
	return &domaince.CustomEvent{
		ID: customEventCode(m), EventID: m.EventID,
		DeviceInternalCode: m.DeviceInternalCode, ChannelInternalCode: m.ChannelInternalCode,
		DeviceID: m.DeviceID, ResourceID: m.ResourceID,
		MajorType: m.MajorType, MinorType: m.MinorType,
		OccurAt: m.OccurAt, Level: m.Level,
		ImageKey: m.ImageKey, ImageURL: m.ImageURL, RawJSON: m.RawJSON,
		CreateTime: m.CreateTime,
	}
}
