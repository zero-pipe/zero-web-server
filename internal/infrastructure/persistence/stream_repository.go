package persistence

import (
	"fmt"
	"time"

	"zero-web-kit/internal/domain/shared"
	"zero-web-kit/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

// --- Stream Push ---

type StreamPushView struct {
	ID               int    `json:"id"`
	App              string `json:"app"`
	Stream           string `json:"stream"`
	MediaServerID    string `json:"mediaServerId"`
	ServerID         string `json:"serverId"`
	PushTime         string `json:"pushTime"`
	CreateTime       string `json:"createTime"`
	UpdateTime       string `json:"updateTime"`
	Pushing          bool   `json:"pushing"`
	StartOfflinePush bool   `json:"startOfflinePush"`
	GbID             int    `json:"gbId"`
	GbDeviceID       string `json:"gbDeviceId"`
	GbName           string `json:"gbName"`
	Name             string `json:"name"`
	DataType         int    `json:"dataType"`
}

type StreamPushRepository struct{ db *gorm.DB }

func NewStreamPushRepository(db *gorm.DB) *StreamPushRepository { return &StreamPushRepository{db: db} }

func (r *StreamPushRepository) Create(m *model.StreamPush) error {
	return r.db.Create(m).Error
}

func (r *StreamPushRepository) Update(m *model.StreamPush) error {
	return r.db.Save(m).Error
}

func (r *StreamPushRepository) Delete(id int) error {
	return r.db.Delete(&model.StreamPush{}, id).Error
}

func (r *StreamPushRepository) GetByID(id int) (*model.StreamPush, error) {
	var m model.StreamPush
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *StreamPushRepository) GetByAppStream(app, stream string) (*model.StreamPush, error) {
	var m model.StreamPush
	if err := r.db.Where("app = ? AND stream = ?", app, stream).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *StreamPushRepository) List(page, count int, query string, pushing *bool, mediaServerID string) ([]StreamPushView, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	base := r.db.Table("wvp_stream_push st").
		Select(`st.id, st.app, st.stream, st.media_server_id, st.server_id, st.push_time, st.create_time, st.update_time, st.pushing, st.start_offline_push,
			wdc.id as gb_id, wdc.gb_device_id, wdc.name as gb_name, wdc.name, wdc.data_type`).
		Joins("LEFT JOIN wvp_device_channel wdc ON wdc.data_type = 2 AND st.id = wdc.data_device_id")
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		base = base.Where("st.app LIKE ? OR st.stream LIKE ? OR wdc.gb_device_id LIKE ? OR wdc.name LIKE ?", like, like, like, like)
	}
	if pushing != nil {
		base = base.Where("st.pushing = ?", *pushing)
	}
	if mediaServerID != "" {
		base = base.Where("st.media_server_id = ?", mediaServerID)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []StreamPushView
	err := base.Order("st.pushing DESC, st.create_time DESC").Offset((page - 1) * count).Limit(count).Scan(&rows).Error
	for i := range rows {
		if rows[i].DataType == 0 {
			rows[i].DataType = shared.ChannelDataTypeStreamPush
		}
	}
	return rows, total, err
}

func (r *StreamPushRepository) UpdatePushing(id int, pushing bool, mediaServerID string) error {
	return r.db.Model(&model.StreamPush{}).Where("id = ?", id).Updates(map[string]any{
		"pushing": pushing, "media_server_id": mediaServerID, "update_time": nowTimeStr(),
	}).Error
}

func (r *StreamPushRepository) UpsertGBChannel(pushID int, gbDeviceID, gbName string) error {
	var ch model.GBDeviceChannel
	err := r.db.Where("data_type = ? AND data_device_id = ?", shared.ChannelDataTypeStreamPush, pushID).First(&ch).Error
	now := nowTimeStr()
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(&model.GBDeviceChannel{
			DataType: shared.ChannelDataTypeStreamPush, DataDeviceID: pushID,
			GBDeviceID: gbDeviceID, Name: gbName, Status: "ON", CreateTime: now, UpdateTime: now,
		}).Error
	}
	if err != nil {
		return err
	}
	ch.GBDeviceID = gbDeviceID
	ch.Name = gbName
	ch.UpdateTime = now
	return r.db.Save(&ch).Error
}

func (r *StreamPushRepository) RemoveGBChannel(pushID int) error {
	return r.db.Where("data_type = ? AND data_device_id = ?", shared.ChannelDataTypeStreamPush, pushID).
		Delete(&model.GBDeviceChannel{}).Error
}

// --- Stream Proxy ---

type StreamProxyRepository struct{ db *gorm.DB }

func NewStreamProxyRepository(db *gorm.DB) *StreamProxyRepository { return &StreamProxyRepository{db: db} }

func (r *StreamProxyRepository) Create(m *model.StreamProxy) error {
	return r.db.Create(m).Error
}

func (r *StreamProxyRepository) Update(m *model.StreamProxy) error {
	return r.db.Save(m).Error
}

func (r *StreamProxyRepository) Delete(id int) error {
	return r.db.Delete(&model.StreamProxy{}, id).Error
}

func (r *StreamProxyRepository) GetByID(id int) (*model.StreamProxy, error) {
	var m model.StreamProxy
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *StreamProxyRepository) List(page, count int, query string, pulling *bool, mediaServerID string) ([]model.StreamProxy, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.StreamProxy{})
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("name LIKE ? OR app LIKE ? OR stream LIKE ? OR src_url LIKE ?", like, like, like, like)
	}
	if pulling != nil {
		q = q.Where("pulling = ?", *pulling)
	}
	if mediaServerID != "" {
		q = q.Where("media_server_id = ? OR relates_media_server_id = ?", mediaServerID, mediaServerID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.StreamProxy
	if err := q.Order("id DESC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *StreamProxyRepository) GetByAppStream(app, stream string) (*model.StreamProxy, error) {
	var m model.StreamProxy
	if err := r.db.Where("app = ? AND stream = ?", app, stream).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *StreamProxyRepository) UpsertGBChannel(proxyID int, gbDeviceID, gbName, app, stream string) error {
	var ch model.GBDeviceChannel
	err := r.db.Where("data_type = ? AND data_device_id = ?", shared.ChannelDataTypeStreamProxy, proxyID).First(&ch).Error
	now := nowTimeStr()
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(&model.GBDeviceChannel{
			DeviceID: app, DataType: shared.ChannelDataTypeStreamProxy, DataDeviceID: proxyID,
			GBDeviceID: gbDeviceID, Name: gbName, Status: "ON", CreateTime: now, UpdateTime: now,
		}).Error
	}
	if err != nil {
		return err
	}
	ch.GBDeviceID = gbDeviceID
	ch.Name = gbName
	ch.UpdateTime = now
	return r.db.Save(&ch).Error
}

// --- Cloud Record ---

type CloudRecordRepository struct{ db *gorm.DB }

func NewCloudRecordRepository(db *gorm.DB) *CloudRecordRepository { return &CloudRecordRepository{db: db} }

func (r *CloudRecordRepository) Create(rec *model.CloudRecord) error {
	return r.db.Create(rec).Error
}

func (r *CloudRecordRepository) Delete(ids []int) error {
	return r.db.Where("id IN ?", ids).Delete(&model.CloudRecord{}).Error
}

func (r *CloudRecordRepository) GetByID(id int) (*model.CloudRecord, error) {
	var m model.CloudRecord
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *CloudRecordRepository) List(page, count int, app, stream, query, callID, mediaServerID string, startTime, endTime int64, asc bool) ([]model.CloudRecord, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.CloudRecord{})
	if app != "" {
		q = q.Where("app = ?", app)
	}
	if stream != "" {
		q = q.Where("stream = ?", stream)
	}
	if callID != "" {
		q = q.Where("call_id = ?", callID)
	}
	if mediaServerID != "" {
		q = q.Where("media_server_id = ?", mediaServerID)
	}
	if startTime > 0 {
		q = q.Where("start_time >= ?", startTime/1000)
	}
	if endTime > 0 {
		q = q.Where("start_time <= ?", endTime/1000)
	}
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("file_name LIKE ? OR file_path LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "start_time DESC"
	if asc {
		order = "start_time ASC"
	}
	var rows []model.CloudRecord
	err := q.Order(order).Offset((page - 1) * count).Limit(count).Find(&rows).Error
	return rows, total, err
}

func (r *CloudRecordRepository) DateList(app, stream, mediaServerID string, year, month int) ([]string, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)
	q := r.db.Model(&model.CloudRecord{}).Where("app = ? AND stream = ?", app, stream).
		Where("start_time >= ? AND start_time < ?", start.Unix(), end.Unix())
	if mediaServerID != "" {
		q = q.Where("media_server_id = ?", mediaServerID)
	}
	var rows []model.CloudRecord
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	set := make(map[string]struct{})
	for _, row := range rows {
		d := time.Unix(row.StartTime, 0).Format("2006-01-02")
		set[d] = struct{}{}
	}
	dates := make([]string, 0, len(set))
	for d := range set {
		dates = append(dates, d)
	}
	return dates, nil
}

// --- Record Plan ---

type RecordPlanRepository struct{ db *gorm.DB }

func NewRecordPlanRepository(db *gorm.DB) *RecordPlanRepository { return &RecordPlanRepository{db: db} }

func (r *RecordPlanRepository) Create(plan *model.RecordPlan, items []model.RecordPlanItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(plan).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].PlanID = plan.ID
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *RecordPlanRepository) Update(plan *model.RecordPlan, items []model.RecordPlanItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(plan).Error; err != nil {
			return err
		}
		if err := tx.Where("plan_id = ?", plan.ID).Delete(&model.RecordPlanItem{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].PlanID = plan.ID
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *RecordPlanRepository) Delete(id int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		_ = tx.Model(&model.GBDeviceChannel{}).Where("record_plan_id = ?", id).Update("record_plan_id", 0)
		_ = tx.Where("plan_id = ?", id).Delete(&model.RecordPlanItem{})
		return tx.Delete(&model.RecordPlan{}, id).Error
	})
}

func (r *RecordPlanRepository) GetByID(id int) (*model.RecordPlan, []model.RecordPlanItem, error) {
	var plan model.RecordPlan
	if err := r.db.First(&plan, id).Error; err != nil {
		return nil, nil, err
	}
	var items []model.RecordPlanItem
	_ = r.db.Where("plan_id = ?", id).Find(&items)
	return &plan, items, nil
}

func (r *RecordPlanRepository) List(page, count int, query string) ([]model.RecordPlan, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.RecordPlan{})
	if query != "" {
		q = q.Where("name LIKE ?", fmt.Sprintf("%%%s%%", query))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.RecordPlan
	err := q.Order("id DESC").Offset((page - 1) * count).Limit(count).Find(&rows).Error
	return rows, total, err
}

func (r *RecordPlanRepository) LinkChannels(planID int, channelIDs []int) error {
	return r.db.Model(&model.GBDeviceChannel{}).Where("id IN ?", channelIDs).Update("record_plan_id", planID).Error
}

func (r *RecordPlanRepository) UnlinkChannels(channelIDs []int) error {
	return r.db.Model(&model.GBDeviceChannel{}).Where("id IN ?", channelIDs).Update("record_plan_id", 0).Error
}

func (r *RecordPlanRepository) LinkAll(planID int) error {
	return r.db.Model(&model.GBDeviceChannel{}).Where("data_type = ?", shared.ChannelDataTypeGB28181).
		Update("record_plan_id", planID).Error
}

func (r *RecordPlanRepository) UnlinkAll(planID int) error {
	return r.db.Model(&model.GBDeviceChannel{}).Where("record_plan_id = ?", planID).Update("record_plan_id", 0).Error
}

func (r *RecordPlanRepository) ChannelList(page, count int, planID int, query string, hasLink *bool) ([]model.GBDeviceChannel, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.GBDeviceChannel{})
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("name LIKE ? OR gb_device_id LIKE ?", like, like)
	}
	if hasLink != nil {
		if *hasLink {
			q = q.Where("record_plan_id = ?", planID)
		} else {
			q = q.Where("record_plan_id = 0 OR record_plan_id IS NULL OR record_plan_id <> ?", planID)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.GBDeviceChannel
	err := q.Order("id ASC").Offset((page - 1) * count).Limit(count).Find(&rows).Error
	return rows, total, err
}

func (r *RecordPlanRepository) CountChannels(planID int) int64 {
	var n int64
	_ = r.db.Model(&model.GBDeviceChannel{}).Where("record_plan_id = ?", planID).Count(&n)
	return n
}

func (r *RecordPlanRepository) QueryRecordingChannels(weekDay, minuteIndex int) ([]model.GBDeviceChannel, error) {
	var rows []model.GBDeviceChannel
	err := r.db.Table("wvp_device_channel wdc").
		Select("wdc.*").
		Joins("JOIN wvp_record_plan_item wrpi ON wrpi.plan_id = wdc.record_plan_id").
		Where("wrpi.week_day = ? AND wrpi.start <= ? AND wrpi.stop >= ?", weekDay, minuteIndex, minuteIndex).
		Group("wdc.id").
		Find(&rows).Error
	return rows, err
}
