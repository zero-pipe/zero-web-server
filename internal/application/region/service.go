package regionapp

import (
	"errors"
	"strings"

	domaintree "zero-web-server/internal/domain/tree"
	"zero-web-server/internal/infrastructure/civilcode"
	"zero-web-server/internal/infrastructure/persistence"
)

var (
	ErrRegionNameRequired     = errors.New("名称必须存在")
	ErrRegionDeviceIDRequired = errors.New("国标编号必须存在")
	ErrRegionDuplicate        = errors.New("此行政区划已存在")
)

type Service struct {
	repo *persistence.GroupRegionRepository
}

func NewService(repo *persistence.GroupRegionRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) QueryForTree(parentID *int, hasChannel *bool) ([]domaintree.Node, error) {
	nodes, err := s.repo.QueryRegionTree(parentID)
	if err != nil {
		return nil, err
	}
	if parentID == nil || hasChannel == nil || !*hasChannel {
		return nodes, nil
	}
	parent, err := s.repo.GetRegionByID(*parentID)
	if err != nil {
		return nodes, nil
	}
	channels, err := s.repo.QueryRegionChannels(parent.DeviceID)
	if err != nil {
		return nodes, nil
	}
	return append(nodes, channels...), nil
}

func (s *Service) GetAllChild(parent string) []civilcode.RegionItem {
	return civilcode.GetAllChild(parent)
}

func (s *Service) GetDescription(code string) string {
	return civilcode.GetDescription(code)
}

func (s *Service) Add(region *persistence.RegionRecord) error {
	if strings.TrimSpace(region.Name) == "" {
		return ErrRegionNameRequired
	}
	if strings.TrimSpace(region.DeviceID) == "" {
		return ErrRegionDeviceIDRequired
	}
	if strings.TrimSpace(region.ParentDeviceID) == "" {
		region.ParentDeviceID = ""
	}
	if err := s.repo.AddRegion(region); err != nil {
		if isDuplicateKey(err) {
			return ErrRegionDuplicate
		}
		return err
	}
	return nil
}

func (s *Service) Update(region *persistence.RegionRecord) error {
	if region.ID <= 0 {
		return errors.New("无效的区划ID")
	}
	if strings.TrimSpace(region.Name) == "" {
		return ErrRegionNameRequired
	}
	if strings.TrimSpace(region.DeviceID) == "" {
		return ErrRegionDeviceIDRequired
	}
	if err := s.repo.UpdateRegion(region); err != nil {
		if isDuplicateKey(err) {
			return ErrRegionDuplicate
		}
		return err
	}
	return nil
}

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "1062")
}
