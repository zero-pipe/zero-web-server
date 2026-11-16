package recordplan

// Plan 录像计划（领域实体）。
type Plan struct {
	ID         int    `json:"id"`
	Snap       bool   `json:"snap"`
	Name       string `json:"name"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

// Item 录像计划时段。
type Item struct {
	ID         int    `json:"id"`
	Start      int    `json:"start"`
	Stop       int    `json:"stop"`
	WeekDay    int    `json:"weekDay"`
	PlanID     int    `json:"planId"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

// ChannelRecord 计划通道视图（字段/JSON 与 zws_device_channel 对齐）。
type ChannelRecord struct {
	ID                int     `json:"id"`
	InternalCode      string  `json:"internalCode"`
	DeviceID          string  `json:"deviceId"`
	Name              string  `json:"name"`
	Manufacturer      string  `json:"manufacturer"`
	Model             string  `json:"model"`
	Owner             string  `json:"owner"`
	CivilCode         string  `json:"civilCode"`
	Block             string  `json:"block"`
	Address           string  `json:"address"`
	Parental          int     `json:"parental"`
	ParentID          string  `json:"parentId"`
	Status            string  `json:"status"`
	Longitude         float64 `json:"longitude"`
	Latitude          float64 `json:"latitude"`
	PTZType           int     `json:"ptzType"`
	BusinessGroupID   string  `json:"businessGroupId"`
	CreateTime        string  `json:"createTime"`
	UpdateTime        string  `json:"updateTime"`
	SubCount          int     `json:"subCount"`
	HasAudio          bool    `json:"hasAudio"`
	ChannelType       int     `json:"channelType"`
	MapLevel          int     `json:"mapLevel"`
	GBDeviceID        string  `json:"gbDeviceId"`
	GBName            string  `json:"gbName"`
	GBManufacturer    string  `json:"gbManufacturer"`
	GBModel           string  `json:"gbModel"`
	GBCivilCode       string  `json:"gbCivilCode"`
	GBParentID        string  `json:"gbParentId"`
	GBStatus          string  `json:"gbStatus"`
	GBLongitude       float64 `json:"gbLongitude"`
	GBLatitude        float64 `json:"gbLatitude"`
	GBBusinessGroupID string  `json:"gbBusinessGroupId"`
	GBPTZType         int     `json:"gbPtzType"`
	RecordPlanID      int     `json:"recordPlanId"`
	DataType          int     `json:"dataType"`
	DataDeviceID      int     `json:"dataDeviceId"`
}

type Repository interface {
	Create(plan *Plan, items []Item) error
	GetByID(id int) (*Plan, []Item, error)
	Update(plan *Plan, items []Item) error
	Delete(id int) error
	List(page, count int, query string) ([]Plan, int64, error)
	CountChannels(planID int) int64
	LinkAll(planID int) error
	UnlinkAll(planID int) error
	LinkChannels(planID int, channelIDs []int) error
	UnlinkChannels(channelIDs []int) error
	LinkChannelsByDevice(planID int, deviceDbIDs []int) error
	UnlinkChannelsByDevice(deviceDbIDs []int) error
	ChannelList(page, count int, planID int, query string, hasLink *bool, online, channelType string) ([]ChannelRecord, int64, error)
	QueryRecordingChannels(weekDay, minuteIndex int) ([]ChannelRecord, error)
}
