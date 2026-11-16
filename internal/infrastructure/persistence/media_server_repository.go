package persistence

import (
	domainmediaserver "zero-web-server/internal/domain/mediaserver"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type MediaServerRepository struct{ db *gorm.DB }

func NewMediaServerRepository(db *gorm.DB) *MediaServerRepository {
	return &MediaServerRepository{db: db}
}

func (r *MediaServerRepository) ListAll() ([]domainmediaserver.MediaServer, error) {
	var rows []model.MediaServer
	if err := r.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domainmediaserver.MediaServer, 0, len(rows))
	for i := range rows {
		out = append(out, *toDomainMediaServer(&rows[i]))
	}
	return out, nil
}

func (r *MediaServerRepository) GetByID(id string) (*domainmediaserver.MediaServer, error) {
	var m model.MediaServer
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return toDomainMediaServer(&m), nil
}

func (r *MediaServerRepository) Save(d *domainmediaserver.MediaServer) error {
	m := toModelMediaServer(d)
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

func toModelMediaServer(d *domainmediaserver.MediaServer) *model.MediaServer {
	return &model.MediaServer{
		ID: d.ID, IP: d.IP, HookIP: d.HookIP, SDPIP: d.SDPIP, StreamIP: d.StreamIP,
		HTTPPort: d.HTTPPort, HTTPSSLPort: d.HTTPSSLPort, RTMPPort: d.RTMPPort,
		RTSPPort: d.RTSPPort, RTPProxyPort: d.RTPProxyPort, Secret: d.Secret,
		Type: d.Type, DefaultServer: d.DefaultServer, HookAliveInterval: d.HookAliveInterval,
		CreateTime: d.CreateTime, UpdateTime: d.UpdateTime, ServerID: d.ServerID,
	}
}

func toDomainMediaServer(m *model.MediaServer) *domainmediaserver.MediaServer {
	return &domainmediaserver.MediaServer{
		ID: m.ID, IP: m.IP, HookIP: m.HookIP, SDPIP: m.SDPIP, StreamIP: m.StreamIP,
		HTTPPort: m.HTTPPort, HTTPSSLPort: m.HTTPSSLPort, RTMPPort: m.RTMPPort,
		RTSPPort: m.RTSPPort, RTPProxyPort: m.RTPProxyPort, Secret: m.Secret,
		Type: m.Type, DefaultServer: m.DefaultServer, HookAliveInterval: m.HookAliveInterval,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime, ServerID: m.ServerID,
	}
}

var _ domainmediaserver.Repository = (*MediaServerRepository)(nil)
