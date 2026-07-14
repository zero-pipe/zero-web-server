package persistence

import (
	"errors"
	"time"

	"zero-web-kit/internal/infrastructure/persistence/model"
	"zero-web-kit/internal/port"

	"gorm.io/gorm"
)

const objectStoreSingletonID = 1

type ObjectStoreConfigRepository struct {
	db *gorm.DB
}

func NewObjectStoreConfigRepository(db *gorm.DB) *ObjectStoreConfigRepository {
	return &ObjectStoreConfigRepository{db: db}
}

func (r *ObjectStoreConfigRepository) Get() (*model.ObjectStoreConfig, error) {
	var row model.ObjectStoreConfig
	err := r.db.First(&row, objectStoreSingletonID).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ObjectStoreConfigRepository) Save(row *model.ObjectStoreConfig) error {
	row.ID = objectStoreSingletonID
	row.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	var existing model.ObjectStoreConfig
	err := r.db.First(&existing, objectStoreSingletonID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if row.CreateTime == "" {
			row.CreateTime = row.UpdateTime
		}
		return r.db.Create(row).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&model.ObjectStoreConfig{}).Where("id = ?", objectStoreSingletonID).Updates(map[string]any{
		"enabled":     row.Enabled,
		"provider":    row.Provider,
		"endpoint":    row.Endpoint,
		"region":      row.Region,
		"bucket":      row.Bucket,
		"access_key":  row.AccessKey,
		"secret_key":  row.SecretKey,
		"use_ssl":     row.UseSSL,
		"path_style":  row.PathStyle,
		"public_base": row.PublicBase,
		"update_time": row.UpdateTime,
	}).Error
}

func (r *ObjectStoreConfigRepository) ToPortConfig(row *model.ObjectStoreConfig) port.ObjectStoreConfig {
	if row == nil {
		return port.ObjectStoreConfig{Provider: "noop"}
	}
	return port.ObjectStoreConfig{
		Enabled:    row.Enabled,
		Provider:   row.Provider,
		Endpoint:   row.Endpoint,
		Region:     row.Region,
		Bucket:     row.Bucket,
		AccessKey:  row.AccessKey,
		SecretKey:  row.SecretKey,
		UseSSL:     row.UseSSL,
		PathStyle:  row.PathStyle,
		PublicBase: row.PublicBase,
	}
}

func DefaultObjectStoreConfig() *model.ObjectStoreConfig {
	now := time.Now().Format("2006-01-02 15:04:05")
	return &model.ObjectStoreConfig{
		ID: 1, Enabled: false, Provider: "noop", PathStyle: true,
		CreateTime: now, UpdateTime: now,
	}
}
