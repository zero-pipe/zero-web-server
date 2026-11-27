package storagenode

// StorageNode 云存储节点（独立于媒体节点 zws_media_server）。
//
// zstorage 作为旁路录制/云存储服务，自注册到本表；录制/回放路由到该节点，
// 不参与取流媒体节点的选择（Resolve/SelectMinimumLoad）。
type StorageNode struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"nodeId"`      // zstorage 上报 media_server_id
	Name        string `json:"name"`
	IP          string `json:"ip"`          // 控制面/出流可达 IP
	HTTPPort    int    `json:"httpPort"`    // zstorage HTTP/API/FLV 端口
	RTSPPort    int    `json:"rtspPort"`
	Secret      string `json:"secret"`      // 与 zstorage [api] secret 一致
	Version     string `json:"version"`
	CapacityGB  int    `json:"capacityGb"`
	UsedGB      int    `json:"usedGb"`
	Online      bool   `json:"online"`
	LastSeen    int64  `json:"lastSeen"` // unix ms
	DefaultNode bool   `json:"defaultNode"`
	CreateTime  string `json:"createTime"`
	UpdateTime  string `json:"updateTime"`
}

type Repository interface {
	ListAll() ([]StorageNode, error)
	GetByNodeID(nodeID string) (*StorageNode, error)
	Save(n *StorageNode) error
	Delete(nodeID string) error
	UpdateLastSeen(nodeID string, ts int64, online bool) error
	ClearAllDefault() error
}
