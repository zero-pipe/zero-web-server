package dict

import "errors"

// ErrNotFound 字典类型/项不存在。
var ErrNotFound = errors.New("dict not found")

// Type 字典类型。
type Type struct {
	ID         int64  `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Remark     string `json:"remark"`
	Builtin    bool   `json:"builtin"`
	Sort       int    `json:"sort"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

// Item 字典项。
type Item struct {
	ID         int64  `json:"id"`
	TypeCode   string `json:"typeCode"`
	Value      string `json:"value"`
	Label      string `json:"label"`
	Sort       int    `json:"sort"`
	Enabled    bool   `json:"enabled"`
	Builtin    bool   `json:"builtin"`
	Remark     string `json:"remark"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

type Repository interface {
	ListTypes() ([]Type, error)
	GetTypeByCode(code string) (*Type, error)
	GetTypeByID(id int64) (*Type, error)
	CreateType(t *Type) error
	UpdateType(t *Type) error
	DeleteType(id int64) error

	ListItems(typeCode string, enabledOnly bool) ([]Item, error)
	GetItemByID(id int64) (*Item, error)
	CreateItem(it *Item) error
	UpdateItem(it *Item) error
	DeleteItem(id int64) error
}
