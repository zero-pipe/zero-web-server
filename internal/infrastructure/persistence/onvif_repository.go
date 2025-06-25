package persistence

import (
	"context"
	"fmt"
	"time"

	domainonvif "zero-web-kit/internal/domain/onvif"
	"zero-web-kit/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type OnvifDeviceRepository struct {
	db *gorm.DB
}

func NewOnvifDeviceRepository(db *gorm.DB) *OnvifDeviceRepository {
	return &OnvifDeviceRepository{db: db}
}

func (r *OnvifDeviceRepository) Create(ctx context.Context, device *domainonvif.Device) error {
	m := toOnvifDeviceModel(device)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	device.ID = m.ID
	return nil
}

func (r *OnvifDeviceRepository) Update(ctx context.Context, device *domainonvif.Device) error {
	m := toOnvifDeviceModel(device)
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *OnvifDeviceRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.OnvifDevice{}, id).Error
}

func (r *OnvifDeviceRepository) GetByID(ctx context.Context, id int64) (*domainonvif.Device, error) {
	var m model.OnvifDevice
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return toOnvifDeviceDomain(&m), nil
}

func (r *OnvifDeviceRepository) List(ctx context.Context, page, count int, keyword string) ([]*domainonvif.Device, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}

	q := r.db.WithContext(ctx).Model(&model.OnvifDevice{})
	if keyword != "" {
		like := fmt.Sprintf("%%%s%%", keyword)
		q = q.Where("name LIKE ? OR ip LIKE ? OR custom_name LIKE ?", like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.OnvifDevice
	offset := (page - 1) * count
	if err := q.Order("id DESC").Offset(offset).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	devices := make([]*domainonvif.Device, 0, len(rows))
	for i := range rows {
		devices = append(devices, toOnvifDeviceDomain(&rows[i]))
	}
	return devices, total, nil
}

func (r *OnvifDeviceRepository) ExistsByIPPort(ctx context.Context, ip string, port int) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.OnvifDevice{}).
		Where("ip = ? AND port = ?", ip, port).Count(&count).Error
	return count > 0, err
}

func (r *OnvifDeviceRepository) UpdateOnlineStatus(ctx context.Context, id int64, online bool) error {
	return r.db.WithContext(ctx).Model(&model.OnvifDevice{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"on_line":     online,
			"update_time": nowStr(),
		}).Error
}

type OnvifChannelRepository struct {
	db *gorm.DB
}

func NewOnvifChannelRepository(db *gorm.DB) *OnvifChannelRepository {
	return &OnvifChannelRepository{db: db}
}

func (r *OnvifChannelRepository) DeleteByDeviceID(ctx context.Context, deviceID int64) error {
	return r.db.WithContext(ctx).Where("device_id = ?", deviceID).Delete(&model.OnvifChannel{}).Error
}

func (r *OnvifChannelRepository) BatchCreate(ctx context.Context, channels []*domainonvif.Channel) error {
	if len(channels) == 0 {
		return nil
	}
	models := make([]model.OnvifChannel, 0, len(channels))
	for _, ch := range channels {
		models = append(models, *toOnvifChannelModel(ch))
	}
	if err := r.db.WithContext(ctx).Create(&models).Error; err != nil {
		return err
	}
	for i := range channels {
		channels[i].ID = models[i].ID
	}
	return nil
}

func (r *OnvifChannelRepository) ListByDeviceID(ctx context.Context, deviceID int64) ([]*domainonvif.Channel, error) {
	var rows []model.OnvifChannel
	if err := r.db.WithContext(ctx).Where("device_id = ?", deviceID).Find(&rows).Error; err != nil {
		return nil, err
	}
	channels := make([]*domainonvif.Channel, 0, len(rows))
	for i := range rows {
		channels = append(channels, toOnvifChannelDomain(&rows[i]))
	}
	return channels, nil
}

func (r *OnvifChannelRepository) GetByID(ctx context.Context, id int64) (*domainonvif.Channel, error) {
	var m model.OnvifChannel
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return toOnvifChannelDomain(&m), nil
}

func (r *OnvifChannelRepository) List(ctx context.Context, page, count int, deviceID int64) ([]*domainonvif.Channel, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}

	q := r.db.WithContext(ctx).Model(&model.OnvifChannel{})
	if deviceID > 0 {
		q = q.Where("device_id = ?", deviceID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.OnvifChannel
	offset := (page - 1) * count
	if err := q.Order("id DESC").Offset(offset).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	channels := make([]*domainonvif.Channel, 0, len(rows))
	for i := range rows {
		channels = append(channels, toOnvifChannelDomain(&rows[i]))
	}
	return channels, total, nil
}

func nowStr() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func toOnvifDeviceModel(d *domainonvif.Device) *model.OnvifDevice {
	return &model.OnvifDevice{
		ID:            d.ID,
		Name:          d.Name,
		IP:            d.IP,
		Port:          d.Port,
		Username:      d.Username,
		Password:      d.Password,
		Manufacturer:  d.Manufacturer,
		Model:         d.Model,
		Firmware:      d.Firmware,
		SerialNumber:  d.SerialNumber,
		HardwareID:    d.HardwareID,
		DeviceURI:     d.DeviceURI,
		MediaURI:      d.MediaURI,
		PTZURI:        d.PTZURI,
		OnLine:        d.OnLine,
		DiscoveryMode: d.DiscoveryMode,
		MediaServerID: d.MediaServerID,
		CustomName:    d.CustomName,
		ServerID:      d.ServerID,
		CreateTime:    d.CreateTime,
		UpdateTime:    d.UpdateTime,
	}
}

func toOnvifDeviceDomain(m *model.OnvifDevice) *domainonvif.Device {
	return &domainonvif.Device{
		ID:            m.ID,
		Name:          m.Name,
		IP:            m.IP,
		Port:          m.Port,
		Username:      m.Username,
		Password:      m.Password,
		Manufacturer:  m.Manufacturer,
		Model:         m.Model,
		Firmware:      m.Firmware,
		SerialNumber:  m.SerialNumber,
		HardwareID:    m.HardwareID,
		DeviceURI:     m.DeviceURI,
		MediaURI:      m.MediaURI,
		PTZURI:        m.PTZURI,
		OnLine:        m.OnLine,
		DiscoveryMode: m.DiscoveryMode,
		MediaServerID: m.MediaServerID,
		CustomName:    m.CustomName,
		ServerID:      m.ServerID,
		CreateTime:    m.CreateTime,
		UpdateTime:    m.UpdateTime,
	}
}

func toOnvifChannelModel(c *domainonvif.Channel) *model.OnvifChannel {
	return &model.OnvifChannel{
		ID:           c.ID,
		DeviceID:     c.DeviceID,
		ProfileToken: c.ProfileToken,
		Name:         c.Name,
		VideoSource:  c.VideoSource,
		EncoderToken: c.EncoderToken,
		Resolution:   c.Resolution,
		Codec:        c.Codec,
		HasAudio:     c.HasAudio,
		HasPTZ:       c.HasPTZ,
		StreamURI:    c.StreamURI,
		Status:       c.Status,
		CreateTime:   c.CreateTime,
		UpdateTime:   c.UpdateTime,
	}
}

func toOnvifChannelDomain(m *model.OnvifChannel) *domainonvif.Channel {
	return &domainonvif.Channel{
		ID:           m.ID,
		DeviceID:     m.DeviceID,
		ProfileToken: m.ProfileToken,
		Name:         m.Name,
		VideoSource:  m.VideoSource,
		EncoderToken: m.EncoderToken,
		Resolution:   m.Resolution,
		Codec:        m.Codec,
		HasAudio:     m.HasAudio,
		HasPTZ:       m.HasPTZ,
		StreamURI:    m.StreamURI,
		Status:       m.Status,
		CreateTime:   m.CreateTime,
		UpdateTime:   m.UpdateTime,
	}
}
