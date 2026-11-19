package subordinate

// Platform is a downstream GB28181 platform that REGISTERs to this node.
type Platform struct {
	ID           int    `json:"id"`
	Enable       bool   `json:"enable"`
	Name         string `json:"name"`
	DeviceGBID   string `json:"deviceGBId"`
	Password     string `json:"password"`
	Transport    string `json:"transport"`
	Status       bool   `json:"status"`
	IP           string `json:"ip"`
	Port         int    `json:"port"`
	HostAddress  string `json:"hostAddress"`
	Expires      int    `json:"expires"`
	RegisterCall string `json:"registerCallId"`
	ServerID     string `json:"serverId"`
	CreateTime   string `json:"createTime"`
	UpdateTime   string `json:"updateTime"`
}

type Repository interface {
	GetByID(id int) (*Platform, error)
	GetByGBID(gbID string) (*Platform, error)
	List(page, count int, query string) ([]*Platform, int64, error)
	Create(p *Platform) error
	Update(p *Platform) error
	Delete(id int) error
	UpdateOnline(gbID, ip string, port, expires int, callID, transport string) error
	UpdateOffline(gbID string) error

	// 下级目录（级联接收）。
	ListCatalog(platformID int) ([]*CatalogEntry, error)
	GetCatalog(platformID int, deviceID string) (*CatalogEntry, error)
	UpsertCatalog(e *CatalogEntry) error
	DeleteCatalog(platformID int, deviceID string) error
}

// CatalogEntry 下级平台推送来的目录项（节点或通道）。
type CatalogEntry struct {
	ID           int    `json:"id"`
	PlatformID   int    `json:"platformId"`
	DeviceID     string `json:"deviceId"`
	Name         string `json:"name"`
	ParentID     string `json:"parentId"`
	TypeCode     string `json:"typeCode"`
	ItemType     string `json:"itemType"` // node | channel
	Status       string `json:"status"`
	CivilCode    string `json:"civilCode"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Orphan       bool   `json:"orphan"`
	RawJSON      string `json:"rawJson"`
	CreateTime   string `json:"createTime"`
	UpdateTime   string `json:"updateTime"`
}

// 目录项类别：node=目录节点（区域/分组/组织/平台），channel=通道/设备点位。
const (
	ItemNode    = "node"
	ItemChannel = "channel"
)
