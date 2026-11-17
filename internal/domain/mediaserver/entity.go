package mediaserver

// MediaServer 媒体节点配置（领域实体）。
//
// 除手工添加外，节点也可由 ZMS 启动时自注册（见 Register），
// 此时 CapabilitiesReported=true，且 *_Port / Enable* 字段为 ZMS 实际上报值。
type MediaServer struct {
	ID            string `json:"id"`
	IP            string `json:"ip"`
	HookIP        string `json:"hookIp"`
	SDPIP         string `json:"sdpIp"`
	StreamIP      string `json:"streamIp"`
	HTTPPort      int    `json:"httpPort"`
	HTTPSSLPort   int    `json:"httpSSlPort"`
	RTMPPort      int    `json:"rtmpPort"`
	RTMPSSLPort   int    `json:"rtmpSslPort"`
	RTSPPort      int    `json:"rtspPort"`
	RTSPSSLPort   int    `json:"rtspSslPort"`
	SRTPort       int    `json:"srtPort"`
	RTPProxyPort  int    `json:"rtpProxyPort"`
	WebRTCPortMin int    `json:"webrtcPortMin"`
	WebRTCPortMax int    `json:"webrtcPortMax"`

	// Enable* 为 ZMS 上报的协议能力开关；CapabilitiesReported 为 false（旧手工节点）时按「全开」处理。
	CapabilitiesReported bool `json:"capabilitiesReported"`
	EnableRTSP           bool `json:"enableRtsp"`
	EnableRTMP           bool `json:"enableRtmp"`
	EnableHLS            bool `json:"enableHls"`
	EnableSRT            bool `json:"enableSrt"`
	EnableWebRTC         bool `json:"enableWebrtc"`

	Secret            string `json:"secret"`
	Type              string `json:"type"`
	Version           string `json:"version"`
	DefaultServer     bool   `json:"defaultServer"`
	HookAliveInterval int    `json:"hookAliveInterval"`
	LastSeen          int64  `json:"lastSeen"` // 最近一次注册/心跳（unix ms）
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
	UpdateLastSeen(id string, ts int64) error
}
