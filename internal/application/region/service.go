package regionapp

import (
	"errors"
	"fmt"
	"strings"
	"zero-web-server/internal/apperr"

	"zero-web-server/internal/civilcode"
	domaingroupregion "zero-web-server/internal/domain/groupregion"
	domaintree "zero-web-server/internal/domain/tree"
)

var (
	ErrRegionNameRequired     = apperr.New(apperr.CodeGroup+20, "名称必须存在")
	ErrRegionDeviceIDRequired = apperr.New(apperr.CodeGroup+21, "国标编号必须存在")
	ErrRegionDuplicate        = apperr.New(apperr.CodeGroup+22, "此行政区划已存在")
)

type Service struct {
	repo domaingroupregion.Repository
}

func NewService(repo domaingroupregion.Repository) *Service {
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

// AddByCivilCode 按行政区划编码一键建树：自动补齐父级链路，已存在的跳过。
func (s *Service) AddByCivilCode(code string) ([]domaintree.Node, error) {
	chain := civilcode.Ancestors(code)
	if len(chain) == 0 {
		return nil, fmt.Errorf("未找到行政区划编码: %s", strings.TrimSpace(code))
	}
	created := make([]domaintree.Node, 0, len(chain))
	var parentDeviceID string
	var parentID *int
	for _, item := range chain {
		if node, err := s.repo.GetRegionByDeviceID(item.DeviceID); err == nil && node != nil {
			id := node.ID
			parentID = &id
			parentDeviceID = node.DeviceID
			continue
		}
		rec := &domaingroupregion.RegionRecord{
			DeviceID:       item.DeviceID,
			Name:           item.Name,
			ParentID:       parentID,
			ParentDeviceID: parentDeviceID,
		}
		if err := s.repo.AddRegion(rec); err != nil {
			return created, err
		}
		node, err := s.repo.GetRegionByDeviceID(item.DeviceID)
		if err != nil {
			return created, err
		}
		created = append(created, *node)
		id := node.ID
		parentID = &id
		parentDeviceID = node.DeviceID
	}
	return created, nil
}

// ListCodeMappings 返回区划编码映射。
func (s *Service) ListCodeMappings() ([]*domaingroupregion.RegionCodeMapping, error) {
	return s.repo.ListRegionCodeMappings()
}

// SaveCodeMapping 设置区划节点对外国标码（必须 20 位数字）。
func (s *Service) SaveCodeMapping(regionDeviceID, gbCode string) error {
	regionDeviceID = strings.TrimSpace(regionDeviceID)
	gbCode = strings.TrimSpace(gbCode)
	if regionDeviceID == "" {
		return fmt.Errorf("区划编号不能为空")
	}
	if len(gbCode) != 20 || !isDigits(gbCode) {
		return fmt.Errorf("对外国标码必须为 20 位数字")
	}
	if _, err := s.repo.GetRegionByDeviceID(regionDeviceID); err != nil {
		return fmt.Errorf("行政区划不存在: %s", regionDeviceID)
	}
	return s.repo.UpsertRegionCodeMapping(&domaingroupregion.RegionCodeMapping{
		RegionDeviceID: regionDeviceID, GBCode: gbCode,
	})
}

// DeleteCodeMapping 删除映射，恢复默认编号推导。
func (s *Service) DeleteCodeMapping(regionDeviceID string) error {
	return s.repo.DeleteRegionCodeMapping(strings.TrimSpace(regionDeviceID))
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (s *Service) GetAllChild(parent string) []civilcode.RegionItem {
	return civilcode.GetAllChild(parent)
}

func (s *Service) GetDescription(code string) string {
	return civilcode.GetDescription(code)
}

func (s *Service) Add(region *domaingroupregion.RegionRecord) error {
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

func (s *Service) Update(region *domaingroupregion.RegionRecord) error {
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
