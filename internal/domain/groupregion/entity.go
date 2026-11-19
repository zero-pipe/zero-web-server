package groupregion

import domaintree "zero-web-server/internal/domain/tree"

// GroupRecord 业务分组记录（领域实体）。
type GroupRecord struct {
	ID             int    `json:"id"`
	DeviceID       string `json:"deviceId"`
	Name           string `json:"name"`
	ParentID       *int   `json:"parentId"`
	ParentDeviceID string `json:"parentDeviceId"`
	BusinessGroup  string `json:"businessGroup"`
	CivilCode      string `json:"civilCode"`
	Alias          string `json:"alias"`
}

// RegionRecord 行政区划记录（领域实体）。
type RegionRecord struct {
	ID             int    `json:"id"`
	DeviceID       string `json:"deviceId"`
	Name           string `json:"name"`
	ParentID       *int   `json:"parentId"`
	ParentDeviceID string `json:"parentDeviceId"`
}

// RegionCodeMapping 区划节点对外国标码映射（为空则按默认规则推导）。
type RegionCodeMapping struct {
	RegionDeviceID string `json:"regionDeviceId"`
	GBCode         string `json:"gbCode"`
}

type Repository interface {
	QueryGroupTree(query string, parentID *int) ([]domaintree.Node, error)
	GetGroupByID(id int) (*domaintree.Node, error)
	QueryGroupChannels(query, parentDeviceID string) ([]domaintree.Node, error)
	QueryRegionTree(parentID *int) ([]domaintree.Node, error)
	// ListAllRegions / ListAllGroups 返回全部区域(0)与分组(215/216)节点，供级联目录组装。
	ListAllRegions() ([]domaintree.Node, error)
	ListAllGroups() ([]domaintree.Node, error)
	GetRegionByID(id int) (*domaintree.Node, error)
	QueryRegionChannels(parentDeviceID string) ([]domaintree.Node, error)
	AddRegion(region *RegionRecord) error
	// GetRegionByDeviceID 按国标/区划编号查节点（一键建树去重用）。
	GetRegionByDeviceID(deviceID string) (*domaintree.Node, error)
	// ListRegionChildren 返回直接子区划（删除子树时逐层收集）。
	ListRegionChildren(parentID int) ([]RegionRecord, error)
	// DeleteRegionsByIDs 删除区划节点（调用方负责先删子后删父）。
	DeleteRegionsByIDs(ids []int) error
	// ClearChannelsByCivilCodes 解除通道对区划的挂载（gb_civil_code 置空）。
	ClearChannelsByCivilCodes(civilCodes []string) error
	// 区划节点对外编码映射（公安等场景上级指定编码）。
	ListRegionCodeMappings() ([]*RegionCodeMapping, error)
	UpsertRegionCodeMapping(m *RegionCodeMapping) error
	DeleteRegionCodeMapping(regionDeviceID string) error
	UpdateRegion(region *RegionRecord) error
	GetGroupRecordByID(id int) (*GroupRecord, error)
	AddBusinessGroup(group *GroupRecord) error
	AddVirtualGroup(group *GroupRecord) error
	GetBusinessGroup(businessGroup string) (*GroupRecord, error)
	GetGroupByDeviceAndBusiness(deviceID, businessGroup string) (*GroupRecord, error)
	ResolveBusinessGroup(deviceID string) (string, error)
	ExistsGroupDeviceID(deviceID string, excludeID int) (bool, error)
	UpdateGroup(group *GroupRecord) error
	ListGroupsByBusinessGroup(businessGroup string) ([]GroupRecord, error)
	CollectGroupDescendants(rootID int) ([]GroupRecord, error)
	DeleteGroupsByIDs(ids []int) error
	ClearChannelsByBusinessGroup(businessGroup string) error
	ClearChannelsByParentDeviceIDs(deviceIDs []string) error
}
