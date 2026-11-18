package port

import "context"

// IPCStreamSource 是插件返回的流源（不包含最终播放地址）。
type IPCStreamSource struct {
	Transport string `json:"transport"` // rtsp_url | rtp_push | rtmp_push
	RTSPURL   string `json:"rtspUrl"`
	PushURL   string `json:"pushUrl"`
}

// IPCChannel 是插件上报的通道。
type IPCChannel struct {
	ChannelID string `json:"channelId"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	PTZType   int    `json:"ptzType"`
	HasAudio  bool   `json:"hasAudio"`
}

// IPCDeviceInfo 是插件登录后返回的设备信息。
type IPCDeviceInfo struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Firmware     string `json:"firmware"`
	SerialNumber string `json:"serialNumber"`
}

// IPCPushParams 是平台为 rtp_push 分配的收流参数。
type IPCPushParams struct {
	SDPIP   string `json:"sdpIp"`
	RTPPort int    `json:"rtpPort"`
	Mode    string `json:"mode"` // UDP / TCP
}

// IPCArmMQTT 是布防时下发给插件的 MQTT 连接参数。
type IPCArmMQTT struct {
	Broker     string `json:"broker"`
	ClientID   string `json:"clientId"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Topic      string `json:"topic"`
	QoS        int    `json:"qos"`
	KeepAlive  int    `json:"keepAlive"`
	TLSEnabled bool   `json:"tls"`
	CACert     string `json:"caCert"`
	ClientCert string `json:"clientCert"`
	ClientKey  string `json:"clientKey"`
}

// IPCArmObjectStore 是布防时下发给插件的对象存储参数。
type IPCArmObjectStore struct {
	Provider   string `json:"provider"`
	Endpoint   string `json:"endpoint"`
	Region     string `json:"region"`
	Bucket     string `json:"bucket"`
	AccessKey  string `json:"accessKey"`
	SecretKey  string `json:"secretKey"`
	UseSSL     bool   `json:"useSSL"`
	PathStyle  bool   `json:"pathStyle"`
	PublicBase string `json:"publicBase"`
}

// IPCArmChannel 是布防时下发的通道内码映射（协议侧通道号 → 通道内码）。
type IPCArmChannel struct {
	ChannelID    string `json:"channel_id"`
	InternalCode string `json:"internal_code"`
}

// IPCArmConfig 是布防请求。
type IPCArmConfig struct {
	DeviceUID          string            `json:"device_uid"`
	DeviceInternalCode string            `json:"device_internal_code"`
	Channels           []IPCArmChannel   `json:"channels"`
	EventTypes         []string          `json:"event_types"`
	MQTT               IPCArmMQTT        `json:"mqtt"`
	ObjectStore        IPCArmObjectStore `json:"object_store"`
}

// IPCArmResult 是布防结果。
type IPCArmResult struct {
	Armed          bool   `json:"armed"`
	SubscriptionID string `json:"subscriptionId"`
}

// IPCClient 是平台调用 sidecar 插件的 RPC 客户端端口。
type IPCClient interface {
	Login(ctx context.Context, deviceUID string, params map[string]any) (*IPCDeviceInfo, error)
	Logout(ctx context.Context, deviceUID string) error
	ListChannels(ctx context.Context, deviceUID string) ([]IPCChannel, error)
	StartLive(ctx context.Context, deviceUID, channelID, streamKey, preferTransport string, push *IPCPushParams) (*IPCStreamSource, error)
	StopLive(ctx context.Context, deviceUID, channelID, streamKey string) error
	PTZ(ctx context.Context, deviceUID, channelID, direction string, h, v, z int) error
	Arm(ctx context.Context, cfg IPCArmConfig) (*IPCArmResult, error)
	Disarm(ctx context.Context, deviceUID, subscriptionID string) error
}

// IPCClientFactory 按插件 endpoint 构造 RPC 客户端。
type IPCClientFactory interface {
	For(endpoint string) IPCClient
}
