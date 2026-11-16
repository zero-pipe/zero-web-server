package ops

import (
	"context"

	"zero-web-server/internal/port"
)

// StatsRepository 仪表盘统计所需的只读仓储端口。
type StatsRepository interface {
	DeviceCount() (total, online int64, err error)
	ChannelCount() (total, online int64, err error)
	StreamPushCount(mediaServerID string) (total, online int64, err error)
	StreamProxyCount(mediaServerID string) (total, online int64, err error)
}

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
	db    StatsRepository
	media port.MediaCluster
}

func NewDashboard(db StatsRepository, media port.MediaCluster) *Dashboard {
	return &Dashboard{db: db, media: media}
}

func (d *Dashboard) ResourceInfo() ResourceInfo {
	out := ResourceInfo{}
	if d.db == nil {
		return out
	}
	out.Device.Total, out.Device.Online, _ = d.db.DeviceCount()
	out.Channel.Total, out.Channel.Online, _ = d.db.ChannelCount()
	out.Push.Total, out.Push.Online, _ = d.db.StreamPushCount("")
	out.Proxy.Total, out.Proxy.Online, _ = d.db.StreamProxyCount("")
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
			_, item.Push, _ = d.db.StreamPushCount(n.ID)
			_, item.Proxy, _ = d.db.StreamProxyCount(n.ID)
		}
		out = append(out, item)
	}
	return out, nil
}
