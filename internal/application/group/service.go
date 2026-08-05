package groupapp

import (
	"errors"
	"fmt"
	"strings"

	domaintree "zero-web-server/internal/domain/tree"
	"zero-web-server/internal/infrastructure/persistence"
)

var (
	ErrGroupNameRequired     = errors.New("分组名称不可为NULL")
	ErrGroupDeviceIDRequired = errors.New("分组编号不可为NULL")
	ErrGroupDeviceIDLength   = errors.New("分组编号必须为20位")
	ErrGroupDeviceIDInvalid  = errors.New("分组编号不满足国标定义")
	ErrGroupDuplicate        = errors.New("该节点编号已存在")
	ErrGroupNotFound         = errors.New("分组不存在")
	ErrBusinessGroupMissing  = errors.New("所属的业务分组分组不存在")
	ErrParentGroupMissing    = errors.New("所属的上级分组分组不存在")
)

type Service struct {
	repo *persistence.GroupRegionRepository
}

func NewService(repo *persistence.GroupRegionRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) QueryForTree(query string, parentID *int, hasChannel *bool) ([]domaintree.Node, error) {
	nodes, err := s.repo.QueryGroupTree(query, parentID)
	if err != nil {
		return nil, err
	}
	if parentID == nil || hasChannel == nil || !*hasChannel {
		return nodes, nil
	}
	parent, err := s.repo.GetGroupByID(*parentID)
	if err != nil {
		return nodes, nil
	}
	channels, err := s.repo.QueryGroupChannels(query, parent.DeviceID)
	if err != nil {
		return nodes, nil
	}
	return append(nodes, channels...), nil
}

func (s *Service) Add(group *persistence.GroupRecord) error {
	if err := validateGroupRecord(group, 0); err != nil {
		return err
	}
	exists, err := s.repo.ExistsGroupDeviceID(group.DeviceID, 0)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("%w: %s", ErrGroupDuplicate, group.DeviceID)
	}

	typeCode := gbTypeCode(group.DeviceID)
	if typeCode == "215" {
		return s.repo.AddBusinessGroup(group)
	}
	if typeCode != "216" {
		return errors.New("创建虚拟组织时设备编号11-13位应使用216")
	}
	if strings.TrimSpace(group.BusinessGroup) == "" {
		return ErrBusinessGroupMissing
	}
	if _, err := s.repo.GetBusinessGroup(group.BusinessGroup); err != nil {
		return ErrBusinessGroupMissing
	}
	if parentDeviceID := strings.TrimSpace(group.ParentDeviceID); parentDeviceID != "" {
		if _, err := s.repo.GetGroupByDeviceAndBusiness(parentDeviceID, group.BusinessGroup); err != nil {
			return ErrParentGroupMissing
		}
	} else {
		group.ParentDeviceID = ""
	}
	return s.repo.AddVirtualGroup(group)
}

func (s *Service) Update(group *persistence.GroupRecord) error {
	if group.ID <= 0 {
		return errors.New("更新必须携带分组ID")
	}
	if err := validateGroupRecord(group, group.ID); err != nil {
		return err
	}
	if _, err := s.repo.GetGroupRecordByID(group.ID); err != nil {
		return ErrGroupNotFound
	}
	exists, err := s.repo.ExistsGroupDeviceID(group.DeviceID, group.ID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("%w: %s", ErrGroupDuplicate, group.DeviceID)
	}
	return s.repo.UpdateGroup(group)
}

func (s *Service) Delete(id int) error {
	group, err := s.repo.GetGroupRecordByID(id)
	if err != nil {
		return ErrGroupNotFound
	}
	typeCode := gbTypeCode(group.DeviceID)
	var toDelete []persistence.GroupRecord
	if typeCode == "215" {
		children, err := s.repo.ListGroupsByBusinessGroup(group.DeviceID)
		if err != nil {
			return err
		}
		toDelete = append([]persistence.GroupRecord{*group}, children...)
		if err := s.repo.ClearChannelsByBusinessGroup(group.DeviceID); err != nil {
			return err
		}
	} else {
		descendants, err := s.repo.CollectGroupDescendants(id)
		if err != nil {
			return err
		}
		toDelete = descendants
		deviceIDs := make([]string, 0, len(descendants))
		for _, item := range descendants {
			deviceIDs = append(deviceIDs, item.DeviceID)
		}
		if err := s.repo.ClearChannelsByParentDeviceIDs(deviceIDs); err != nil {
			return err
		}
	}
	ids := make([]int, 0, len(toDelete))
	for _, item := range toDelete {
		ids = append(ids, item.ID)
	}
	return s.repo.DeleteGroupsByIDs(ids)
}

func (s *Service) GetPath(deviceID, businessGroup string) ([]persistence.GroupRecord, error) {
	if _, err := s.repo.GetBusinessGroup(businessGroup); err != nil {
		return nil, errors.New("业务分组不存在")
	}
	group, err := s.repo.GetGroupByDeviceAndBusiness(deviceID, businessGroup)
	if err != nil {
		return nil, errors.New("虚拟组织不存在")
	}
	path := make([]persistence.GroupRecord, 0, 4)
	current := group
	for {
		path = append([]persistence.GroupRecord{*current}, path...)
		if current.ParentDeviceID == "" || current.DeviceID == current.BusinessGroup {
			break
		}
		parent, err := s.repo.GetGroupByDeviceAndBusiness(current.ParentDeviceID, current.BusinessGroup)
		if err != nil {
			break
		}
		current = parent
	}
	return path, nil
}

func validateGroupRecord(group *persistence.GroupRecord, excludeID int) error {
	_ = excludeID
	if strings.TrimSpace(group.Name) == "" {
		return ErrGroupNameRequired
	}
	group.DeviceID = strings.TrimSpace(group.DeviceID)
	if group.DeviceID == "" {
		return ErrGroupDeviceIDRequired
	}
	if len(group.DeviceID) != 20 {
		return ErrGroupDeviceIDLength
	}
	if gbTypeCode(group.DeviceID) == "" {
		return ErrGroupDeviceIDInvalid
	}
	return nil
}

func gbTypeCode(deviceID string) string {
	if len(deviceID) < 13 {
		return ""
	}
	code := deviceID[10:13]
	if code == "215" || code == "216" {
		return code
	}
	return ""
}

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "1062")
}
