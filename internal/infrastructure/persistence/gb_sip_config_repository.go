package persistence

import (
	"errors"
	"strings"
	"time"

	domaingbsipconfig "zero-web-server/internal/domain/gbsipconfig"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

const gbSipConfigSingletonID = 1

type GbSipConfigRepository struct {
	db *gorm.DB
}

func NewGbSipConfigRepository(db *gorm.DB) *GbSipConfigRepository {
	return &GbSipConfigRepository{db: db}
}

func (r *GbSipConfigRepository) Get() (*domaingbsipconfig.Config, error) {
	var row model.GbSipConfig
	if err := r.db.First(&row, gbSipConfigSingletonID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaingbsipconfig.ErrNotFound
		}
		return nil, err
	}
	return toDomainGbSipConfig(&row), nil
}

func (r *GbSipConfigRepository) Save(d *domaingbsipconfig.Config) error {
	d.ID = gbSipConfigSingletonID
	d.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	var existing model.GbSipConfig
	err := r.db.First(&existing, gbSipConfigSingletonID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if d.CreateTime == "" {
			d.CreateTime = d.UpdateTime
		}
		return r.db.Create(toModelGbSipConfig(d)).Error
	}
	if err != nil {
		return err
	}
	transport := strings.TrimSpace(d.Transport)
	if transport == "" {
		transport = "UDP"
	}
	return r.db.Model(&model.GbSipConfig{}).Where("id = ?", gbSipConfigSingletonID).Updates(map[string]any{
		"ip":                   d.IP,
		"port":                 d.Port,
		"domain":               d.Domain,
		"device_id":            d.DeviceID,
		"password":             d.Password,
		"alarm":                d.Alarm,
		"require_pre_register": d.RequirePreRegister,
		"transport":            transport,
		"update_time":          d.UpdateTime,
	}).Error
}

func toModelGbSipConfig(d *domaingbsipconfig.Config) *model.GbSipConfig {
	return &model.GbSipConfig{
		ID: d.ID, IP: d.IP, Port: d.Port, Domain: d.Domain, DeviceID: d.DeviceID,
		Password: d.Password, Alarm: d.Alarm, RequirePreRegister: d.RequirePreRegister,
		Transport: d.Transport, CreateTime: d.CreateTime, UpdateTime: d.UpdateTime,
	}
}

func toDomainGbSipConfig(m *model.GbSipConfig) *domaingbsipconfig.Config {
	return &domaingbsipconfig.Config{
		ID: m.ID, IP: m.IP, Port: m.Port, Domain: m.Domain, DeviceID: m.DeviceID,
		Password: m.Password, Alarm: m.Alarm, RequirePreRegister: m.RequirePreRegister,
		Transport: m.Transport, CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

var _ domaingbsipconfig.Repository = (*GbSipConfigRepository)(nil)
