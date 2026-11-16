package mediaserver

// MediaServer 媒体节点配置（领域实体）。
type MediaServer struct {
	ID                string `json:"id"`
	IP                string `json:"ip"`
	HookIP            string `json:"hookIp"`
	SDPIP             string `json:"sdpIp"`
	StreamIP          string `json:"streamIp"`
	HTTPPort          int    `json:"httpPort"`
	HTTPSSLPort       int    `json:"httpSSlPort"`
	RTMPPort          int    `json:"rtmpPort"`
	RTSPPort          int    `json:"rtspPort"`
	RTPProxyPort      int    `json:"rtpProxyPort"`
	Secret            string `json:"secret"`
	Type              string `json:"type"`
	DefaultServer     bool   `json:"defaultServer"`
	HookAliveInterval int    `json:"hookAliveInterval"`
	CreateTime        string `json:"createTime"`
	UpdateTime        string `json:"updateTime"`
	ServerID          string `json:"serverId"`
}

type Repository interface {
	ListAll() ([]MediaServer, error)
	GetByID(id string) (*MediaServer, error)
	Save(m *MediaServer) error
	Delete(id string) error
	ClearAllDefault() error
}
