package persistence

import (
	"zero-web-kit/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type MediaServerRepository struct{ db *gorm.DB }

func NewMediaServerRepository(db *gorm.DB) *MediaServerRepository {
	return &MediaServerRepository{db: db}
}

func (r *MediaServerRepository) ListAll() ([]model.MediaServer, error) {
	var rows []model.MediaServer
	err := r.db.Find(&rows).Error
	return rows, err
}

func (r *MediaServerRepository) GetByID(id string) (*model.MediaServer, error) {
	var m model.MediaServer
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MediaServerRepository) Save(m *model.MediaServer) error {
	var existing model.MediaServer
	if err := r.db.Where("id = ?", m.ID).First(&existing).Error; err == gorm.ErrRecordNotFound {
		return r.db.Create(m).Error
	}
	return r.db.Save(m).Error
}

func (r *MediaServerRepository) Delete(id string) error {
	return r.db.Delete(&model.MediaServer{}, "id = ?", id).Error
}

func (r *MediaServerRepository) ClearDefaultExcept(id string) error {
	return r.db.Model(&model.MediaServer{}).Where("id <> ?", id).
		Update("default_server", false).Error
}

func (r *MediaServerRepository) ClearAllDefault() error {
	return r.db.Model(&model.MediaServer{}).Where("default_server = ?", true).
		Update("default_server", false).Error
}

func (r *MediaServerRepository) GetDefault() (*model.MediaServer, error) {
	var m model.MediaServer
	err := r.db.Where("default_server = ?", true).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		err = r.db.First(&m).Error
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
