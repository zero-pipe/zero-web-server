package eventsub

// Subscription 是插件设备的事件订阅/布防状态。
type Subscription struct {
	ID                 int      `json:"id"`
	DeviceID           int      `json:"deviceId"` // 插件设备库 ID
	PluginID           string   `json:"pluginId"`
	DeviceUID          string   `json:"deviceUid"`
	DeviceInternalCode string   `json:"deviceInternalCode"`
	EventTypes         []string `json:"eventTypes"`
	Armed              bool     `json:"armed"`
	Status             string   `json:"status"` // armed | disarmed | error
	LastError          string   `json:"lastError"`
	RemoteID           string   `json:"remoteId"` // 插件返回的 subscription_id
	LastArmedAt        int64    `json:"lastArmedAt"`
	CreateTime         string   `json:"createTime"`
	UpdateTime         string   `json:"updateTime"`
}

type Repository interface {
	Save(s *Subscription) error
	GetByDevice(deviceID int) (*Subscription, error)
	ListArmedByPlugin(pluginID string) ([]*Subscription, error)
	DeleteByDevice(deviceID int) error
}
