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
}
