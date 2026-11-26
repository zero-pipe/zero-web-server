package persistence

import (
	"fmt"
	"strings"
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
	e := customEventFromModel(&m)
	r.fillNames([]*domaince.CustomEvent{e})
	return e, nil
}

func (r *CustomEventRepository) List(page, count int, f domaince.ListFilter) ([]*domaince.CustomEvent, int64, error) {
	if page < 1 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.CustomEvent{})
	if name := strings.TrimSpace(f.DeviceName); name != "" {
		codes, err := r.deviceCodesByName(name)
		if err != nil {
			return nil, 0, err
		}
		if len(codes) == 0 {
			return []*domaince.CustomEvent{}, 0, nil
		}
		q = q.Where("device_internal_code IN ?", codes)
	}
	if name := strings.TrimSpace(f.ChannelName); name != "" {
		codes, err := r.channelCodesByName(name)
		if err != nil {
			return nil, 0, err
		}
		if len(codes) == 0 {
			return []*domaince.CustomEvent{}, 0, nil
		}
		q = q.Where("channel_internal_code IN ?", codes)
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
	r.fillNames(out)
	return out, total, nil
}

// deviceCodesByName / channelCodesByName 按名称模糊匹配出内码集合，供名称筛选。
func (r *CustomEventRepository) deviceCodesByName(name string) ([]string, error) {
	var codes []string
	err := r.db.Table("zws_device").Where("name LIKE ?", "%"+name+"%").Pluck("internal_code", &codes).Error
	return codes, err
}

func (r *CustomEventRepository) channelCodesByName(name string) ([]string, error) {
	var codes []string
	err := r.db.Table("zws_channel").Where("name LIKE ?", "%"+name+"%").Pluck("internal_code", &codes).Error
	return codes, err
}

// fillNames 批量把内码补成设备/通道名称（列表与详情共用）。
func (r *CustomEventRepository) fillNames(rows []*domaince.CustomEvent) {
	if len(rows) == 0 {
		return
	}
	devCodes := make([]string, 0, len(rows))
	chCodes := make([]string, 0, len(rows))
	for _, e := range rows {
		if e.DeviceInternalCode != "" {
			devCodes = append(devCodes, e.DeviceInternalCode)
		}
		if e.ChannelInternalCode != "" {
			chCodes = append(chCodes, e.ChannelInternalCode)
		}
	}
	devNames := map[string]string{}
	if len(devCodes) > 0 {
		var ds []struct {
			InternalCode string
			Name         string
		}
		if err := r.db.Table("zws_device").Select("internal_code, name").Where("internal_code IN ?", devCodes).Scan(&ds).Error; err == nil {
			for _, d := range ds {
				devNames[d.InternalCode] = d.Name
			}
		}
	}
	chNames := map[string]string{}
	if len(chCodes) > 0 {
		var cs []struct {
			InternalCode string
			Name         string
		}
		if err := r.db.Table("zws_channel").Select("internal_code, name").Where("internal_code IN ?", chCodes).Scan(&cs).Error; err == nil {
			for _, c := range cs {
				chNames[c.InternalCode] = c.Name
			}
		}
	}
	for _, e := range rows {
		e.DeviceName = devNames[e.DeviceInternalCode]
		e.ChannelName = chNames[e.ChannelInternalCode]
	}
}

func (r *CustomEventRepository) DistinctTypes(deviceName string) ([]domaince.TypeCount, error) {
	q := r.db.Model(&model.CustomEvent{}).
		Select("major_type, minor_type, count(*) as count")
	if name := strings.TrimSpace(deviceName); name != "" {
		codes, err := r.deviceCodesByName(name)
		if err != nil {
			return nil, err
		}
		if len(codes) == 0 {
			return []domaince.TypeCount{}, nil
		}
		q = q.Where("device_internal_code IN ?", codes)
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
