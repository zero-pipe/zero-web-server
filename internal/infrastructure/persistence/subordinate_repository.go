package persistence

import (
	"fmt"
	"strings"

	domainsub "zero-web-kit/internal/domain/subordinate"
	"zero-web-kit/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type SubordinateRepository struct {
	db *gorm.DB
}

func NewSubordinateRepository(db *gorm.DB) *SubordinateRepository {
	return &SubordinateRepository{db: db}
}

func (r *SubordinateRepository) GetByID(id int) (*domainsub.Platform, error) {
	var row model.SubordinatePlatform
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return toSubDomain(&row), nil
}

func (r *SubordinateRepository) GetByGBID(gbID string) (*domainsub.Platform, error) {
	var row model.SubordinatePlatform
	if err := r.db.Where("device_gb_id = ?", gbID).First(&row).Error; err != nil {
		return nil, err
	}
	return toSubDomain(&row), nil
}

func (r *SubordinateRepository) List(page, count int, query string) ([]*domainsub.Platform, int64, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 20
	}
	q := r.db.Model(&model.SubordinatePlatform{})
	if query = strings.TrimSpace(query); query != "" {
		like := "%" + query + "%"
		q = q.Where("name LIKE ? OR device_gb_id LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.SubordinatePlatform
	if err := q.Order("id DESC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*domainsub.Platform, 0, len(rows))
	for i := range rows {
		out = append(out, toSubDomain(&rows[i]))
	}
	return out, total, nil
}

func (r *SubordinateRepository) Create(p *domainsub.Platform) error {
	row := toSubModel(p)
	if err := r.db.Create(row).Error; err != nil {
		return err
	}
	p.ID = row.ID
	return nil
}

func (r *SubordinateRepository) Update(p *domainsub.Platform) error {
	return r.db.Model(&model.SubordinatePlatform{}).Where("id = ?", p.ID).Updates(map[string]any{
		"enable":     p.Enable,
		"name":       p.Name,
		"device_gb_id": p.DeviceGBID,
		"password":   p.Password,
		"transport":  p.Transport,
		"update_time": p.UpdateTime,
	}).Error
}

func (r *SubordinateRepository) Delete(id int) error {
	return r.db.Delete(&model.SubordinatePlatform{}, id).Error
}

func (r *SubordinateRepository) UpdateOnline(gbID, ip string, port, expires int, callID, transport string) error {
	host := ip
	if port > 0 {
		host = fmt.Sprintf("%s:%d", ip, port)
	}
	return r.db.Model(&model.SubordinatePlatform{}).Where("device_gb_id = ?", gbID).Updates(map[string]any{
		"status":           true,
		"ip":               ip,
		"port":             port,
		"host_address":     host,
		"expires":          expires,
		"register_call_id": callID,
		"transport":        transport,
	}).Error
}

func (r *SubordinateRepository) UpdateOffline(gbID string) error {
	return r.db.Model(&model.SubordinatePlatform{}).Where("device_gb_id = ?", gbID).Updates(map[string]any{
		"status": false,
	}).Error
}

func toSubDomain(row *model.SubordinatePlatform) *domainsub.Platform {
	if row == nil {
		return nil
	}
	return &domainsub.Platform{
		ID:           row.ID,
		Enable:       row.Enable,
		Name:         row.Name,
		DeviceGBID:   row.DeviceGBID,
		Password:     row.Password,
		Transport:    row.Transport,
		Status:       row.Status,
		IP:           row.IP,
		Port:         row.Port,
		HostAddress:  row.HostAddress,
		Expires:      row.Expires,
		RegisterCall: row.RegisterCall,
		ServerID:     row.ServerID,
		CreateTime:   row.CreateTime,
		UpdateTime:   row.UpdateTime,
	}
}

func toSubModel(p *domainsub.Platform) *model.SubordinatePlatform {
	return &model.SubordinatePlatform{
		ID:           p.ID,
		Enable:       p.Enable,
		Name:         p.Name,
		DeviceGBID:   p.DeviceGBID,
		Password:     p.Password,
		Transport:    p.Transport,
		Status:       p.Status,
		IP:           p.IP,
		Port:         p.Port,
		HostAddress:  p.HostAddress,
		Expires:      p.Expires,
		RegisterCall: p.RegisterCall,
		ServerID:     p.ServerID,
		CreateTime:   p.CreateTime,
		UpdateTime:   p.UpdateTime,
	}
}
