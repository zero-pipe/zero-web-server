package tree

// Node matches ZWS GroupTree / RegionTree JSON shape for lazy tree UIs.
type Node struct {
	ID       int    `json:"id" gorm:"column:id"`
	DeviceID string `json:"deviceId" gorm:"column:device_id"`
	Name     string `json:"name" gorm:"column:name"`
	// 平台身份内码（业务关联唯一键）：设备 D... / 通道 C...。非自增主键、非国标码。
	InternalCode       string `json:"internalCode,omitempty" gorm:"column:internal_code"`
	DeviceInternalCode string `json:"deviceInternalCode,omitempty" gorm:"column:device_internal_code"`
	ParentID           *int   `json:"parentId,omitempty" gorm:"column:parent_id"`
	ParentDeviceID     string `json:"parentDeviceId,omitempty" gorm:"column:parent_device_id"`
	BusinessGroup      string `json:"businessGroup,omitempty" gorm:"column:business_group"`
	CivilCode          string `json:"civilCode,omitempty" gorm:"column:civil_code"`
	Alias              string `json:"alias,omitempty" gorm:"column:alias"`
	TreeID             string `json:"treeId" gorm:"column:tree_id"`
	Type               int    `json:"type" gorm:"column:type"`
	IsLeaf             bool   `json:"isLeaf" gorm:"column:is_leaf"`
	Leaf               bool   `json:"leaf,omitempty"`
	Status             string `json:"status" gorm:"column:status"`
}
