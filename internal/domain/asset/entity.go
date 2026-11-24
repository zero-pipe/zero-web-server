package asset

// PhysicalDevice 是对用户可见的物理设备资产。
type PhysicalDevice struct {
	ID           string `json:"id"`
	InternalCode string `json:"internalCode"`
	Name         string `json:"name"`
	Vendor       string `json:"vendor"`
	Model        string `json:"model"`
	SerialNumber string `json:"serialNumber"`
	Enabled      bool   `json:"enabled"`
	Online       bool   `json:"online"`
	CreateTime   string `json:"createTime"`
	UpdateTime   string `json:"updateTime"`
}

// AccessBinding 是物理设备唯一的接入方式（本次设计严格 1:1）。
type AccessBinding struct {
	ID               string `json:"id"`
	DeviceID         string `json:"deviceId"`
	Protocol         string `json:"protocol"` // gb28181 | onvif | plugin
	Mode             string `json:"mode"`     // passive | active
	ExternalID       string `json:"externalId"`
	Endpoint         string `json:"endpoint"`
	PluginInstanceID string `json:"pluginInstanceId"`
	ConfigJSON       string `json:"configJson"`
	Status           string `json:"status"`
	LastSeen         int64  `json:"lastSeen"`
	LegacyDeviceID   int    `json:"-"`
	CreateTime       string `json:"createTime"`
	UpdateTime       string `json:"updateTime"`
}

// 资源类型。
const (
	ResourceVideoSensor   = "video_sensor"
	ResourceAudioInput    = "audio_input"
	ResourceAudioOutput   = "audio_output"
	ResourceAlarmInput    = "alarm_input"
	ResourceRelayOutput   = "relay_output"
	ResourceMetadataPoint = "metadata_source"
)

// Resource 是设备提供的稳定子资源（视频源、音频、告警输入、AI 元数据等）。
type Resource struct {
	ID              string  `json:"id"`
	InternalCode    string  `json:"internalCode"`
	DeviceID        string  `json:"deviceId"`
	BindingID       string  `json:"bindingId"`
	Kind            string  `json:"kind"`
	ExternalKey     string  `json:"externalKey"`
	Name            string  `json:"name"`
	Status          string  `json:"status"`
	LegacyChannelID int     `json:"-"`
	AttrsJSON       string  `json:"attrsJson"`
	Longitude       float64 `json:"longitude"`
	Latitude        float64 `json:"latitude"`
	MapLevel        int     `json:"mapLevel"`
	PTZType         int     `json:"ptzType"`
	HasAudio        bool    `json:"hasAudio"`
	Tags            string  `json:"tags"`
	Enabled         bool    `json:"enabled"`
	CreateTime      string  `json:"createTime"`
	UpdateTime      string  `json:"updateTime"`
}

// 码流角色。
const (
	ProfileMain   = "main"
	ProfileSub    = "sub"
	ProfileThird  = "third"
	ProfileCustom = "custom"
)

// StreamProfile 是视频资源的一个码流配置。
type StreamProfile struct {
	ID           string `json:"id"`
	ResourceID   string `json:"resourceId"`
	Role         string `json:"role"`
	Name         string `json:"name"`
	SelectorJSON string `json:"selectorJson"`
	Codec        string `json:"codec"`
	Resolution   string `json:"resolution"`
	IsDefault    bool   `json:"isDefault"`
	CreateTime   string `json:"createTime"`
	UpdateTime   string `json:"updateTime"`
}

// Capability 是某作用域上的能力事实。
type Capability struct {
	ID         string `json:"id"`
	ScopeType  string `json:"scopeType"` // device | resource | profile
	ScopeID    string `json:"scopeId"`
	Operation  string `json:"operation"`
	Supported  bool   `json:"supported"`
	Configured bool   `json:"configured"`
	Available  bool   `json:"available"`
	ParamsJSON string `json:"paramsJson"`
	Source     string `json:"source"`
	UpdateTime string `json:"updateTime"`
}

// EventSource 是事件/数据来源（与告警记录分离）。
type EventSource struct {
	ID          string `json:"id"`
	DeviceID    string `json:"deviceId"`
	ResourceID  string `json:"resourceId"`
	Kind        string `json:"kind"`
	ExternalKey string `json:"externalKey"`
	Enabled     bool   `json:"enabled"`
	ConfigJSON  string `json:"configJson"`
}

// DeviceSummary 用于列表展示。
type DeviceSummary struct {
	Device        *PhysicalDevice `json:"device"`
	Binding       *AccessBinding  `json:"binding"`
	ResourceCount int             `json:"resourceCount"`
}

// DeviceDetail 用于详情展示。
type DeviceDetail struct {
	Device       *PhysicalDevice `json:"device"`
	Binding      *AccessBinding  `json:"binding"`
	Resources    []*Resource     `json:"resources"`
	Capabilities []*Capability   `json:"capabilities"`
	EventSources []*EventSource  `json:"eventSources"`
}

// DevicePatch 是设备可编辑字段（nil 表示不修改）。
type DevicePatch struct {
	Name         *string `json:"name"`
	Vendor       *string `json:"vendor"`
	Model        *string `json:"model"`
	SerialNumber *string `json:"serialNumber"`
	Enabled      *bool   `json:"enabled"`
}

// ResourcePatch 是通道可编辑字段（nil 表示不修改）。
type ResourcePatch struct {
	Name      *string  `json:"name"`
	Longitude *float64 `json:"longitude"`
	Latitude  *float64 `json:"latitude"`
	MapLevel  *int     `json:"mapLevel"`
	PTZType   *int     `json:"ptzType"`
	HasAudio  *bool    `json:"hasAudio"`
	Tags      *string  `json:"tags"`
	Enabled   *bool    `json:"enabled"`
	// GBCode 是通道国标编码。国标通道该值由外部编码（externalKey）给出；
	// ONVIF / 插件通道没有原生国标码，可在此手工指定，落 attrs_json["gbCode"]。
	GBCode *string `json:"gbCode"`
	// GBFields 承载国标属性表单（行政区域、父节点、监视方位等）的其余字段，
	// 非国标通道没有 zws_device_channel 行，统一按键值存入 attrs_json["gb"]。
	GBFields map[string]any `json:"gb"`
}

// DeviceNode 是设备→通道树节点。
type DeviceNode struct {
	Device    *PhysicalDevice `json:"device"`
	Binding   *AccessBinding  `json:"binding"`
	Resources []*ResourceNode `json:"resources"`
}

// ResourceNode 是树中的通道/资源节点。
type ResourceNode struct {
	Resource     *Resource        `json:"resource"`
	Profiles     []*StreamProfile `json:"profiles"`
	Capabilities []*Capability    `json:"capabilities"`
}

// ChannelView 是跨协议统一的通道列表视图（通道内码为主键）。
type ChannelView struct {
	ResourceID         string  `json:"resourceId"`
	InternalCode       string  `json:"internalCode"`
	DeviceID           string  `json:"deviceId"`
	DeviceInternalCode string  `json:"deviceInternalCode"`
	DeviceName         string  `json:"deviceName"`
	Protocol           string  `json:"protocol"`
	Name               string  `json:"name"`
	Kind               string  `json:"kind"`
	ExternalKey        string  `json:"externalKey"` // GB 通道编码 / ONVIF ProfileToken / 插件通道号
	GBCode             string  `json:"gbCode"`
	Status             string  `json:"status"`
	Online             bool    `json:"online"`
	PTZType            int     `json:"ptzType"`
	HasAudio           bool    `json:"hasAudio"`
	Longitude          float64 `json:"longitude"`
	Latitude           float64 `json:"latitude"`
	MapLevel           int     `json:"mapLevel"`
	Tags               string  `json:"tags"`
	Enabled            bool    `json:"enabled"`
	// Caps 该通道真实可用的操作能力（驱动实现 ∩ 通道事实），前端能力集列据此点亮。
	Caps []string `json:"caps"`
	// GB 是非国标通道手工填写的国标属性（国标通道走 zws_device_channel，不在这里）。
	GB map[string]any `json:"gb,omitempty"`
}

// ResourceDetail 包含资源、码流与能力。
type ResourceDetail struct {
	Resource     *Resource        `json:"resource"`
	Profiles     []*StreamProfile `json:"profiles"`
	Capabilities []*Capability    `json:"capabilities"`
}

type Repository interface {
	CreateDevice(d *PhysicalDevice) error
	UpsertDevice(d *PhysicalDevice) error
	GetDevice(id string) (*PhysicalDevice, error)
	GetDeviceByInternalCode(code string) (*PhysicalDevice, error)
	ListDevices(page, count int, query, protocol, status string) ([]*DeviceSummary, int64, error)
	UpdateDeviceFields(id string, patch DevicePatch) error

	CreateBinding(b *AccessBinding) error
	UpsertBinding(b *AccessBinding) error
	GetBindingByDevice(deviceID string) (*AccessBinding, error)

	UpsertResource(r *Resource) error
	ListResources(deviceID string) ([]*Resource, error)
	ListChannels(page, count int, query, protocol, deviceInternalCode string) ([]*ChannelView, int64, error)
	GetResource(id string) (*Resource, error)
	GetResourceByInternalCode(code string) (*Resource, error)
	UpdateResourceFields(id string, patch ResourcePatch) error

	UpsertProfile(p *StreamProfile) error
	ListProfiles(resourceID string) ([]*StreamProfile, error)
	GetProfile(id string) (*StreamProfile, error)

	ReplaceCapabilities(scopeType, scopeID string, caps []*Capability) error
	ListCapabilities(scopeType, scopeID string) ([]*Capability, error)
	// SetCapabilityAvailable 运行时降级/恢复某一能力（首次调用失败即熔断）。
	SetCapabilityAvailable(scopeType, scopeID, operation string, available bool) error

	UpsertEventSource(e *EventSource) error
	ListEventSources(deviceID string) ([]*EventSource, error)

	GetLegacyRef(table string, legacyID string) (entityType, unifiedID string, err error)
	UpsertLegacyRef(table, legacyID, entityType, unifiedID string) error
}
