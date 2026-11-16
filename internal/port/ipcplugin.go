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

// IPCClient 是平台调用 sidecar 插件的 RPC 客户端端口。
type IPCClient interface {
	Login(ctx context.Context, deviceUID string, params map[string]any) (*IPCDeviceInfo, error)
	Logout(ctx context.Context, deviceUID string) error
	ListChannels(ctx context.Context, deviceUID string) ([]IPCChannel, error)
	StartLive(ctx context.Context, deviceUID, channelID, streamKey string) (*IPCStreamSource, error)
	StopLive(ctx context.Context, deviceUID, channelID, streamKey string) error
	PTZ(ctx context.Context, deviceUID, channelID, direction string, h, v, z int) error
}

// IPCClientFactory 按插件 endpoint 构造 RPC 客户端。
type IPCClientFactory interface {
	For(endpoint string) IPCClient
}
