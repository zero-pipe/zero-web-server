package persistence

import (
	"errors"
	"time"

	"zero-web-kit/internal/infrastructure/config"
	"zero-web-kit/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

const gbSipConfigSingletonID = 1

type GbSipConfigRepository struct {
	db *gorm.DB
}

func NewGbSipConfigRepository(db *gorm.DB) *GbSipConfigRepository {
	return &GbSipConfigRepository{db: db}
}

func (r *GbSipConfigRepository) Get() (*model.GbSipConfig, error) {
	var row model.GbSipConfig
	err := r.db.First(&row, gbSipConfigSingletonID).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *GbSipConfigRepository) Save(row *model.GbSipConfig) error {
	row.ID = gbSipConfigSingletonID
	row.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	var existing model.GbSipConfig
	err := r.db.First(&existing, gbSipConfigSingletonID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if row.CreateTime == "" {
			row.CreateTime = row.UpdateTime
		}
		return r.db.Create(row).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&model.GbSipConfig{}).Where("id = ?", gbSipConfigSingletonID).Updates(map[string]any{
		"ip":          row.IP,
		"port":        row.Port,
		"domain":      row.Domain,
		"device_id":   row.DeviceID,
		"password":    row.Password,
		"alarm":       row.Alarm,
		"update_time": row.UpdateTime,
	}).Error
}

func (r *GbSipConfigRepository) ToSIPConfig(row *model.GbSipConfig) config.SIPConfig {
	if row == nil {
		return config.SIPConfig{}
	}
	return config.SIPConfig{
		IP:       row.IP,
		Port:     row.Port,
		Domain:   row.Domain,
		ID:       row.DeviceID,
		Password: row.Password,
		Alarm:    row.Alarm,
	}
}
