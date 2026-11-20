package plugin

// Plugin 是向平台注册的 sidecar 插件实例。
type Plugin struct {
	ID               string   `json:"id"` // 插件实例 ID（Hello 上报）
	PluginID         string   `json:"pluginId"`
	Vendor           string   `json:"vendor"`
	Version          string   `json:"version"`
	Protocols        []string `json:"protocols"`
	Capabilities     []string `json:"capabilities"`
	StreamTransports []string `json:"streamTransports"`
	ConfigSchema     string   `json:"configSchema"` // JSON Schema 文本
	Endpoint         string   `json:"endpoint"`
	Online           bool     `json:"online"`
	LastSeen         int64    `json:"lastSeen"`
	CreateTime       string   `json:"createTime"`
	UpdateTime       string   `json:"updateTime"`
}

// Device 是通过插件接入的设备。
type Device struct {
	ID           int    `json:"id"`
	InternalCode string `json:"internalCode"`
	PluginID     string `json:"pluginId"`
	Protocol     string `json:"protocol"`
	DeviceUID    string `json:"deviceUid"`
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	Online       bool   `json:"online"`
	DriverParams string `json:"driverParams"` // 厂商字段 JSON
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Firmware     string `json:"firmware"`
	SerialNumber string `json:"serialNumber"`
	ChannelCount int    `json:"channelCount"`
	CreateTime   string `json:"createTime"`
	UpdateTime   string `json:"updateTime"`
}

// Channel 是插件设备的通道。
type Channel struct {
	ID           int    `json:"id"`
	InternalCode string `json:"internalCode"`
	DeviceDBID   int    `json:"deviceDbId"`
	ChannelID    string `json:"channelId"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	PTZType      int    `json:"ptzType"`
	HasAudio     bool   `json:"hasAudio"`
	CreateTime   string `json:"createTime"`
	UpdateTime   string `json:"updateTime"`
}

type Repository interface {
	UpsertPlugin(p *Plugin) error
	UpdatePluginSeen(id string, ts int64, online bool) error
	MarkPluginOffline(id string) error
	ListPlugins() ([]*Plugin, error)
	GetPlugin(id string) (*Plugin, error)
	ResolvePlugin(protocol string) (*Plugin, error)

	CreateDevice(d *Device) error
	UpdateDevice(d *Device) error
	DeleteDevice(id int) error
	ListDevices() ([]*Device, error)
	GetDevice(id int) (*Device, error)
	GetDeviceByUID(pluginID, deviceUID string) (*Device, error)
	CountDevicesByPlugin(pluginID string) (int, error)

	ReplaceChannels(deviceDBID int, channels []*Channel) error
	ListChannels(deviceDBID int) ([]*Channel, error)
	GetChannel(deviceDBID int, channelID string) (*Channel, error)
}
