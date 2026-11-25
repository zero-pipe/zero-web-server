package model

// DictType 数据字典类型（device_type/channel_type/coordinate_system/manufacturer…）。
type DictType struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Code       string `gorm:"column:code;size:64;uniqueIndex:uk_dict_type_code" json:"code"`
	Name       string `gorm:"column:name;size:128" json:"name"`
	Remark     string `gorm:"column:remark;size:255" json:"remark"`
	Builtin    bool   `gorm:"column:builtin" json:"builtin"`
	Sort       int    `gorm:"column:sort" json:"sort"`
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (DictType) TableName() string { return "zws_dict_type" }

// DictItem 数据字典项（某类型下的枚举值）。
type DictItem struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TypeCode   string `gorm:"column:type_code;size:64;uniqueIndex:uk_dict_item,priority:1;index:idx_dict_item_type" json:"typeCode"`
	Value      string `gorm:"column:value;size:64;uniqueIndex:uk_dict_item,priority:2" json:"value"`
	Label      string `gorm:"column:label;size:128" json:"label"`
	Sort       int    `gorm:"column:sort" json:"sort"`
	Enabled    bool   `gorm:"column:enabled;default:true" json:"enabled"`
	Builtin    bool   `gorm:"column:builtin" json:"builtin"`
	Remark     string `gorm:"column:remark;size:255" json:"remark"`
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (DictItem) TableName() string { return "zws_dict_item" }
