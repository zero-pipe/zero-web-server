package persistence

import (
	"time"

	domainaccesstoken "zero-web-server/internal/domain/accesstoken"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type AccessTokenRepository struct {
	db *gorm.DB
}

func NewAccessTokenRepository(db *gorm.DB) *AccessTokenRepository {
	return &AccessTokenRepository{db: db}
}

func (r *AccessTokenRepository) List(page, count int) ([]domainaccesstoken.AccessToken, int64, error) {
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 15
	}
	q := r.db.Model(&model.AccessToken{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.AccessToken
	if err := q.Order("id DESC").Offset((page - 1) * count).Limit(count).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domainaccesstoken.AccessToken, len(rows))
	for i := range rows {
		out[i] = toDomainAccessToken(&rows[i])
	}
	return out, total, nil
}

func (r *AccessTokenRepository) GetByID(id int64) (*domainaccesstoken.AccessToken, error) {
	var m model.AccessToken
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainaccesstoken.ErrNotFound
		}
		return nil, err
	}
	out := toDomainAccessToken(&m)
	return &out, nil
}

func (r *AccessTokenRepository) Create(t *domainaccesstoken.AccessToken) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	m := model.AccessToken{
		Name:       t.Name,
		Token:      t.Token,
		Enabled:    t.Enabled,
		Remark:     t.Remark,
		CreateTime: now,
		UpdateTime: now,
	}
	if err := r.db.Create(&m).Error; err != nil {
		return err
	}
	t.ID = m.ID
	t.CreateTime = now
	t.UpdateTime = now
	return nil
}

func (r *AccessTokenRepository) Update(t *domainaccesstoken.AccessToken) error {
	return r.db.Model(&model.AccessToken{}).Where("id = ?", t.ID).Updates(map[string]any{
		"name":        t.Name,
		"token":       t.Token,
		"enabled":     t.Enabled,
		"remark":      t.Remark,
		"update_time": time.Now().Format("2006-01-02 15:04:05"),
	}).Error
}

func (r *AccessTokenRepository) Delete(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.AccessToken{}).Error
}

func (r *AccessTokenRepository) FindEnabledByToken(token string) (*domainaccesstoken.AccessToken, error) {
	if token == "" {
		return nil, domainaccesstoken.ErrNotFound
	}
	var m model.AccessToken
	if err := r.db.Where("token = ? AND enabled = ?", token, true).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainaccesstoken.ErrNotFound
		}
		return nil, err
	}
	out := toDomainAccessToken(&m)
	return &out, nil
}

func (r *AccessTokenRepository) ListEnabledTokens() ([]string, error) {
	var tokens []string
	if err := r.db.Model(&model.AccessToken{}).
		Where("enabled = ?", true).
		Pluck("token", &tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func toDomainAccessToken(m *model.AccessToken) domainaccesstoken.AccessToken {
	return domainaccesstoken.AccessToken{
		ID:         m.ID,
		Name:       m.Name,
		Token:      m.Token,
		Enabled:    m.Enabled,
		Remark:     m.Remark,
		CreateTime: m.CreateTime,
		UpdateTime: m.UpdateTime,
	}
}
