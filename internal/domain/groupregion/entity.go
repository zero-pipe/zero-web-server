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

type Repository interface {
	QueryGroupTree(query string, parentID *int) ([]domaintree.Node, error)
	GetGroupByID(id int) (*domaintree.Node, error)
	QueryGroupChannels(query, parentDeviceID string) ([]domaintree.Node, error)
	QueryRegionTree(parentID *int) ([]domaintree.Node, error)
	GetRegionByID(id int) (*domaintree.Node, error)
	QueryRegionChannels(parentDeviceID string) ([]domaintree.Node, error)
	AddRegion(region *RegionRecord) error
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
