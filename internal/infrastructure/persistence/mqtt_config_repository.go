package persistence

import (
	"errors"
	"time"

	domaininfra "zero-web-server/internal/domain/infraconfig"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type MQTTConfigRepository struct{ db *gorm.DB }

func NewMQTTConfigRepository(db *gorm.DB) *MQTTConfigRepository {
	return &MQTTConfigRepository{db: db}
}

var _ domaininfra.Repository = (*MQTTConfigRepository)(nil)

func (r *MQTTConfigRepository) Get() (*domaininfra.MQTTConfig, error) {
	var m model.MQTTConfig
	if err := r.db.Where("id = ?", 1).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domaininfra.ErrNotFound
		}
		return nil, err
	}
	return &domaininfra.MQTTConfig{
		ID: m.ID, Enabled: m.Enabled, Broker: m.Broker, ClientIDPrefix: m.ClientIDPrefix,
		Username: m.Username, Password: m.Password, TopicPrefix: m.TopicPrefix,
		QoS: m.QoS, KeepAliveSec: m.KeepAliveSec, TLSEnabled: m.TLSEnabled,
		CACert: m.CACert, ClientCert: m.ClientCert, ClientKey: m.ClientKey,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}, nil
}

func (r *MQTTConfigRepository) Save(row *domaininfra.MQTTConfig) error {
	if row == nil {
		return errors.New("mqtt config is nil")
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	row.ID = 1
	var existing model.MQTTConfig
	err := r.db.Where("id = ?", 1).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(&model.MQTTConfig{
			ID: 1, Enabled: row.Enabled, Broker: row.Broker, ClientIDPrefix: row.ClientIDPrefix,
			Username: row.Username, Password: row.Password, TopicPrefix: row.TopicPrefix,
			QoS: row.QoS, KeepAliveSec: row.KeepAliveSec, TLSEnabled: row.TLSEnabled,
			CACert: row.CACert, ClientCert: row.ClientCert, ClientKey: row.ClientKey,
			CreateTime: now, UpdateTime: now,
		}).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&model.MQTTConfig{}).Where("id = ?", 1).Updates(map[string]any{
		"enabled": row.Enabled, "broker": row.Broker, "client_id_prefix": row.ClientIDPrefix,
		"username": row.Username, "password": row.Password, "topic_prefix": row.TopicPrefix,
		"qos": row.QoS, "keepalive_sec": row.KeepAliveSec, "tls_enabled": row.TLSEnabled,
		"ca_cert": row.CACert, "client_cert": row.ClientCert, "client_key": row.ClientKey,
		"update_time": now,
	}).Error
}
