package upstreamapp

import (
	"fmt"

	domainchannel "zero-web-server/internal/domain/channel"
	domainplatform "zero-web-server/internal/domain/platform"
	"zero-web-server/internal/port"
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
	GetPlatformChannel(platformID, channelID int) (*domainplatform.Channel, error)
	GetByCustomDeviceID(platformID int, customID string) (*domainplatform.Channel, error)
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
	nodes      catalogNodeRepo
	sip        port.PlatformClient
}

func NewChannelService(
	platforms domainplatform.Repository,
	channels channelRepo,
	platformCh PlatformChannelRepository,
	nodes catalogNodeRepo,
	sipClient port.PlatformClient,
) *ChannelService {
	return &ChannelService{
		platforms: platforms, channels: channels, platformCh: platformCh,
		nodes: nodes, sip: sipClient,
	}
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
	return s.pushCatalog(platformID)
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

// ListGroupShares 返回该平台已共享的业务目录节点。
func (s *ChannelService) ListGroupShares(platformID int) ([]string, error) {
	return s.platforms.ListGroupShares(platformID)
}

// AddGroupShares 按业务目录节点（215/216）共享，节点下通道动态跟随。
func (s *ChannelService) AddGroupShares(platformID int, groupDeviceIDs []string) error {
	if _, err := s.platforms.GetByID(platformID); err != nil {
		return fmt.Errorf("平台不存在")
	}
	return s.platforms.AddGroupShares(platformID, groupDeviceIDs)
}

// RemoveGroupShares 取消按业务目录节点的共享。
func (s *ChannelService) RemoveGroupShares(platformID int, groupDeviceIDs []string) error {
	if _, err := s.platforms.GetByID(platformID); err != nil {
		return fmt.Errorf("平台不存在")
	}
	return s.platforms.RemoveGroupShares(platformID, groupDeviceIDs)
}
