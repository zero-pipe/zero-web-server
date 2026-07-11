package persistence

import (
	"fmt"
	"time"

	domainchannel "zero-web-kit/internal/domain/channel"
	domaindevice "zero-web-kit/internal/domain/device"
	"zero-web-kit/internal/domain/shared"
	"zero-web-kit/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type DeviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) GetByDeviceID(deviceID string) (*domaindevice.Device, error) {
	var m model.GBDevice
	if err := r.db.Where("device_id = ?", deviceID).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainDevice(&m), nil
}

func (r *DeviceRepository) GetByID(id int) (*domaindevice.Device, error) {
	var m model.GBDevice
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return toDomainDevice(&m), nil
}

func (r *DeviceRepository) Create(device *domaindevice.Device) error {
	m := toModelDevice(device)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	device.ID = m.ID
	return nil
}

func (r *DeviceRepository) Update(device *domaindevice.Device) error {
	m := toModelDevice(device)
	return r.db.Save(m).Error
}

func (r *DeviceRepository) UpdateOnline(deviceID string, online bool) error {
	return r.db.Model(&model.GBDevice{}).Where("device_id = ?", deviceID).
		Updates(map[string]any{
			"on_line":     online,
			"update_time": nowTimeStr(),
		}).Error
}

func (r *DeviceRepository) DeleteByDeviceID(deviceID string) error {
	return r.db.Where("device_id = ?", deviceID).Delete(&model.GBDevice{}).Error
}

func (r *DeviceRepository) List(page, count int, query string, online *bool) ([]*domaindevice.Device, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Table("zws_device de").Select(`de.*,
		(SELECT COUNT(0) FROM zws_device_channel dc WHERE dc.data_type = 1 AND dc.data_device_id = de.id) AS channel_count`)
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("de.device_id LIKE ? OR de.name LIKE ? OR de.custom_name LIKE ? OR de.ip LIKE ?", like, like, like, like)
	}
	if online != nil {
		q = q.Where("de.on_line = ?", *online)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	type deviceRow struct {
		model.GBDevice
		ChannelCount int `gorm:"column:channel_count"`
	}
	var rows []deviceRow
	if err := q.Order("de.id DESC").Offset((page - 1) * count).Limit(count).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	list := make([]*domaindevice.Device, 0, len(rows))
	for i := range rows {
		d := toDomainDevice(&rows[i].GBDevice)
		d.ChannelCount = rows[i].ChannelCount
		list = append(list, d)
	}
	return list, total, nil
}

func (r *DeviceRepository) ListOnline() ([]*domaindevice.Device, error) {
	var rows []model.GBDevice
	if err := r.db.Where("on_line = ?", true).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*domaindevice.Device, 0, len(rows))
	for i := range rows {
		list = append(list, toDomainDevice(&rows[i]))
	}
	return list, nil
}

type ChannelRepository struct {
	db *gorm.DB
}

func NewChannelRepository(db *gorm.DB) *ChannelRepository {
	return &ChannelRepository{db: db}
}

func (r *ChannelRepository) GetOne(deviceID, channelDeviceID string) (*domainchannel.Channel, error) {
	var m model.GBDeviceChannel
	err := r.db.Where("device_id = ? AND gb_device_id = ?", deviceID, channelDeviceID).First(&m).Error
	if err != nil {
		return nil, err
	}
	return toDomainChannel(&m), nil
}

func (r *ChannelRepository) GetByID(id int) (*domainchannel.Channel, error) {
	var m model.GBDeviceChannel
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return toDomainChannel(&m), nil
}

func (r *ChannelRepository) ListByDevice(deviceID string, page, count int, query string, online *bool) ([]*domainchannel.Channel, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.GBDeviceChannel{}).Where("device_id = ?", deviceID)
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("name LIKE ? OR gb_device_id LIKE ?", like, like)
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
	var rows []model.GBDeviceChannel
	if err := q.Order("id ASC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	list := make([]*domainchannel.Channel, 0, len(rows))
	for i := range rows {
		list = append(list, toDomainChannel(&rows[i]))
	}
	return list, total, nil
}

func (r *ChannelRepository) ResetByDevice(deviceID string, dataDeviceID int, channels []*domainchannel.Channel) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("device_id = ?", deviceID).Delete(&model.GBDeviceChannel{}).Error; err != nil {
			return err
		}
		if len(channels) == 0 {
			return nil
		}
		models := make([]model.GBDeviceChannel, 0, len(channels))
		for _, ch := range channels {
			models = append(models, model.GBDeviceChannel{
				DeviceID:     deviceID,
				DataType:     shared.ChannelDataTypeGB28181,
				DataDeviceID: dataDeviceID,
				GBDeviceID:   ch.GBDeviceID,
				Name:         ch.Name,
				Manufacturer: ch.Manufacturer,
				Model:        ch.Model,
				Parental:     ch.Parental,
				ParentID:     ch.ParentID,
				Status:       ch.Status,
				Longitude:    ch.Longitude,
				Latitude:     ch.Latitude,
				PTZType:      ch.PTZType,
				HasAudio:     ch.HasAudio,
				SubCount:     ch.SubCount,
				CreateTime:   ch.CreateTime,
				UpdateTime:   ch.UpdateTime,
			})
		}
		return tx.Create(&models).Error
	})
}

func (r *ChannelRepository) Update(m *model.GBDeviceChannel) error {
	m.UpdateTime = nowTimeStr()
	return r.db.Save(m).Error
}

func (r *ChannelRepository) ListCommon(page, count int, query string, channelType *int, online *bool, hasRecordPlan *bool) ([]*domainchannel.Channel, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.GBDeviceChannel{})
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("name LIKE ? OR gb_device_id LIKE ? OR device_id LIKE ?", like, like, like)
	}
	if channelType != nil {
		q = q.Where("data_type = ?", *channelType)
	}
	if online != nil {
		if *online {
			q = q.Where("status = ?", "ON")
		} else {
			q = q.Where("status <> ?", "ON")
		}
	}
	if hasRecordPlan != nil {
		if *hasRecordPlan {
			q = q.Where("record_plan_id > 0")
		} else {
			q = q.Where("record_plan_id = 0 OR record_plan_id IS NULL")
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.GBDeviceChannel
	if err := q.Order("id ASC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	list := make([]*domainchannel.Channel, 0, len(rows))
	for i := range rows {
		list = append(list, toDomainChannel(&rows[i]))
	}
	return list, total, nil
}

func (r *ChannelRepository) ListAll(page, count int, query string) ([]*domainchannel.Channel, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.GBDeviceChannel{})
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("name LIKE ? OR gb_device_id LIKE ? OR device_id LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.GBDeviceChannel
	if err := q.Order("id ASC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	list := make([]*domainchannel.Channel, 0, len(rows))
	for i := range rows {
		list = append(list, toDomainChannel(&rows[i]))
	}
	return list, total, nil
}

func (r *ChannelRepository) ListIDsByDevice(deviceID string) ([]int, error) {
	var ids []int
	err := r.db.Model(&model.GBDeviceChannel{}).Where("device_id = ?", deviceID).Pluck("id", &ids).Error
	return ids, err
}

func (r *ChannelRepository) DeleteByDevice(deviceID string) error {
	return r.db.Where("device_id = ?", deviceID).Delete(&model.GBDeviceChannel{}).Error
}

func (r *ChannelRepository) ChangeAudio(channelID int, audio bool) error {
	return r.db.Model(&model.GBDeviceChannel{}).Where("id = ?", channelID).
		Update("has_audio", audio).Error
}

func (r *ChannelRepository) CountCommonByIDs(channelIDs []int) (int64, error) {
	if len(channelIDs) == 0 {
		return 0, nil
	}
	var total int64
	err := r.db.Model(&model.GBDeviceChannel{}).
		Where("channel_type = 0 AND id IN ?", channelIDs).
		Count(&total).Error
	return total, err
}

func (r *ChannelRepository) ListByCivilCode(page, count int, query string, channelType *int, online *bool, civilCode string) ([]*domainchannel.Channel, int64, error) {
	code := civilCode
	return r.listCommonByAssociation(page, count, query, channelType, online, &code, nil)
}

func (r *ChannelRepository) ListByGroupParent(page, count int, query string, channelType *int, online *bool, groupDeviceID string) ([]*domainchannel.Channel, int64, error) {
	parent := groupDeviceID
	return r.listCommonByAssociation(page, count, query, channelType, online, nil, &parent)
}

func (r *ChannelRepository) listCommonByAssociation(page, count int, query string, channelType *int, online *bool, civilCode, groupDeviceID *string) ([]*domainchannel.Channel, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.GBDeviceChannel{}).Where("channel_type = 0")
	if query != "" {
		like := fmt.Sprintf("%%%s%%", query)
		q = q.Where("COALESCE(gb_device_id, device_id) LIKE ? OR COALESCE(gb_name, name) LIKE ?", like, like)
	}
	if channelType != nil {
		q = q.Where("data_type = ?", *channelType)
	}
	if online != nil {
		if *online {
			q = q.Where("COALESCE(gb_status, status) = ?", "ON")
		} else {
			q = q.Where("COALESCE(gb_status, status) <> ?", "ON")
		}
	}
	// 平台挂载只用 gb_*；parent_id/civil_code 可能是设备目录同步写入的国标字段，
	// 若用 COALESCE 会把已上线但未挂载的通道误判为「已挂载」，添加通道列表为空。
	if civilCode != nil {
		if *civilCode == "" {
			q = q.Where("(gb_civil_code IS NULL OR gb_civil_code = '')")
		} else {
			q = q.Where("gb_civil_code = ?", *civilCode)
		}
	}
	if groupDeviceID != nil {
		if *groupDeviceID == "" {
			q = q.Where("(gb_parent_id IS NULL OR gb_parent_id = '')")
		} else {
			q = q.Where("gb_parent_id = ?", *groupDeviceID)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.GBDeviceChannel
	if err := q.Order("id ASC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	list := make([]*domainchannel.Channel, 0, len(rows))
	for i := range rows {
		list = append(list, toDomainChannel(&rows[i]))
	}
	return list, total, nil
}

func (r *ChannelRepository) SetCivilCode(civilCode string, channelIDs []int) error {
	if len(channelIDs) == 0 {
		return fmt.Errorf("通道ID不可为空")
	}
	return r.db.Exec(
		`UPDATE zws_device_channel SET gb_civil_code = ?, update_time = ? WHERE channel_type = 0 AND id IN ?`,
		civilCode, nowTimeStr(), channelIDs,
	).Error
}

func (r *ChannelRepository) ClearCivilCode(channelIDs []int) error {
	if len(channelIDs) == 0 {
		return fmt.Errorf("通道ID不可为空")
	}
	return r.db.Exec(
		`UPDATE zws_device_channel SET gb_civil_code = NULL, civil_code = NULL, update_time = ? WHERE channel_type = 0 AND id IN ?`,
		nowTimeStr(), channelIDs,
	).Error
}

func (r *ChannelRepository) SetGroup(parentID, businessGroup string, channelIDs []int) error {
	if len(channelIDs) == 0 {
		return fmt.Errorf("通道ID不可为空")
	}
	return r.db.Exec(
		`UPDATE zws_device_channel SET gb_parent_id = ?, gb_business_group_id = ?, update_time = ? WHERE channel_type = 0 AND id IN ?`,
		parentID, businessGroup, nowTimeStr(), channelIDs,
	).Error
}

func (r *ChannelRepository) ClearGroupParent(channelIDs []int) error {
	if len(channelIDs) == 0 {
		return fmt.Errorf("通道ID不可为空")
	}
	return r.db.Exec(
		`UPDATE zws_device_channel SET gb_parent_id = NULL, gb_business_group_id = NULL, update_time = ? WHERE channel_type = 0 AND id IN ?`,
		nowTimeStr(), channelIDs,
	).Error
}

func (r *ChannelRepository) SetCivilCodeByDataDeviceIDs(civilCode string, dataDeviceIDs []int) error {
	if len(dataDeviceIDs) == 0 {
		return fmt.Errorf("设备ID不可为空")
	}
	return r.db.Exec(
		`UPDATE zws_device_channel SET gb_civil_code = ?, update_time = ? WHERE channel_type = 0 AND data_type = 1 AND data_device_id IN ?`,
		civilCode, nowTimeStr(), dataDeviceIDs,
	).Error
}

func (r *ChannelRepository) ClearCivilCodeByDataDeviceIDs(dataDeviceIDs []int) error {
	if len(dataDeviceIDs) == 0 {
		return fmt.Errorf("设备ID不可为空")
	}
	return r.db.Exec(
		`UPDATE zws_device_channel SET gb_civil_code = NULL, civil_code = NULL, update_time = ? WHERE channel_type = 0 AND data_type = 1 AND data_device_id IN ?`,
		nowTimeStr(), dataDeviceIDs,
	).Error
}

func (r *ChannelRepository) SetGroupByDataDeviceIDs(parentID, businessGroup string, dataDeviceIDs []int) error {
	if len(dataDeviceIDs) == 0 {
		return fmt.Errorf("设备ID不可为空")
	}
	return r.db.Exec(
		`UPDATE zws_device_channel SET gb_parent_id = ?, gb_business_group_id = ?, update_time = ? WHERE channel_type = 0 AND data_type = 1 AND data_device_id IN ?`,
		parentID, businessGroup, nowTimeStr(), dataDeviceIDs,
	).Error
}

func (r *ChannelRepository) ClearGroupByDataDeviceIDs(dataDeviceIDs []int) error {
	if len(dataDeviceIDs) == 0 {
		return fmt.Errorf("设备ID不可为空")
	}
	return r.db.Exec(
		`UPDATE zws_device_channel SET gb_parent_id = NULL, gb_business_group_id = NULL, update_time = ? WHERE channel_type = 0 AND data_type = 1 AND data_device_id IN ?`,
		nowTimeStr(), dataDeviceIDs,
	).Error
}

func (r *ChannelRepository) CountCommonByDataDeviceIDs(dataDeviceIDs []int) (int64, error) {
	if len(dataDeviceIDs) == 0 {
		return 0, nil
	}
	var cnt int64
	err := r.db.Raw(
		`SELECT COUNT(1) FROM zws_device_channel WHERE channel_type = 0 AND data_type = 1 AND data_device_id IN ?`,
		dataDeviceIDs,
	).Scan(&cnt).Error
	return cnt, err
}

func nowTimeStr() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func toDomainDevice(m *model.GBDevice) *domaindevice.Device {
	return &domaindevice.Device{
		ID:                       m.ID,
		DeviceID:                 m.DeviceID,
		Name:                     m.Name,
		Manufacturer:             m.Manufacturer,
		Model:                    m.Model,
		Firmware:                 m.Firmware,
		Transport:                m.Transport,
		StreamMode:               m.StreamMode,
		IP:                       m.IP,
		Port:                     m.Port,
		HostAddress:              m.HostAddress,
		OnLine:                   m.OnLine,
		Charset:                  m.Charset,
		Expires:                  m.Expires,
		Password:                 m.Password,
		MediaServerID:            m.MediaServerID,
		CustomName:               m.CustomName,
		SDPIP:                    m.SDPIP,
		LocalIP:                  m.LocalIP,
		ServerID:                 m.ServerID,
		HeartBeatInterval:        m.HeartBeatInterval,
		HeartBeatCount:           m.HeartBeatCount,
		SubscribeCycleForCatalog:         m.SubscribeCycleForCatalog,
		SubscribeCycleForMobilePosition:  m.SubscribeCycleForMobilePosition,
		MobilePositionSubmissionInterval: m.MobilePositionSubmissionInterval,
		SubscribeCycleForAlarm:           m.SubscribeCycleForAlarm,
		CreateTime:               m.CreateTime,
		UpdateTime:               m.UpdateTime,
	}
}

func toModelDevice(d *domaindevice.Device) *model.GBDevice {
	return &model.GBDevice{
		ID:                       d.ID,
		DeviceID:                 d.DeviceID,
		Name:                     d.Name,
		Manufacturer:             d.Manufacturer,
		Model:                    d.Model,
		Firmware:                 d.Firmware,
		Transport:                d.Transport,
		StreamMode:               d.StreamMode,
		OnLine:                   d.OnLine,
		IP:                       d.IP,
		Port:                     d.Port,
		Expires:                  d.Expires,
		HostAddress:              d.HostAddress,
		Charset:                  d.Charset,
		MediaServerID:            d.MediaServerID,
		CustomName:               d.CustomName,
		SDPIP:                    d.SDPIP,
		LocalIP:                  d.LocalIP,
		Password:                 d.Password,
		HeartBeatInterval:        d.HeartBeatInterval,
		HeartBeatCount:           d.HeartBeatCount,
		SubscribeCycleForCatalog:         d.SubscribeCycleForCatalog,
		SubscribeCycleForMobilePosition:  d.SubscribeCycleForMobilePosition,
		MobilePositionSubmissionInterval: d.MobilePositionSubmissionInterval,
		SubscribeCycleForAlarm:           d.SubscribeCycleForAlarm,
		ServerID:                 d.ServerID,
		CreateTime:               d.CreateTime,
		UpdateTime:               d.UpdateTime,
	}
}

func toDomainChannel(m *model.GBDeviceChannel) *domainchannel.Channel {
	return &domainchannel.Channel{
		ID:           m.ID,
		DeviceID:     m.DeviceID,
		DataType:     m.DataType,
		DataDeviceID: m.DataDeviceID,
		GBDeviceID:   m.GBDeviceID,
		Name:         m.Name,
		Manufacturer: m.Manufacturer,
		Model:        m.Model,
		Status:       m.Status,
		PTZType:      m.PTZType,
		Parental:     m.Parental,
		ParentID:     m.ParentID,
		Longitude:    m.Longitude,
		Latitude:     m.Latitude,
		HasAudio:     m.HasAudio,
		SubCount:     m.SubCount,
		CreateTime:   m.CreateTime,
		UpdateTime:   m.UpdateTime,
	}
}
