package port

import "context"

// MediaNode 流媒体集群中的一个节点视图（列表/运维用，不含网关操作）。
type MediaNode struct {
	ID       string
	IP       string
	HTTPPort int
	Secret   string
	Type     string
	Online   bool
	Load     int64
	SDPIP    string
	StreamIP string
}

// StreamProbe 节点上某路流的探测信息（平台只关心可播性与编解码）。
type StreamProbe struct {
	VideoCodec  string
	AudioCodec  string
	Video       bool
	Audio       bool
	ReaderCount int
	AliveSecond int64
	BytesSpeed  int64
	VideoFps    uint
	Width       int
	Height      int
}

// ProxyResult 拉流代理创建结果。
type ProxyResult struct {
	Code int
	Msg  string
	Key  string
}

// StreamPlayURLSet 平台层播放地址集合。
type StreamPlayURLSet struct {
	Flv, WsFlv, Hls, Rtmp, Rtsp, Rtc, Rtcs string
}

// MediaEndpoint 已解析的媒体节点 + 对外部 ZMS 的网关操作。
// 应用层依赖本接口，禁止直接依赖 mediaserver.Node / mediakit.Client。
type MediaEndpoint interface {
	ID() string
	SDPIP() string
	StreamIP() string
	Secret() string
	BaseURL() string
	SignalingBaseURL(serverPort int) string
	PlayURLs(app, stream string) map[string]string
	StreamPlayURLs(app, stream string, webrtcPush bool, signalingPort int) StreamPlayURLSet

	OpenRtpServer(ctx context.Context, app, stream string, port, tcpMode int) (rtpPort int, err error)
	ConnectRtpServer(ctx context.Context, app, stream, host string, port int) error
	CloseStreams(ctx context.Context, vhost, app, stream string) error
	LookupStream(ctx context.Context, app, stream string) *StreamProbe
	AddStreamProxy(ctx context.Context, vhost, app, stream, streamURL, rtpType string, enableHLS, enableMP4, autoClose bool) (*ProxyResult, error)
	DelStreamProxy(ctx context.Context, key string) error
	DeleteRecordFile(ctx context.Context, filePath string) error
	LoadMP4File(ctx context.Context, app, stream, filePath string) error
}

// MediaCluster 流媒体集群配置 + 简单调度端口。
// 平台不实现流媒体本身，只做节点配置、健康与选路。
type MediaCluster interface {
	List(ctx context.Context) ([]MediaNode, error)
	// Resolve 按 preferID 选节点；空或 auto 走负载策略。
	Resolve(ctx context.Context, preferID string) (MediaEndpoint, error)
	// ResolveForStream 同一 app/stream 尽量粘滞到同一节点。
	ResolveForStream(ctx context.Context, app, stream, preferID string) (MediaEndpoint, error)
	SelectMinimumLoad(ctx context.Context) (MediaEndpoint, error)
	BindStream(nodeID, app, stream string)
	UnbindStream(app, stream string)
	StreamNodeID(app, stream string) (string, bool)
}

// LoadScheduler 通用简单调度（least-load 等）。
type LoadScheduler[T any] interface {
	Pick(candidates []T) (T, error)
}

// LoadAware 可参与负载调度的候选。
type LoadAware interface {
	NodeID() string
	CurrentLoad() int64
	Healthy() bool
}
