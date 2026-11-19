package cascadeapp

import (
	"fmt"
	"strings"

	domainchannel "zero-web-server/internal/domain/channel"
	domainplatform "zero-web-server/internal/domain/platform"
	"zero-web-server/internal/gbsip"
)

type platformChannelRepo interface {
	GetByCustomDeviceID(platformID int, customID string) (*domainplatform.Channel, error)
	ListPlatformChannels(platformID int) ([]int, error)
	GetPlatformChannel(platformID, channelID int) (*domainplatform.Channel, error)
}

type channelRepo interface {
	GetByID(id int) (*domainchannel.Channel, error)
	GetOne(deviceID, channelDeviceID string) (*domainchannel.Channel, error)
}

// ResolvedChannel maps an upstream catalog ID to a local leaf channel.
type ResolvedChannel struct {
	PlatformID  int
	DeviceID    string
	ChannelGBID string
	ChannelID   int
}

type Resolver struct {
	platforms  domainplatform.Repository
	platformCh platformChannelRepo
	channels   channelRepo
}

func NewResolver(platforms domainplatform.Repository, platformCh platformChannelRepo, channels channelRepo) *Resolver {
	return &Resolver{platforms: platforms, platformCh: platformCh, channels: channels}
}

// ListSharedCatalogItems 返回共享给该上级平台的目录项（含自定义编号/名称）。
func (r *Resolver) ListSharedCatalogItems(platformID int) ([]gbsip.CatalogItem, error) {
	ids, err := r.platformCh.ListPlatformChannels(platformID)
	if err != nil {
		return nil, err
	}
	items := make([]gbsip.CatalogItem, 0, len(ids))
	for _, id := range ids {
		ch, err := r.channels.GetByID(id)
		if err != nil || ch == nil {
			continue
		}
		status := ch.Status
		if status == "" {
			status = "ON"
		}
		item := gbsip.CatalogItem{
			DeviceID:     ch.GBDeviceID,
			Name:         ch.Name,
			Manufacturer: ch.Manufacturer,
			Model:        ch.Model,
			Parental:     ch.Parental,
			ParentID:     ch.ParentID,
			Status:       status,
			Longitude:    ch.Longitude,
			Latitude:     ch.Latitude,
		}
		if row, err := r.platformCh.GetPlatformChannel(platformID, id); err == nil && row != nil {
			if row.CustomDeviceID != "" {
				item.DeviceID = row.CustomDeviceID
			}
			if row.CustomName != "" {
				item.Name = row.CustomName
			}
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Resolver) Resolve(upstreamGBID, catalogChannelGBID string) (*ResolvedChannel, error) {
	upstreamGBID = strings.TrimSpace(upstreamGBID)
	catalogChannelGBID = strings.TrimSpace(catalogChannelGBID)
	if upstreamGBID == "" || catalogChannelGBID == "" {
		return nil, fmt.Errorf("empty cascade id")
	}
	platform, err := r.platforms.GetByServerGBID(upstreamGBID)
	if err != nil || platform == nil {
		return nil, fmt.Errorf("未知上级平台: %s", upstreamGBID)
	}
	if row, err := r.platformCh.GetByCustomDeviceID(platform.ID, catalogChannelGBID); err == nil && row != nil {
		ch, err := r.channels.GetByID(row.DeviceChannelID)
		if err != nil {
			return nil, err
		}
		return &ResolvedChannel{
			PlatformID: platform.ID, DeviceID: ch.DeviceID,
			ChannelGBID: ch.GBDeviceID, ChannelID: ch.ID,
		}, nil
	}
	// Fallback: catalog ID == local GB device id among shared channels.
	ids, err := r.platformCh.ListPlatformChannels(platform.ID)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		ch, err := r.channels.GetByID(id)
		if err != nil {
			continue
		}
		if ch.GBDeviceID == catalogChannelGBID {
			return &ResolvedChannel{
				PlatformID: platform.ID, DeviceID: ch.DeviceID,
				ChannelGBID: ch.GBDeviceID, ChannelID: ch.ID,
			}, nil
		}
		if row, err := r.platformCh.GetPlatformChannel(platform.ID, id); err == nil && row != nil {
			if row.CustomDeviceID == catalogChannelGBID {
				return &ResolvedChannel{
					PlatformID: platform.ID, DeviceID: ch.DeviceID,
					ChannelGBID: ch.GBDeviceID, ChannelID: ch.ID,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("未共享通道: %s", catalogChannelGBID)
}
