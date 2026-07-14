package platformapp

import (
	"fmt"

	domainchannel "zero-web-kit/internal/domain/channel"
	domainplatform "zero-web-kit/internal/domain/platform"
	"zero-web-kit/internal/infrastructure/persistence/model"
	sipinfra "zero-web-kit/internal/infrastructure/sip"
)

type channelRepo interface {
	ListAll(page, count int, query string) ([]*domainchannel.Channel, int64, error)
	GetByID(id int) (*domainchannel.Channel, error)
}

type PlatformChannelRepository interface {
	ListPlatformChannels(platformID int) ([]int, error)
	AddPlatformChannel(platformID, channelID int) error
	RemovePlatformChannel(platformID, channelID int) error
	ListChannelIDsByDevice(deviceID string) ([]int, error)
	GetPlatformChannel(platformID, channelID int) (*model.PlatformChannel, error)
	GetByCustomDeviceID(platformID int, customID string) (*model.PlatformChannel, error)
	UpdatePlatformChannelCustom(platformID, channelID int, customID, customName string) error
}

type ChannelView struct {
	domainchannel.Channel
	HasShare       bool   `json:"hasShare"`
	CustomDeviceID string `json:"customDeviceId"`
	CustomName     string `json:"customName"`
}

type ChannelService struct {
	platforms  domainplatform.Repository
	channels   channelRepo
	platformCh PlatformChannelRepository
	sip        *sipinfra.PlatformClient
}

func NewChannelService(
	platforms domainplatform.Repository,
	channels channelRepo,
	platformCh PlatformChannelRepository,
	sipClient *sipinfra.PlatformClient,
) *ChannelService {
	return &ChannelService{platforms: platforms, channels: channels, platformCh: platformCh, sip: sipClient}
}

func (s *ChannelService) List(platformID, page, count int, query string, hasShare *bool) ([]ChannelView, int64, error) {
	shared, _ := s.platformCh.ListPlatformChannels(platformID)
	sharedSet := make(map[int]struct{}, len(shared))
	for _, id := range shared {
		sharedSet[id] = struct{}{}
	}

	all, total, err := s.channels.ListAll(page, count, query)
	if err != nil {
		return nil, 0, err
	}
	list := make([]ChannelView, 0, len(all))
	for _, ch := range all {
		_, ok := sharedSet[ch.ID]
		view := ChannelView{Channel: *ch, HasShare: ok}
		if ok {
			if row, err := s.platformCh.GetPlatformChannel(platformID, ch.ID); err == nil && row != nil {
				view.CustomDeviceID = row.CustomDeviceID
				view.CustomName = row.CustomName
			}
		}
		if hasShare != nil && view.HasShare != *hasShare {
			continue
		}
		list = append(list, view)
	}
	return list, total, nil
}

func (s *ChannelService) AddChannels(platformID int, channelIDs []int, all bool) error {
	if _, err := s.platforms.GetByID(platformID); err != nil {
		return fmt.Errorf("平台不存在")
	}
	if all {
		allChannels, _, err := s.channels.ListAll(1, 100000, "")
		if err != nil {
			return err
		}
		shared, _ := s.platformCh.ListPlatformChannels(platformID)
		sharedSet := make(map[int]struct{}, len(shared))
		for _, id := range shared {
			sharedSet[id] = struct{}{}
		}
		for _, ch := range allChannels {
			if _, ok := sharedSet[ch.ID]; ok {
				continue
			}
			if err := s.platformCh.AddPlatformChannel(platformID, ch.ID); err != nil {
				return err
			}
		}
		s.maybeAutoPush(platformID)
		return nil
	}
	for _, id := range channelIDs {
		if err := s.platformCh.AddPlatformChannel(platformID, id); err != nil {
			return err
		}
	}
	s.maybeAutoPush(platformID)
	return nil
}

func (s *ChannelService) RemoveChannels(platformID int, channelIDs []int, all bool) error {
	if all {
		ids, err := s.platformCh.ListPlatformChannels(platformID)
		if err != nil {
			return err
		}
		channelIDs = ids
	}
	for _, id := range channelIDs {
		_ = s.platformCh.RemovePlatformChannel(platformID, id)
	}
	return nil
}

func (s *ChannelService) AddChannelsByDevice(platformID int, deviceIDs []string) error {
	for _, deviceID := range deviceIDs {
		ids, err := s.platformCh.ListChannelIDsByDevice(deviceID)
		if err != nil {
			return err
		}
		for _, id := range ids {
			_ = s.platformCh.AddPlatformChannel(platformID, id)
		}
	}
	s.maybeAutoPush(platformID)
	return nil
}

func (s *ChannelService) RemoveChannelsByDevice(platformID int, deviceIDs []string) error {
	for _, deviceID := range deviceIDs {
		ids, err := s.platformCh.ListChannelIDsByDevice(deviceID)
		if err != nil {
			return err
		}
		for _, id := range ids {
			_ = s.platformCh.RemovePlatformChannel(platformID, id)
		}
	}
	return nil
}

func (s *ChannelService) PushCatalog(platformID int) error {
	platform, err := s.platforms.GetByID(platformID)
	if err != nil {
		return fmt.Errorf("平台不存在")
	}
	channelIDs, err := s.platformCh.ListPlatformChannels(platformID)
	if err != nil || len(channelIDs) == 0 {
		return fmt.Errorf("未配置共享通道")
	}
	items := make([]sipinfra.CatalogItem, 0, len(channelIDs))
	for _, id := range channelIDs {
		ch, err := s.channels.GetByID(id)
		if err != nil {
			continue
		}
		status := ch.Status
		if status == "" {
			status = "ON"
		}
		catalogID := ch.GBDeviceID
		catalogName := ch.Name
		if row, err := s.platformCh.GetPlatformChannel(platformID, id); err == nil && row != nil {
			if row.CustomDeviceID != "" {
				catalogID = row.CustomDeviceID
			}
			if row.CustomName != "" {
				catalogName = row.CustomName
			}
		}
		items = append(items, sipinfra.CatalogItem{
			DeviceID: catalogID, Name: catalogName, Status: status,
		})
	}
	return s.sip.SendCatalogNotify(platform, items)
}

func (s *ChannelService) UpdateCustom(platformID, channelID int, customID, customName string) error {
	if _, err := s.platforms.GetByID(platformID); err != nil {
		return fmt.Errorf("平台不存在")
	}
	if _, err := s.platformCh.GetPlatformChannel(platformID, channelID); err != nil {
		return fmt.Errorf("通道未共享到该平台")
	}
	return s.platformCh.UpdatePlatformChannelCustom(platformID, channelID, customID, customName)
}

func (s *ChannelService) maybeAutoPush(platformID int) {
	p, err := s.platforms.GetByID(platformID)
	if err != nil || p == nil || !p.AutoPushChannel || !p.Enable {
		return
	}
	_ = s.PushCatalog(platformID)
}
