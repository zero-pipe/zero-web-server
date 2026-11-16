package persistence

import (
	"errors"
	"time"

	domainobjectstore "zero-web-server/internal/domain/objectstore"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

const objectStoreSingletonID = 1

type ObjectStoreConfigRepository struct {
	db *gorm.DB
}

func NewObjectStoreConfigRepository(db *gorm.DB) *ObjectStoreConfigRepository {
	return &ObjectStoreConfigRepository{db: db}
}

func (r *ObjectStoreConfigRepository) Get() (*domainobjectstore.Config, error) {
	var row model.ObjectStoreConfig
	if err := r.db.First(&row, objectStoreSingletonID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainobjectstore.ErrNotFound
		}
		return nil, err
	}
	return toDomainObjectStoreConfig(&row), nil
}

func (r *ObjectStoreConfigRepository) Save(d *domainobjectstore.Config) error {
	d.ID = objectStoreSingletonID
	d.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	var existing model.ObjectStoreConfig
	err := r.db.First(&existing, objectStoreSingletonID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if d.CreateTime == "" {
			d.CreateTime = d.UpdateTime
		}
		return r.db.Create(toModelObjectStoreConfig(d)).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&model.ObjectStoreConfig{}).Where("id = ?", objectStoreSingletonID).Updates(map[string]any{
		"enabled":     d.Enabled,
		"provider":    d.Provider,
		"endpoint":    d.Endpoint,
		"region":      d.Region,
		"bucket":      d.Bucket,
		"access_key":  d.AccessKey,
		"secret_key":  d.SecretKey,
		"use_ssl":     d.UseSSL,
		"path_style":  d.PathStyle,
		"public_base": d.PublicBase,
		"update_time": d.UpdateTime,
	}).Error
}

func toModelObjectStoreConfig(d *domainobjectstore.Config) *model.ObjectStoreConfig {
	return &model.ObjectStoreConfig{
		ID: d.ID, Enabled: d.Enabled, Provider: d.Provider, Endpoint: d.Endpoint,
		Region: d.Region, Bucket: d.Bucket, AccessKey: d.AccessKey, SecretKey: d.SecretKey,
		UseSSL: d.UseSSL, PathStyle: d.PathStyle, PublicBase: d.PublicBase,
		CreateTime: d.CreateTime, UpdateTime: d.UpdateTime,
	}
}

func toDomainObjectStoreConfig(m *model.ObjectStoreConfig) *domainobjectstore.Config {
	return &domainobjectstore.Config{
		ID: m.ID, Enabled: m.Enabled, Provider: m.Provider, Endpoint: m.Endpoint,
		Region: m.Region, Bucket: m.Bucket, AccessKey: m.AccessKey, SecretKey: m.SecretKey,
		UseSSL: m.UseSSL, PathStyle: m.PathStyle, PublicBase: m.PublicBase,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

var _ domainobjectstore.Repository = (*ObjectStoreConfigRepository)(nil)
