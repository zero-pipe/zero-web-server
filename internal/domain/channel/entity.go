package channel

type Channel struct {
	ID              int     `json:"id"`
	InternalCode    string  `json:"internalCode"`
	DeviceID        string  `json:"deviceId"`
	DataType        int     `json:"dataType"`
	DataDeviceID    int     `json:"dataDeviceId"`
	GBDeviceID      string  `json:"gbDeviceId"`
	Name            string  `json:"name"`
	Manufacturer    string  `json:"manufacturer"`
	Model           string  `json:"model"`
	Status          string  `json:"status"`
	PTZType         int     `json:"ptzType"`
	Parental        int     `json:"parental"`
	ParentID        string  `json:"parentId"`
	BusinessGroupID string  `json:"businessGroupId"`
	// CivilCode 行政区划编码（gb_civil_code，级联目录挂载用）。
	CivilCode string `json:"civilCode"`
	Longitude       float64 `json:"longitude"`
	Latitude        float64 `json:"latitude"`
	HasAudio        bool    `json:"hasAudio"`
	SubCount        int     `json:"subCount"`
	// StreamIdentification 主/子码流选择（如 stream:0/stream:1、streamnumber:0/1）。
	// GB28181 目录不上报设备侧选择，平台默认主码流(stream:0)，可由用户覆盖并随点播 SDP 下发。
	StreamIdentification string `json:"streamIdentification"`
	CreateTime           string `json:"createTime"`
	UpdateTime           string `json:"updateTime"`
}

type Repository interface {
	GetOne(deviceID, channelDeviceID string) (*Channel, error)
	GetByID(id int) (*Channel, error)
	GetByInternalCode(code string) (*Channel, error)
	ListByDevice(deviceID string, page, count int, query string, online *bool) ([]*Channel, int64, error)
	ResetByDevice(deviceID string, dataDeviceID int, channels []*Channel) error
	DeleteByDevice(deviceID string) error
	ChangeAudio(channelID int, audio bool) error
	Update(ch *Channel) error
	// UpdateStreamIdentification 单通道设置码流类型；deviceID 非空时按设备批量设置（id<=0 时）。
	UpdateStreamIdentification(id int, deviceID string, value string) error

	// 业务分组 / 行政区划 / 通用检索相关（channel、group、region 应用服务使用）
	ListCommon(page, count int, query string, channelType *int, online *bool, hasRecordPlan *bool) ([]*Channel, int64, error)
	ListAll(page, count int, query string) ([]*Channel, int64, error)
	ListByCivilCode(page, count int, query string, channelType *int, online *bool, civilCode string) ([]*Channel, int64, error)
	ListByGroupParent(page, count int, query string, channelType *int, online *bool, groupDeviceID string) ([]*Channel, int64, error)
	ListIDsByDevice(deviceID string) ([]int, error)
	CountCommonByIDs(channelIDs []int) (int64, error)
	CountCommonByDataDeviceIDs(dataDeviceIDs []int) (int64, error)
	SetCivilCode(civilCode string, channelIDs []int) error
	ClearCivilCode(channelIDs []int) error
	SetCivilCodeByDataDeviceIDs(civilCode string, dataDeviceIDs []int) error
	ClearCivilCodeByDataDeviceIDs(dataDeviceIDs []int) error
	SetGroup(parentID, businessGroup string, channelIDs []int) error
	SetGroupByDataDeviceIDs(parentID, businessGroup string, dataDeviceIDs []int) error
	ClearGroupParent(channelIDs []int) error
	ClearGroupByDataDeviceIDs(dataDeviceIDs []int) error
}
