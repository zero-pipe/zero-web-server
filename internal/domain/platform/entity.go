package platform

type Platform struct {
	ID              int    `json:"id"`
	Enable          bool   `json:"enable"`
	Name            string `json:"name"`
	ServerGBID      string `json:"serverGBId"`
	ServerGBDomain  string `json:"serverGBDomain"`
	ServerIP        string `json:"serverIP"`
	ServerPort      int    `json:"serverPort"`
	DeviceGBID      string `json:"deviceGBId"`
	DeviceIP        string `json:"deviceIP"`
	DevicePort      string `json:"devicePort"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Expires         string `json:"expires"`
	KeepTimeout     string `json:"keepTimeout"`
	Transport       string `json:"transport"`
	Status          bool   `json:"status"`
	AutoPushChannel bool   `json:"autoPushChannel"`
	ServerID        string `json:"serverId"`
	CreateTime      string `json:"createTime"`
	UpdateTime      string `json:"updateTime"`
}

// CatalogSnapshot 记录上一次推送给该平台的目录，用于计算 ADD/UPDATE/DEL。
type CatalogSnapshot struct {
	PlatformID int    `json:"platformId"`
	DeviceID   string `json:"deviceId"`
	Name       string `json:"name"`
	ParentID   string `json:"parentId"`
	ItemType   string `json:"itemType"`
	Status     string `json:"status"`
	Hash       string `json:"hash"`
}

type Repository interface {
	GetByID(id int) (*Platform, error)
	GetByServerGBID(serverGBID string) (*Platform, error)
	List(page, count int, query string) ([]*Platform, int64, error)
	Create(p *Platform) error
	Update(p *Platform) error
	Delete(id int) error
	UpdateStatus(id int, online bool) error
	ListCatalogSnapshot(platformID int) ([]*CatalogSnapshot, error)
	// 按业务分组（215/216 节点）共享（级联共享源）。
	ListGroupShares(platformID int) ([]string, error)
	AddGroupShares(platformID int, groupDeviceIDs []string) error
	RemoveGroupShares(platformID int, groupDeviceIDs []string) error
	ReplaceCatalogSnapshot(platformID int, list []*CatalogSnapshot) error
	ListEnabled() ([]*Platform, error)
}

// Channel 上级平台共享通道映射（领域实体）。
type Channel struct {
	ID              int    `json:"id"`
	PlatformID      int    `json:"platformId"`
	DeviceChannelID int    `json:"deviceChannelId"`
	CustomDeviceID  string `json:"customDeviceId"`
	CustomName      string `json:"customName"`
}

type ChannelRepository interface {
	ListPlatformChannels(platformID int) ([]int, error)
	AddPlatformChannel(platformID, channelID int) error
	RemovePlatformChannel(platformID, channelID int) error
	ListChannelIDsByDevice(deviceID string) ([]int, error)
	GetPlatformChannel(platformID, channelID int) (*Channel, error)
	GetByCustomDeviceID(platformID int, customID string) (*Channel, error)
	UpdatePlatformChannelCustom(platformID, channelID int, customID, customName string) error
}
