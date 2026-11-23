// Package customevent 持久化"自定义事件"：平台不理解其语义，只按基础字段存 raw。
// 与平台已知事件（domain/event）完全隔离，单独表 zws_custom_event。
package customevent

// CustomEvent 是一条自定义事件记录。平台只保证内码关联与基础信封，raw 原样保存。
type CustomEvent struct {
	ID                  string `json:"id"`
	EventID             string `json:"eventId"`
	DeviceInternalCode  string `json:"deviceInternalCode"`
	ChannelInternalCode string `json:"channelInternalCode"`
	DeviceID            string `json:"deviceId"`
	ResourceID          string `json:"resourceId"`
	MajorType           string `json:"majorType"` // 大类
	MinorType           string `json:"minorType"` // 小类（上报什么就是什么）
	OccurAt             int64  `json:"occurAt"`
	Level               string `json:"level"`
	ImageKey            string `json:"imageKey"`
	ImageURL            string `json:"imageUrl"`
	RawJSON             string `json:"rawJson"`
	CreateTime          string `json:"createTime"`
}

// ListFilter 自定义事件查询条件。
type ListFilter struct {
	DeviceInternalCode  string
	ChannelInternalCode string
	MajorType           string
	MinorType           string
	Begin               int64
	End                 int64
}

// TypeCount 是 (大类, 小类) 聚合，供筛选下拉。
type TypeCount struct {
	MajorType string `json:"majorType"`
	MinorType string `json:"minorType"`
	Count     int64  `json:"count"`
}

// Repository 自定义事件持久化端口。
type Repository interface {
	Create(e *CustomEvent) error
	Exists(eventID string) (bool, error)
	Get(id string) (*CustomEvent, error)
	List(page, count int, f ListFilter) ([]*CustomEvent, int64, error)
	DistinctTypes(deviceInternalCode string) ([]TypeCount, error)
}
