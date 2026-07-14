package ops

import (
	"context"

	"zero-web-server/internal/infrastructure/persistence/model"
	"zero-web-server/internal/port"

	"gorm.io/gorm"
)

type ResourceBase struct {
	Total  int64 `json:"total"`
	Online int64 `json:"online"`
}

type ResourceInfo struct {
	Device  ResourceBase `json:"device"`
	Channel ResourceBase `json:"channel"`
	Push    ResourceBase `json:"push"`
	Proxy   ResourceBase `json:"proxy"`
}

type MediaServerLoad struct {
	ID        string `json:"id"`
	Push      int64  `json:"push"`
	Proxy     int64  `json:"proxy"`
	GbReceive int64  `json:"gbReceive"`
	GbSend    int64  `json:"gbSend"`
}

type Dashboard struct {
	db    *gorm.DB
	media port.MediaCluster
}

func NewDashboard(db *gorm.DB, media port.MediaCluster) *Dashboard {
	return &Dashboard{db: db, media: media}
}

func (d *Dashboard) ResourceInfo() ResourceInfo {
	out := ResourceInfo{}
	if d.db == nil {
		return out
	}
	_ = d.db.Model(&model.GBDevice{}).Count(&out.Device.Total).Error
	_ = d.db.Model(&model.GBDevice{}).Where("on_line = ?", true).Count(&out.Device.Online).Error

	_ = d.db.Model(&model.GBDeviceChannel{}).Count(&out.Channel.Total).Error
	_ = d.db.Model(&model.GBDeviceChannel{}).
		Where("UPPER(COALESCE(NULLIF(TRIM(gb_status), ''), NULLIF(TRIM(status), ''), 'OFF')) = ?", "ON").
		Count(&out.Channel.Online).Error

	_ = d.db.Model(&model.StreamPush{}).Count(&out.Push.Total).Error
	_ = d.db.Model(&model.StreamPush{}).Where("pushing = ?", true).Count(&out.Push.Online).Error

	_ = d.db.Model(&model.StreamProxy{}).Count(&out.Proxy.Total).Error
	_ = d.db.Model(&model.StreamProxy{}).Where("pulling = ?", true).Count(&out.Proxy.Online).Error
	return out
}

func (d *Dashboard) MediaLoads() ([]MediaServerLoad, error) {
	if d.media == nil {
		return []MediaServerLoad{}, nil
	}
	nodes, err := d.media.List(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]MediaServerLoad, 0, len(nodes))
	for _, n := range nodes {
		if !n.Online {
			continue
		}
		item := MediaServerLoad{
			ID:        n.ID,
			GbReceive: n.Load,
			GbSend:    0,
		}
		if d.db != nil {
			_ = d.db.Model(&model.StreamPush{}).
				Where("pushing = ? AND media_server_id = ?", true, n.ID).
				Count(&item.Push).Error
			_ = d.db.Model(&model.StreamProxy{}).
				Where("pulling = ? AND media_server_id = ?", true, n.ID).
				Count(&item.Proxy).Error
		}
		out = append(out, item)
	}
	return out, nil
}
