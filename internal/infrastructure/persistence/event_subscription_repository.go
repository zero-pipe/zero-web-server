package persistence

import (
	"encoding/json"
	"errors"
	"time"

	domainevent "zero-web-server/internal/domain/eventsub"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type EventSubscriptionRepository struct{ db *gorm.DB }

func NewEventSubscriptionRepository(db *gorm.DB) *EventSubscriptionRepository {
	return &EventSubscriptionRepository{db: db}
}

var _ domainevent.Repository = (*EventSubscriptionRepository)(nil)

func (r *EventSubscriptionRepository) Save(s *domainevent.Subscription) error {
	if s == nil {
		return errors.New("subscription is nil")
	}
	raw, _ := json.Marshal(s.EventTypes)
	now := time.Now().Format("2006-01-02 15:04:05")
	var existing model.EventSubscription
	err := r.db.Where("device_id = ?", s.DeviceID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		m := &model.EventSubscription{
			DeviceID: s.DeviceID, PluginID: s.PluginID, DeviceUID: s.DeviceUID,
			DeviceInternalCode: s.DeviceInternalCode, EventTypesJSON: string(raw),
			Armed: s.Armed, Status: s.Status, LastError: s.LastError, RemoteID: s.RemoteID,
			LastArmedAt: s.LastArmedAt, CreateTime: now, UpdateTime: now,
		}
		return r.db.Create(m).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&model.EventSubscription{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"plugin_instance_id": s.PluginID, "device_uid": s.DeviceUID,
		"device_internal_code": s.DeviceInternalCode, "event_types": string(raw),
		"armed": s.Armed, "status": s.Status, "last_error": s.LastError,
		"remote_id": s.RemoteID, "last_armed_at": s.LastArmedAt, "update_time": now,
	}).Error
}

func (r *EventSubscriptionRepository) GetByDevice(deviceID int) (*domainevent.Subscription, error) {
	var m model.EventSubscription
	if err := r.db.Where("device_id = ?", deviceID).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainEventSubscription(&m), nil
}

func (r *EventSubscriptionRepository) ListArmedByPlugin(pluginID string) ([]*domainevent.Subscription, error) {
	var rows []model.EventSubscription
	if err := r.db.Where("plugin_instance_id = ? AND armed = ?", pluginID, true).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainevent.Subscription, 0, len(rows))
	for i := range rows {
		out = append(out, toDomainEventSubscription(&rows[i]))
	}
	return out, nil
}

func (r *EventSubscriptionRepository) DeleteByDevice(deviceID int) error {
	return r.db.Where("device_id = ?", deviceID).Delete(&model.EventSubscription{}).Error
}

func toDomainEventSubscription(m *model.EventSubscription) *domainevent.Subscription {
	var types []string
	_ = json.Unmarshal([]byte(m.EventTypesJSON), &types)
	return &domainevent.Subscription{
		ID: m.ID, DeviceID: m.DeviceID, PluginID: m.PluginID, DeviceUID: m.DeviceUID,
		DeviceInternalCode: m.DeviceInternalCode, EventTypes: types,
		Armed: m.Armed, Status: m.Status, LastError: m.LastError, RemoteID: m.RemoteID,
		LastArmedAt: m.LastArmedAt, CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}
