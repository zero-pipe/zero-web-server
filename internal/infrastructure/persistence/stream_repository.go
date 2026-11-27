package persistence

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	domaincloudrecord "zero-web-server/internal/domain/cloudrecord"
	domainrecordplan "zero-web-server/internal/domain/recordplan"
	"zero-web-server/internal/domain/shared"
	domainstreamproxy "zero-web-server/internal/domain/streamproxy"
	domainstreampush "zero-web-server/internal/domain/streampush"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

var _ domaincloudrecord.Repository = (*CloudRecordRepository)(nil)

// --- Stream Push ---

type StreamPushRepository struct{ db *gorm.DB }

func NewStreamPushRepository(db *gorm.DB) *StreamPushRepository { return &StreamPushRepository{db: db} }

func (r *StreamPushRepository) Create(d *domainstreampush.StreamPush) error {
	m := toModelStreamPush(d)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	d.ID = int(m.ID)
	return nil
}

func (r *StreamPushRepository) Update(d *domainstreampush.StreamPush) error {
	return r.db.Save(toModelStreamPush(d)).Error
}

func (r *StreamPushRepository) Delete(id int) error {
	return r.db.Delete(&model.StreamPush{}, id).Error
}

func (r *StreamPushRepository) GetByID(id int) (*domainstreampush.StreamPush, error) {
	var m model.StreamPush
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return toDomainStreamPush(&m), nil
}

func (r *StreamPushRepository) GetByAppStream(app, stream string) (*domainstreampush.StreamPush, error) {
	var m model.StreamPush
	if err := r.db.Where("app = ? AND stream = ?", app, stream).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainStreamPush(&m), nil
}

func (r *StreamPushRepository) List(page, count int, query string, pushing *bool, mediaServerID string) ([]domainstreampush.View, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.StreamPush{})
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("app LIKE ? OR stream LIKE ? OR gb_code LIKE ? OR gb_name LIKE ?", like, like, like, like)
	}
	if pushing != nil {
		q = q.Where("pushing = ?", *pushing)
	}
	if mediaServerID != "" {
		q = q.Where("media_server_id = ?", mediaServerID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.StreamPush
	if err := q.Order("pushing DESC, create_time DESC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domainstreampush.View, 0, len(rows))
	for i := range rows {
		name := rows[i].GbName
		if strings.TrimSpace(name) == "" {
			name = rows[i].Stream
		}
		out = append(out, domainstreampush.View{
			ID: int(rows[i].ID), App: rows[i].App, Stream: rows[i].Stream,
			MediaServerID: rows[i].MediaServerID, ServerID: rows[i].ServerID,
			PushTime: rows[i].PushTime, CreateTime: rows[i].CreateTime, UpdateTime: rows[i].UpdateTime,
			Pushing: rows[i].Pushing, StartOfflinePush: rows[i].StartOfflinePush,
			GbID: 0, GbDeviceID: rows[i].GbCode, GbName: rows[i].GbName,
			Name: name, DataType: shared.ChannelDataTypeStreamPush,
		})
	}
	return out, total, nil
}

func (r *StreamPushRepository) UpdatePushing(id int, pushing bool, mediaServerID string) error {
	return r.db.Model(&model.StreamPush{}).Where("id = ?", id).Updates(map[string]any{
		"pushing": pushing, "media_server_id": mediaServerID, "update_time": nowTimeStr(),
	}).Error
}

// UpsertGBChannel 推流外出国标码：直接记在推流行上（统一模型下不再造无主通道）。
func (r *StreamPushRepository) UpsertGBChannel(pushID int, gbDeviceID, gbName string) error {
	return r.db.Model(&model.StreamPush{}).Where("id = ?", pushID).
		Updates(map[string]any{"gb_code": gbDeviceID, "gb_name": gbName, "update_time": nowTimeStr()}).Error
}

func (r *StreamPushRepository) RemoveGBChannel(pushID int) error {
	return r.db.Model(&model.StreamPush{}).Where("id = ?", pushID).
		Updates(map[string]any{"gb_code": "", "gb_name": "", "update_time": nowTimeStr()}).Error
}

// --- Stream Proxy ---

type StreamProxyRepository struct{ db *gorm.DB }

func NewStreamProxyRepository(db *gorm.DB) *StreamProxyRepository {
	return &StreamProxyRepository{db: db}
}

func (r *StreamProxyRepository) Create(d *domainstreamproxy.StreamProxy) error {
	m := toModelStreamProxy(d)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	d.ID = int(m.ID)
	return nil
}

func (r *StreamProxyRepository) Update(d *domainstreamproxy.StreamProxy) error {
	return r.db.Save(toModelStreamProxy(d)).Error
}

func (r *StreamProxyRepository) Delete(id int) error {
	return r.db.Delete(&model.StreamProxy{}, id).Error
}

func (r *StreamProxyRepository) GetByID(id int) (*domainstreamproxy.StreamProxy, error) {
	var m model.StreamProxy
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return toDomainStreamProxy(&m), nil
}

func (r *StreamProxyRepository) List(page, count int, query string, pulling *bool, mediaServerID string) ([]domainstreamproxy.StreamProxy, int64, error) {
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
	out := make([]domainstreamproxy.StreamProxy, 0, len(rows))
	for i := range rows {
		out = append(out, *toDomainStreamProxy(&rows[i]))
	}
	return out, total, nil
}

func (r *StreamProxyRepository) GetByAppStream(app, stream string) (*domainstreamproxy.StreamProxy, error) {
	var m model.StreamProxy
	if err := r.db.Where("app = ? AND stream = ?", app, stream).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainStreamProxy(&m), nil
}

func (r *StreamProxyRepository) UpsertGBChannel(proxyID int, gbDeviceID, gbName, app, stream string) error {
	return r.db.Model(&model.StreamProxy{}).Where("id = ?", proxyID).
		Updates(map[string]any{"gb_code": gbDeviceID, "gb_name": gbName, "update_time": nowTimeStr()}).Error
}

// --- Cloud Record ---

type CloudRecordRepository struct{ db *gorm.DB }

func NewCloudRecordRepository(db *gorm.DB) *CloudRecordRepository {
	return &CloudRecordRepository{db: db}
}

func (r *CloudRecordRepository) Create(rec *domaincloudrecord.Record) error {
	m := toModelCloudRecord(rec)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	rec.ID = int(m.ID)
	return nil
}

func (r *CloudRecordRepository) Delete(ids []int) error {
	return r.db.Where("id IN ?", ids).Delete(&model.CloudRecord{}).Error
}

func (r *CloudRecordRepository) ListByIDs(ids []int) ([]domaincloudrecord.Record, error) {
	out := make([]domaincloudrecord.Record, 0, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []model.CloudRecord
	if err := r.db.Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		out = append(out, *toDomainCloudRecord(&rows[i]))
	}
	return out, nil
}

func (r *CloudRecordRepository) GetByID(id int) (*domaincloudrecord.Record, error) {
	var m model.CloudRecord
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return toDomainCloudRecord(&m), nil
}

func (r *CloudRecordRepository) List(page, count int, app, stream, query, callID, mediaServerID string, startTime, endTime int64, asc bool) ([]domaincloudrecord.Record, int64, error) {
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
		sec, ms := startTime, startTime
		if startTime >= 1_000_000_000_000 {
			sec = startTime / 1000
		} else {
			ms = startTime * 1000
		}
		q = q.Where("start_time >= ? OR start_time >= ?", sec, ms)
	}
	if endTime > 0 {
		sec, ms := endTime, endTime
		if endTime >= 1_000_000_000_000 {
			sec = endTime / 1000
		} else {
			ms = endTime * 1000
		}
		q = q.Where("start_time <= ? OR start_time <= ?", sec, ms)
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
	if err := q.Order(order).Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domaincloudrecord.Record, 0, len(rows))
	for i := range rows {
		out = append(out, *toDomainCloudRecord(&rows[i]))
	}
	return out, total, nil
}

func toModelCloudRecord(d *domaincloudrecord.Record) *model.CloudRecord {
	return &model.CloudRecord{
		ID: int64(d.ID), App: d.App, Stream: d.Stream, CallID: d.CallID,
		StartTime: d.StartTime, EndTime: d.EndTime,
		MediaServerID: d.MediaServerID, ServerID: d.ServerID,
		FileName: d.FileName, Folder: d.Folder, FilePath: d.FilePath,
		PlayURL: d.PlayURL, Collect: d.Collect, FileSize: d.FileSize, TimeLen: d.TimeLen,
		StorageNodeID: d.StorageNodeID, ContainerFormat: d.ContainerFormat,
		SlotID: d.SlotID, Status: d.Status,
	}
}

func toDomainCloudRecord(m *model.CloudRecord) *domaincloudrecord.Record {
	return &domaincloudrecord.Record{
		ID: int(m.ID), App: m.App, Stream: m.Stream, CallID: m.CallID,
		StartTime: m.StartTime, EndTime: m.EndTime,
		MediaServerID: m.MediaServerID, ServerID: m.ServerID,
		FileName: m.FileName, Folder: m.Folder, FilePath: m.FilePath,
		PlayURL: m.PlayURL, Collect: m.Collect, FileSize: m.FileSize, TimeLen: m.TimeLen,
		StorageNodeID: m.StorageNodeID, ContainerFormat: m.ContainerFormat,
		SlotID: m.SlotID, Status: m.Status,
	}
}

// UpsertPs 按 (storage_node_id, slot_id) 幂等写入 PS 旁路录像。
func (r *CloudRecordRepository) UpsertPs(rec *domaincloudrecord.Record) error {
	m := toModelCloudRecord(rec)
	if m.ContainerFormat == "" {
		m.ContainerFormat = "ps"
	}
	var existing model.CloudRecord
	err := r.db.Where("storage_node_id = ? AND slot_id = ?", m.StorageNodeID, m.SlotID).
		First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(m).Error
	}
	if err != nil {
		return err
	}
	m.ID = existing.ID
	if m.Collect {
		m.Collect = existing.Collect
	}
	return r.db.Save(m).Error
}

// ListSegments 返回某流在 [startMs,endMs] 内与之重叠的云端片段（PS+MP4），按开始时间升序。
func (r *CloudRecordRepository) ListSegments(stream string, startMs, endMs int64) ([]domaincloudrecord.Record, error) {
	q := r.db.Model(&model.CloudRecord{}).Where("app = ?", "live").Where("stream = ?", stream)
	if startMs > 0 {
		q = q.Where("end_time >= ?", startMs)
	}
	if endMs > 0 {
		q = q.Where("start_time <= ?", endMs)
	}
	var rows []model.CloudRecord
	if err := q.Order("start_time ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domaincloudrecord.Record, 0, len(rows))
	for i := range rows {
		out = append(out, *toDomainCloudRecord(&rows[i]))
	}
	return out, nil
}

func (r *CloudRecordRepository) DateList(app, stream, mediaServerID string, year, month int) ([]string, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)
	startSec, endSec := start.Unix(), end.Unix()
	startMs, endMs := start.UnixMilli(), end.UnixMilli()
	q := r.db.Model(&model.CloudRecord{}).Where("app = ? AND stream = ?", app, stream).
		Where("(start_time >= ? AND start_time < ?) OR (start_time >= ? AND start_time < ?)",
			startSec, endSec, startMs, endMs)
	if mediaServerID != "" {
		q = q.Where("media_server_id = ?", mediaServerID)
	}
	var rows []model.CloudRecord
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	set := make(map[string]struct{})
	for _, row := range rows {
		ts := row.StartTime
		if ts >= 1_000_000_000_000 {
			ts = ts / 1000
		}
		d := time.Unix(ts, 0).In(time.Local).Format("2006-01-02")
		set[d] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for d := range set {
		out = append(out, d)
	}
	sort.Strings(out)
	return out, nil
}

// --- Record Plan ---

type RecordPlanRepository struct{ db *gorm.DB }

func NewRecordPlanRepository(db *gorm.DB) *RecordPlanRepository { return &RecordPlanRepository{db: db} }

func (r *RecordPlanRepository) Create(plan *domainrecordplan.Plan, items []domainrecordplan.Item) error {
	mp := toModelRecordPlan(plan)
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(mp).Error; err != nil {
			return err
		}
		plan.ID = int(mp.ID)
		for i := range items {
			items[i].PlanID = plan.ID
			mi := toModelRecordPlanItem(&items[i])
			if err := tx.Create(mi).Error; err != nil {
				return err
			}
			items[i].ID = int(mi.ID)
		}
		return nil
	})
}

func (r *RecordPlanRepository) Update(plan *domainrecordplan.Plan, items []domainrecordplan.Item) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(toModelRecordPlan(plan)).Error; err != nil {
			return err
		}
		if err := tx.Where("plan_id = ?", plan.ID).Delete(&model.RecordPlanItem{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].PlanID = plan.ID
			if err := tx.Create(toModelRecordPlanItem(&items[i])).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *RecordPlanRepository) Delete(id int) error {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		_ = tx.Model(&model.ChannelExt{}).Where("record_plan_id = ?", id).Update("record_plan_id", 0)
		_ = tx.Where("plan_id = ?", id).Delete(&model.RecordPlanItem{})
		return tx.Delete(&model.RecordPlan{}, id).Error
	})
	if err != nil {
		return err
	}
	return r.resyncPlanChannels()
}

// resyncPlanChannels 重建「计划↔通道」内码关联表，作为录制的权威来源。
// 关联以 设备内码 D... + 通道内码 C... 表达（非自增 id、非国标码）。
// channel_ext.record_plan_id 仍作为列表/筛选的兼容缓存，两者随每次写入保持一致。
func (r *RecordPlanRepository) resyncPlanChannels() error {
	if err := r.db.Exec(`DELETE FROM zws_record_plan_channel`).Error; err != nil {
		return err
	}
	return r.db.Exec(`INSERT INTO zws_record_plan_channel (plan_id, device_code, channel_code)
		SELECT e.record_plan_id, COALESCE(d.internal_code,''), c.internal_code
		FROM zws_channel_ext e
		JOIN zws_channel c ON c.id = e.channel_id
		LEFT JOIN zws_device d ON d.id = c.device_id
		WHERE e.record_plan_id > 0 AND c.internal_code <> ''`).Error
}

func (r *RecordPlanRepository) GetByID(id int) (*domainrecordplan.Plan, []domainrecordplan.Item, error) {
	var plan model.RecordPlan
	if err := r.db.First(&plan, id).Error; err != nil {
		return nil, nil, err
	}
	var rows []model.RecordPlanItem
	_ = r.db.Where("plan_id = ?", id).Find(&rows)
	items := make([]domainrecordplan.Item, 0, len(rows))
	for i := range rows {
		items = append(items, *toDomainRecordPlanItem(&rows[i]))
	}
	return toDomainRecordPlan(&plan), items, nil
}

func (r *RecordPlanRepository) List(page, count int, query string) ([]domainrecordplan.Plan, int64, error) {
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
	if err := q.Order("id DESC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domainrecordplan.Plan, 0, len(rows))
	for i := range rows {
		out = append(out, *toDomainRecordPlan(&rows[i]))
	}
	return out, total, nil
}

func (r *RecordPlanRepository) LinkChannels(planID int, channelIDs []int) error {
	if len(channelIDs) == 0 {
		return nil
	}
	if err := r.db.Model(&model.ChannelExt{}).Where("channel_id IN ?", channelIDs).Update("record_plan_id", planID).Error; err != nil {
		return err
	}
	return r.resyncPlanChannels()
}

func (r *RecordPlanRepository) UnlinkChannels(channelIDs []int) error {
	if len(channelIDs) == 0 {
		return nil
	}
	if err := r.db.Model(&model.ChannelExt{}).Where("channel_id IN ?", channelIDs).Update("record_plan_id", 0).Error; err != nil {
		return err
	}
	return r.resyncPlanChannels()
}

func (r *RecordPlanRepository) LinkChannelsByDevice(planID int, deviceDbIDs []int) error {
	if len(deviceDbIDs) == 0 {
		return nil
	}
	if err := r.db.Model(&model.ChannelExt{}).
		Where("channel_id IN (SELECT id FROM zws_channel WHERE device_id IN ?)", deviceDbIDs).
		Update("record_plan_id", planID).Error; err != nil {
		return err
	}
	return r.resyncPlanChannels()
}

func (r *RecordPlanRepository) UnlinkChannelsByDevice(deviceDbIDs []int) error {
	if len(deviceDbIDs) == 0 {
		return nil
	}
	if err := r.db.Model(&model.ChannelExt{}).
		Where("channel_id IN (SELECT id FROM zws_channel WHERE device_id IN ?)", deviceDbIDs).
		Update("record_plan_id", 0).Error; err != nil {
		return err
	}
	return r.resyncPlanChannels()
}

func (r *RecordPlanRepository) LinkAll(planID int) error {
	if err := r.db.Model(&model.ChannelExt{}).Where("1=1").Update("record_plan_id", planID).Error; err != nil {
		return err
	}
	return r.resyncPlanChannels()
}

func (r *RecordPlanRepository) UnlinkAll(planID int) error {
	if err := r.db.Model(&model.ChannelExt{}).Where("record_plan_id = ?", planID).Update("record_plan_id", 0).Error; err != nil {
		return err
	}
	return r.resyncPlanChannels()
}

func (r *RecordPlanRepository) ChannelList(page, count int, planID int, query string, hasLink *bool, online, channelType string) ([]domainrecordplan.ChannelRecord, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Table("zws_channel c").
		Joins("LEFT JOIN zws_channel_ext e ON e.channel_id = c.id").
		Where("c.kind = ?", "video_sensor")
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("(c.name LIKE ? OR c.gb_code LIKE ? OR c.internal_code LIKE ?)", like, like, like)
	}
	if hasLink != nil {
		if *hasLink {
			q = q.Where("e.record_plan_id = ?", planID)
		} else {
			q = q.Where("(e.record_plan_id = 0 OR e.record_plan_id IS NULL OR e.record_plan_id <> ?)", planID)
		}
	}
	if online == "true" {
		q = q.Where("c.status = ?", "ON")
	} else if online == "false" {
		q = q.Where("c.status <> ?", "ON")
	}
	if channelType != "" {
		// 统一模型下所有通道均为视频；保留参数以兼容调用方。
		if _, err := strconv.Atoi(channelType); err != nil {
			_ = err
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		model.Channel
		ExtRecordPlanID    int64   `gorm:"column:ext_record_plan_id"`
		ExtLongitude       float64 `gorm:"column:ext_longitude"`
		ExtLatitude        float64 `gorm:"column:ext_latitude"`
		ExtPTZType         int     `gorm:"column:ext_ptz_type"`
		ExtHasAudio        bool    `gorm:"column:ext_has_audio"`
		ExtGBJSON          string  `gorm:"column:ext_gb_json"`
		ExtStorageNodeID   string  `gorm:"column:ext_storage_node_id"`
		ExtProtocol        string  `gorm:"column:ext_protocol"`
		DeviceInternalCode string  `gorm:"column:device_internal_code"`
		DeviceGbCode       string  `gorm:"column:device_gb_code"`
	}
	err := q.Select(`c.*, COALESCE(e.record_plan_id,0) AS ext_record_plan_id,
		COALESCE(e.longitude,0) AS ext_longitude, COALESCE(e.latitude,0) AS ext_latitude,
		COALESCE(e.ptz_type,0) AS ext_ptz_type, COALESCE(e.has_audio,0) AS ext_has_audio,
		COALESCE(e.gb_json,'') AS ext_gb_json, COALESCE(rp.storage_node_id,'') AS ext_storage_node_id,
		COALESCE(ab.protocol,'') AS ext_protocol, d.internal_code AS device_internal_code,
		COALESCE(d.gb_code,'') AS device_gb_code`).
		Joins("LEFT JOIN zws_device d ON d.id = c.device_id").
		Joins("LEFT JOIN zws_record_plan rp ON rp.id = e.record_plan_id").
		Joins("LEFT JOIN zws_access_binding ab ON ab.device_id = c.device_id").
		Order("c.id ASC").Offset((page - 1) * count).Limit(count).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]domainrecordplan.ChannelRecord, 0, len(rows))
	for i := range rows {
		out = append(out, channelRowToRecord(&rows[i].Channel, int(rows[i].ExtRecordPlanID),
			rows[i].ExtLongitude, rows[i].ExtLatitude, rows[i].ExtPTZType, rows[i].ExtHasAudio, rows[i].ExtGBJSON, rows[i].ExtStorageNodeID, rows[i].ExtProtocol, rows[i].DeviceInternalCode, rows[i].DeviceGbCode))
	}
	return out, total, nil
}

func (r *RecordPlanRepository) CountChannels(planID int) int64 {
	var n int64
	_ = r.db.Model(&model.ChannelExt{}).Where("record_plan_id = ?", planID).Count(&n)
	return n
}

func (r *RecordPlanRepository) QueryRecordingChannels(weekDay, minuteIndex int) ([]domainrecordplan.ChannelRecord, error) {
	var rows []struct {
		model.Channel
		ExtRecordPlanID    int64   `gorm:"column:ext_record_plan_id"`
		ExtLongitude       float64 `gorm:"column:ext_longitude"`
		ExtLatitude        float64 `gorm:"column:ext_latitude"`
		ExtPTZType         int     `gorm:"column:ext_ptz_type"`
		ExtHasAudio        bool    `gorm:"column:ext_has_audio"`
		ExtGBJSON          string  `gorm:"column:ext_gb_json"`
		ExtStorageNodeID   string  `gorm:"column:ext_storage_node_id"`
		ExtProtocol        string  `gorm:"column:ext_protocol"`
		DeviceInternalCode string  `gorm:"column:device_internal_code"`
		DeviceGbCode       string  `gorm:"column:device_gb_code"`
	}
	err := r.db.Table("zws_record_plan_channel rpc").
		Select(`c.*, rpc.plan_id AS ext_record_plan_id, COALESCE(e.longitude,0) AS ext_longitude,
			COALESCE(e.latitude,0) AS ext_latitude, COALESCE(e.ptz_type,0) AS ext_ptz_type,
			COALESCE(e.has_audio,0) AS ext_has_audio, COALESCE(e.gb_json,'') AS ext_gb_json,
			COALESCE(rp.storage_node_id,'') AS ext_storage_node_id,
			COALESCE(ab.protocol,'') AS ext_protocol,
			d.internal_code AS device_internal_code,
			COALESCE(d.gb_code,'') AS device_gb_code`).
		Joins("JOIN zws_record_plan_item wrpi ON wrpi.plan_id = rpc.plan_id").
		Joins("JOIN zws_channel c ON c.internal_code = rpc.channel_code").
		Joins("LEFT JOIN zws_device d ON d.internal_code = rpc.device_code").
		Joins("LEFT JOIN zws_channel_ext e ON e.channel_id = c.id").
		Joins("LEFT JOIN zws_record_plan rp ON rp.id = rpc.plan_id").
		Joins("LEFT JOIN zws_access_binding ab ON ab.device_id = c.device_id").
		Where("wrpi.week_day = ? AND wrpi.start_min <= ? AND wrpi.stop_min >= ?", weekDay, minuteIndex, minuteIndex).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domainrecordplan.ChannelRecord, 0, len(rows))
	for i := range rows {
		out = append(out, channelRowToRecord(&rows[i].Channel, int(rows[i].ExtRecordPlanID),
			rows[i].ExtLongitude, rows[i].ExtLatitude, rows[i].ExtPTZType, rows[i].ExtHasAudio, rows[i].ExtGBJSON, rows[i].ExtStorageNodeID, rows[i].ExtProtocol, rows[i].DeviceInternalCode, rows[i].DeviceGbCode))
	}
	return out, nil
}

// channelRowToRecord 把统一通道行拼成录像计划用的扁平 ChannelRecord。
func channelRowToRecord(m *model.Channel, recordPlanID int, lon, lat float64, ptz int, hasAudio bool, gbJSON, storageNodeID, protocol, deviceInternalCode, deviceGbCode string) domainrecordplan.ChannelRecord {
	var g gbChannelJSON
	if strings.TrimSpace(gbJSON) != "" {
		_ = json.Unmarshal([]byte(gbJSON), &g)
	}
	dataType := shared.ChannelDataTypeGB28181
	if protocol == "onvif" || protocol == "plugin" {
		dataType = shared.ChannelDataTypeStreamPush
	}
	return domainrecordplan.ChannelRecord{
		ID: int(m.ID), InternalCode: m.InternalCode, DeviceInternalCode: deviceInternalCode,
		Name: m.Name, Status: m.Status, Longitude: lon, Latitude: lat, PTZType: ptz,
		HasAudio: hasAudio, RecordPlanID: recordPlanID, StorageNodeID: storageNodeID, Protocol: protocol,
		DeviceID: deviceGbCode, GBDeviceID: m.GbCode,
		CivilCode: g.CivilCode, ParentID: g.ParentID, BusinessGroupID: g.BusinessGroupID,
		Manufacturer: g.Manufacturer, Model: g.Model, Parental: g.Parental, SubCount: g.SubCount,
		DataType: dataType, DataDeviceID: int(m.DeviceID),
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func toModelStreamPush(d *domainstreampush.StreamPush) *model.StreamPush {
	return &model.StreamPush{
		ID: int64(d.ID), App: d.App, Stream: d.Stream, MediaServerID: d.MediaServerID,
		ServerID: d.ServerID, PushTime: d.PushTime, Status: d.Status, UpdateTime: d.UpdateTime,
		CreateTime: d.CreateTime, Pushing: d.Pushing, StartOfflinePush: d.StartOfflinePush,
	}
}

func toDomainStreamPush(m *model.StreamPush) *domainstreampush.StreamPush {
	return &domainstreampush.StreamPush{
		ID: int(m.ID), App: m.App, Stream: m.Stream, MediaServerID: m.MediaServerID,
		ServerID: m.ServerID, PushTime: m.PushTime, Status: m.Status, UpdateTime: m.UpdateTime,
		CreateTime: m.CreateTime, Pushing: m.Pushing, StartOfflinePush: m.StartOfflinePush,
	}
}

func toModelStreamProxy(d *domainstreamproxy.StreamProxy) *model.StreamProxy {
	return &model.StreamProxy{
		ID: int64(d.ID), Type: d.Type, App: d.App, Stream: d.Stream, SrcURL: d.SrcURL, Timeout: d.Timeout,
		FFmpegCmdKey: d.FFmpegCmdKey, RTSPType: d.RTSPType, MediaServerID: d.MediaServerID,
		EnableAudio: d.EnableAudio, EnableMP4: d.EnableMP4, Pulling: d.Pulling, Enable: d.Enable,
		CreateTime: d.CreateTime, Name: d.Name, UpdateTime: d.UpdateTime, StreamKey: d.StreamKey,
		ServerID: d.ServerID, EnableDisableNoneReader: d.EnableDisableNoneReader,
		RelatesMediaServerID: d.RelatesMediaServerID,
	}
}

func toDomainStreamProxy(m *model.StreamProxy) *domainstreamproxy.StreamProxy {
	return &domainstreamproxy.StreamProxy{
		ID: int(m.ID), Type: m.Type, App: m.App, Stream: m.Stream, SrcURL: m.SrcURL, Timeout: m.Timeout,
		FFmpegCmdKey: m.FFmpegCmdKey, RTSPType: m.RTSPType, MediaServerID: m.MediaServerID,
		EnableAudio: m.EnableAudio, EnableMP4: m.EnableMP4, Pulling: m.Pulling, Enable: m.Enable,
		CreateTime: m.CreateTime, Name: m.Name, UpdateTime: m.UpdateTime, StreamKey: m.StreamKey,
		ServerID: m.ServerID, EnableDisableNoneReader: m.EnableDisableNoneReader,
		RelatesMediaServerID: m.RelatesMediaServerID,
	}
}

var (
	_ domainstreampush.Repository  = (*StreamPushRepository)(nil)
	_ domainstreamproxy.Repository = (*StreamProxyRepository)(nil)
)

func toModelRecordPlan(d *domainrecordplan.Plan) *model.RecordPlan {
	return &model.RecordPlan{ID: int64(d.ID), Snap: d.Snap, Name: d.Name, StorageNodeID: d.StorageNodeID, CreateTime: d.CreateTime, UpdateTime: d.UpdateTime}
}

func toDomainRecordPlan(m *model.RecordPlan) *domainrecordplan.Plan {
	return &domainrecordplan.Plan{ID: int(m.ID), Snap: m.Snap, Name: m.Name, StorageNodeID: m.StorageNodeID, CreateTime: m.CreateTime, UpdateTime: m.UpdateTime}
}

func toModelRecordPlanItem(d *domainrecordplan.Item) *model.RecordPlanItem {
	return &model.RecordPlanItem{
		ID: int64(d.ID), Start: d.Start, Stop: d.Stop, WeekDay: d.WeekDay, PlanID: int64(d.PlanID),
		CreateTime: d.CreateTime, UpdateTime: d.UpdateTime,
	}
}

func toDomainRecordPlanItem(m *model.RecordPlanItem) *domainrecordplan.Item {
	return &domainrecordplan.Item{
		ID: int(m.ID), Start: m.Start, Stop: m.Stop, WeekDay: m.WeekDay, PlanID: int(m.PlanID),
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

var _ domainrecordplan.Repository = (*RecordPlanRepository)(nil)
