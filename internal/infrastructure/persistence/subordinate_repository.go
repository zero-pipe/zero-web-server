package persistence

import (
	"fmt"
	"strings"
	"time"

	domainsub "zero-web-server/internal/domain/subordinate"
	"zero-web-server/internal/infrastructure/persistence/model"

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
		"enable":       p.Enable,
		"name":         p.Name,
		"device_gb_id": p.DeviceGBID,
		"password":     p.Password,
		"transport":    p.Transport,
		"update_time":  p.UpdateTime,
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

// ---- 下级目录（级联接收）----

func toCatalogEntry(m *model.SubordinateCatalog) *domainsub.CatalogEntry {
	return &domainsub.CatalogEntry{
		ID: m.ID, PlatformID: m.PlatformID, DeviceID: m.DeviceID, Name: m.Name,
		ParentID: m.ParentID, TypeCode: m.TypeCode, ItemType: m.ItemType,
		Status: m.Status, CivilCode: m.CivilCode, Manufacturer: m.Manufacturer,
		Model: m.Model, Orphan: m.Orphan, RawJSON: m.RawJSON,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func (r *SubordinateRepository) ListCatalog(platformID int) ([]*domainsub.CatalogEntry, error) {
	var rows []model.SubordinateCatalog
	if err := r.db.Where("platform_id = ?", platformID).Order("device_id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainsub.CatalogEntry, 0, len(rows))
	for i := range rows {
		out = append(out, toCatalogEntry(&rows[i]))
	}
	return out, nil
}

func (r *SubordinateRepository) GetCatalog(platformID int, deviceID string) (*domainsub.CatalogEntry, error) {
	var row model.SubordinateCatalog
	if err := r.db.Where("platform_id = ? AND device_id = ?", platformID, deviceID).First(&row).Error; err != nil {
		return nil, err
	}
	return toCatalogEntry(&row), nil
}

// UpsertCatalog 按 (platform_id, device_id) 更新或插入；保留 create_time。
func (r *SubordinateRepository) UpsertCatalog(e *domainsub.CatalogEntry) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	var row model.SubordinateCatalog
	err := r.db.Where("platform_id = ? AND device_id = ?", e.PlatformID, e.DeviceID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		row = model.SubordinateCatalog{
			PlatformID: e.PlatformID, DeviceID: e.DeviceID, CreateTime: now,
		}
		e.CreateTime = now
	} else if err != nil {
		return err
	}
	row.Name = e.Name
	row.ParentID = e.ParentID
	row.TypeCode = e.TypeCode
	row.ItemType = e.ItemType
	row.Status = e.Status
	row.CivilCode = e.CivilCode
	row.Manufacturer = e.Manufacturer
	row.Model = e.Model
	row.Orphan = e.Orphan
	row.RawJSON = e.RawJSON
	row.UpdateTime = now
	e.UpdateTime = now
	if row.ID == 0 {
		return r.db.Create(&row).Error
	}
	return r.db.Save(&row).Error
}

func (r *SubordinateRepository) DeleteCatalog(platformID int, deviceID string) error {
	return r.db.Where("platform_id = ? AND device_id = ?", platformID, deviceID).
		Delete(&model.SubordinateCatalog{}).Error
}
