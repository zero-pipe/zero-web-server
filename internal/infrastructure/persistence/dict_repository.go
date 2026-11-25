package persistence

import (
	"time"

	domaindict "zero-web-server/internal/domain/dict"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

// DictRepository 数据字典仓储。
type DictRepository struct {
	db *gorm.DB
}

func NewDictRepository(db *gorm.DB) *DictRepository {
	return &DictRepository{db: db}
}

func toDomainDictType(m *model.DictType) domaindict.Type {
	return domaindict.Type{
		ID: m.ID, Code: m.Code, Name: m.Name, Remark: m.Remark,
		Builtin: m.Builtin, Sort: m.Sort, CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func toDomainDictItem(m *model.DictItem) domaindict.Item {
	return domaindict.Item{
		ID: m.ID, TypeCode: m.TypeCode, Value: m.Value, Label: m.Label,
		Sort: m.Sort, Enabled: m.Enabled, Builtin: m.Builtin, Remark: m.Remark,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func (r *DictRepository) ListTypes() ([]domaindict.Type, error) {
	var rows []model.DictType
	if err := r.db.Order("sort ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domaindict.Type, len(rows))
	for i := range rows {
		out[i] = toDomainDictType(&rows[i])
	}
	return out, nil
}

func (r *DictRepository) GetTypeByCode(code string) (*domaindict.Type, error) {
	var m model.DictType
	if err := r.db.Where("code = ?", code).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domaindict.ErrNotFound
		}
		return nil, err
	}
	out := toDomainDictType(&m)
	return &out, nil
}

func (r *DictRepository) GetTypeByID(id int64) (*domaindict.Type, error) {
	var m model.DictType
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domaindict.ErrNotFound
		}
		return nil, err
	}
	out := toDomainDictType(&m)
	return &out, nil
}

func (r *DictRepository) CreateType(t *domaindict.Type) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	m := model.DictType{
		Code: t.Code, Name: t.Name, Remark: t.Remark, Builtin: t.Builtin, Sort: t.Sort,
		CreateTime: now, UpdateTime: now,
	}
	if err := r.db.Create(&m).Error; err != nil {
		return err
	}
	t.ID, t.CreateTime, t.UpdateTime = m.ID, now, now
	return nil
}

func (r *DictRepository) UpdateType(t *domaindict.Type) error {
	return r.db.Model(&model.DictType{}).Where("id = ?", t.ID).Updates(map[string]any{
		"name": t.Name, "remark": t.Remark, "sort": t.Sort,
		"update_time": time.Now().Format("2006-01-02 15:04:05"),
	}).Error
}

func (r *DictRepository) DeleteType(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.DictType{}).Error
}

func (r *DictRepository) ListItems(typeCode string, enabledOnly bool) ([]domaindict.Item, error) {
	q := r.db.Model(&model.DictItem{})
	if typeCode != "" {
		q = q.Where("type_code = ?", typeCode)
	}
	if enabledOnly {
		q = q.Where("enabled = ?", true)
	}
	var rows []model.DictItem
	if err := q.Order("sort ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domaindict.Item, len(rows))
	for i := range rows {
		out[i] = toDomainDictItem(&rows[i])
	}
	return out, nil
}

func (r *DictRepository) GetItemByID(id int64) (*domaindict.Item, error) {
	var m model.DictItem
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domaindict.ErrNotFound
		}
		return nil, err
	}
	out := toDomainDictItem(&m)
	return &out, nil
}

func (r *DictRepository) CreateItem(it *domaindict.Item) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	m := model.DictItem{
		TypeCode: it.TypeCode, Value: it.Value, Label: it.Label, Sort: it.Sort,
		Enabled: it.Enabled, Builtin: it.Builtin, Remark: it.Remark,
		CreateTime: now, UpdateTime: now,
	}
	if err := r.db.Create(&m).Error; err != nil {
		return err
	}
	it.ID, it.CreateTime, it.UpdateTime = m.ID, now, now
	return nil
}

func (r *DictRepository) UpdateItem(it *domaindict.Item) error {
	return r.db.Model(&model.DictItem{}).Where("id = ?", it.ID).Updates(map[string]any{
		"label": it.Label, "sort": it.Sort, "enabled": it.Enabled, "remark": it.Remark,
		"update_time": time.Now().Format("2006-01-02 15:04:05"),
	}).Error
}

func (r *DictRepository) DeleteItem(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.DictItem{}).Error
}
