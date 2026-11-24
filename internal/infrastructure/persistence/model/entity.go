package model

// ===========================================================================
// A. 物模型：设备 / 接入绑定 / 通道 / 扩展 / 码流 / 能力
// ===========================================================================

// Device 设备（物理资产，稳定信息）。
type Device struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	InternalCode string `gorm:"column:internal_code;size:32;uniqueIndex:uk_device_internal_code" json:"internalCode"`
	GbCode       string `gorm:"column:gb_code;size:20;uniqueIndex:uk_device_gb_code" json:"gbCode"`
	Name         string `gorm:"column:name;size:255" json:"name"`
	Manufacturer string `gorm:"column:manufacturer;size:255" json:"manufacturer"`
	Model        string `gorm:"column:model;size:255" json:"model"`
	Firmware     string `gorm:"column:firmware;size:255" json:"firmware"`
	SerialNumber string `gorm:"column:serial_number;size:255" json:"serialNumber"`
	Online       bool   `gorm:"column:online" json:"online"`
	Enabled      bool   `gorm:"column:enabled;default:true" json:"enabled"`
	Status       string `gorm:"column:status;size:32" json:"status"`
	CreateTime   string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime   string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (Device) TableName() string { return "zws_device" }

// AccessBindingAgent 设备接入绑定(1:1)。
type AccessBinding struct {
	ID               int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DeviceID         int64  `gorm:"column:device_id;uniqueIndex:uk_binding_device" json:"deviceId"`
	Protocol         string `gorm:"column:protocol;size:16;uniqueIndex:uk_binding_external,priority:1" json:"protocol"`
	Mode             string `gorm:"column:mode;size:16" json:"mode"`
	ExternalID       string `gorm:"column:external_id;size:255;uniqueIndex:uk_binding_external,priority:2" json:"externalId"`
	Endpoint         string `gorm:"column:endpoint;size:255" json:"endpoint"`
	PluginInstanceID string `gorm:"column:plugin_instance_id;size:128" json:"pluginInstanceId"`
	ConfigJSON       string `gorm:"column:config_json;type:longtext" json:"configJson"`
	Status           string `gorm:"column:status;size:32" json:"status"`
	LastSeen         int64  `gorm:"column:last_seen" json:"lastSeen"`
	CreateTime       string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime       string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (AccessBinding) TableName() string { return "zws_access_binding" }

// Channel 通道（业务唯一事实来源）。
type Channel struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DeviceID     int64  `gorm:"column:device_id;index" json:"deviceId"`
	InternalCode string `gorm:"column:internal_code;size:32;uniqueIndex:uk_channel_internal_code" json:"internalCode"`
	GbCode       string `gorm:"column:gb_code;size:20;uniqueIndex:uk_channel_gb_code" json:"gbCode"`
	ExternalKey  string `gorm:"column:external_key;size:255" json:"externalKey"`
	Kind         string `gorm:"column:kind;size:32" json:"kind"`
	Name         string `gorm:"column:name;size:255" json:"name"`
	Online       bool   `gorm:"column:online" json:"online"`
	Status       string `gorm:"column:status;size:32" json:"status"`
	CreateTime   string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime   string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (Channel) TableName() string { return "zws_channel" }

// DeviceExt 设备扩展信息（易变）。
type DeviceExt struct {
	ID          int64   `gorm:"column:id;primaryKey;autoIncrement"`
	DeviceID    int64   `gorm:"column:device_id;uniqueIndex:uk_device_ext_device"`
	Longitude   float64 `gorm:"column:longitude"`
	Latitude    float64 `gorm:"column:latitude"`
	Address     string  `gorm:"column:address;size:255"`
	GeoCoordSys string  `gorm:"column:geo_coord_sys;size:50"`
	Remark      string  `gorm:"column:remark;size:255"`
	Tags        string  `gorm:"column:tags;size:255"`
	ExtraJSON   string  `gorm:"column:extra_json;type:longtext"`
	CreateTime  string  `gorm:"column:create_time;size:50"`
	UpdateTime  string  `gorm:"column:update_time;size:50"`
}

func (DeviceExt) TableName() string { return "zws_device_ext" }

// ChannelExt 通道扩展信息（易变：经纬度/PTZ/标签/录像计划）。
type ChannelExt struct {
	ID           int64   `gorm:"column:id;primaryKey;autoIncrement"`
	ChannelID    int64   `gorm:"column:channel_id;uniqueIndex:uk_channel_ext_channel"`
	Longitude    float64 `gorm:"column:longitude"`
	Latitude     float64 `gorm:"column:latitude"`
	MapLevel     int     `gorm:"column:map_level"`
	PTZType      int     `gorm:"column:ptz_type"`
	HasAudio     bool    `gorm:"column:has_audio"`
	Enabled      bool    `gorm:"column:enabled;default:true"`
	Tags         string  `gorm:"column:tags;size:255"`
	RecordPlanID int64   `gorm:"column:record_plan_id"`
	GBJSON       string  `gorm:"column:gb_json;type:longtext"`
	ExtraJSON    string  `gorm:"column:extra_json;type:longtext"`
	CreateTime   string  `gorm:"column:create_time;size:50"`
	UpdateTime   string  `gorm:"column:update_time;size:50"`
}

func (ChannelExt) TableName() string { return "zws_channel_ext" }

// StreamProfile 视频通道码流。
type StreamProfile struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ChannelID    int64  `gorm:"column:channel_id;uniqueIndex:uk_profile_channel_role,priority:1" json:"channelId"`
	Role         string `gorm:"column:role;size:16;uniqueIndex:uk_profile_channel_role,priority:2" json:"role"`
	Name         string `gorm:"column:name;size:128" json:"name"`
	SelectorJSON string `gorm:"column:selector_json;type:text" json:"selectorJson"`
	Codec        string `gorm:"column:codec;size:64" json:"codec"`
	Resolution   string `gorm:"column:resolution;size:64" json:"resolution"`
	IsDefault    bool   `gorm:"column:is_default" json:"isDefault"`
	CreateTime   string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime   string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (StreamProfile) TableName() string { return "zws_stream_profile" }

// Capability 通道能力集合（三态制约操作）。
type Capability struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement"`
	ChannelID  int64  `gorm:"column:channel_id;uniqueIndex:uk_cap_channel_op,priority:1"`
	Operation  string `gorm:"column:operation;size:64;uniqueIndex:uk_cap_channel_op,priority:2"`
	Supported  bool   `gorm:"column:supported"`
	Configured bool   `gorm:"column:configured"`
	Available  bool   `gorm:"column:available"`
	ParamsJSON string `gorm:"column:params_json;type:text"`
	Source     string `gorm:"column:source;size:32"`
	UpdateTime string `gorm:"column:update_time;size:50"`
}

func (Capability) TableName() string { return "zws_capability" }

// ===========================================================================
// B. 目录树：行政区划 / 业务分组 / 多挂载
// ===========================================================================

// Region 行政区划目录。
type Region struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ParentID   *int64 `gorm:"column:parent_id;index" json:"parentId"`
	Code       string `gorm:"column:code;size:20;uniqueIndex:uk_region_code" json:"code"`
	Name       string `gorm:"column:name;size:255" json:"name"`
	Alias      string `gorm:"column:alias;size:255" json:"alias"`
	Sort       int    `gorm:"column:sort" json:"sort"`
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (Region) TableName() string { return "zws_region" }

// Group 业务分组目录。
type Group struct {
	ID            int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ParentID      *int64 `gorm:"column:parent_id;index" json:"parentId"`
	Code          string `gorm:"column:code;size:20;uniqueIndex:uk_group_code" json:"code"`
	Name          string `gorm:"column:name;size:255" json:"name"`
	BusinessGroup string `gorm:"column:business_group;size:255" json:"businessGroup"`
	CivilCode     string `gorm:"column:civil_code;size:255" json:"civilCode"`
	Alias         string `gorm:"column:alias;size:255" json:"alias"`
	Sort          int    `gorm:"column:sort" json:"sort"`
	CreateTime    string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime    string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (Group) TableName() string { return "zws_group" }

// Catalog 目录挂载（多对多，可多挂）。
type Catalog struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RegionID   *int64 `gorm:"column:region_id;uniqueIndex:uk_catalog_region,priority:1" json:"regionId"`
	GroupID    *int64 `gorm:"column:group_id;uniqueIndex:uk_catalog_group,priority:1" json:"groupId"`
	TargetType string `gorm:"column:target_type;size:16;uniqueIndex:uk_catalog_region,priority:2;uniqueIndex:uk_catalog_group,priority:2" json:"targetType"`
	TargetID   int64  `gorm:"column:target_id;uniqueIndex:uk_catalog_region,priority:3;uniqueIndex:uk_catalog_group,priority:3;index:idx_catalog_target,priority:2" json:"targetId"`
	Sort       int    `gorm:"column:sort" json:"sort"`
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
}

func (Catalog) TableName() string { return "zws_catalog" }

// ===========================================================================
// C. 事件
// ===========================================================================

// EventType 统一事件类型注册表。
type EventType struct {
	Code           string `gorm:"column:code;primaryKey;size:64"`
	Name           string `gorm:"column:name;size:128"`
	Category       string `gorm:"column:category;size:32"`
	Severity       string `gorm:"column:severity;size:32"`
	AlarmEligible  bool   `gorm:"column:alarm_eligible"`
	RequiresVideo  bool   `gorm:"column:requires_video"`
	BBoxCapable    bool   `gorm:"column:bbox_capable"`
	VendorMappings string `gorm:"column:vendor_mappings;type:text"`
	Enabled        bool   `gorm:"column:enabled"`
	UpdateTime     string `gorm:"column:update_time;size:50"`
}

func (EventType) TableName() string { return "zws_event_type" }

// Event 平台事件记录（统一信封）。
type Event struct {
	ID                   int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	EventID              string `gorm:"column:event_id;size:64;uniqueIndex:uk_event_id" json:"eventId"`
	DeviceID             int64  `gorm:"column:device_id;index:idx_event_device_time,priority:1" json:"deviceId"`
	ChannelID            *int64 `gorm:"column:channel_id;index:idx_event_channel_time,priority:1" json:"channelId"`
	EventType            string `gorm:"column:event_type;size:64;index:idx_event_type_time,priority:1" json:"eventType"`
	Severity             string `gorm:"column:severity;size:16" json:"severity"`
	Status               string `gorm:"column:status;size:16" json:"status"`
	OccurredAt           int64  `gorm:"column:occurred_at;index:idx_event_device_time,priority:2;index:idx_event_channel_time,priority:2;index:idx_event_type_time,priority:2" json:"occurredAt"`
	ReceivedAt           int64  `gorm:"column:received_at" json:"receivedAt"`
	DeviceInternalCode   string `gorm:"column:device_internal_code;size:64" json:"deviceInternalCode"`
	ChannelInternalCode  string `gorm:"column:channel_internal_code;size:64" json:"channelInternalCode"`
	BBoxJSON             string `gorm:"column:bbox_json;type:text" json:"bboxJson"`
	ImageURL             string `gorm:"column:image_url;size:1024" json:"imageUrl"`
	RawJSON              string `gorm:"column:raw_json;type:longtext" json:"rawJson"`
	CreateTime           string `gorm:"column:create_time;size:50" json:"createTime"`
}

func (Event) TableName() string { return "zws_event" }

// Alarm 报警（处置投影 + 查询字段，1:1 事件）。
type Alarm struct {
	ID                  int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	EventID             string  `gorm:"column:event_id;size:64;uniqueIndex:uk_alarm_event" json:"eventId"`
	ChannelID           int64   `gorm:"column:channel_id" json:"channelId"`
	Description         string  `gorm:"column:description;size:255" json:"description"`
	AlarmType           int     `gorm:"column:alarm_type;index:idx_alarm_type" json:"alarmType"`
	AlarmCode           string  `gorm:"column:alarm_code;size:64" json:"alarmCode"`
	AlarmTime           int64   `gorm:"column:alarm_time;index:idx_alarm_time" json:"alarmTime"`
	DeviceInternalCode  string  `gorm:"column:device_internal_code;size:64" json:"deviceInternalCode"`
	ChannelInternalCode string  `gorm:"column:channel_internal_code;size:64" json:"channelInternalCode"`
	Longitude           float64 `gorm:"column:longitude" json:"longitude"`
	Latitude            float64 `gorm:"column:latitude" json:"latitude"`
	BBoxJSON            string  `gorm:"column:bbox_json;type:text" json:"bboxJson"`
	Status              string  `gorm:"column:status;size:32" json:"status"`
	Severity            string  `gorm:"column:severity;size:32" json:"severity"`
	HandedBy            string  `gorm:"column:handed_by;size:128" json:"handedBy"`
	HandledAt           int64   `gorm:"column:handled_at" json:"handledAt"`
	SnapPath            string  `gorm:"column:snap_path;size:255" json:"snapPath"`
	RecordPath          string  `gorm:"column:record_path;size:255" json:"recordPath"`
	Remark              string  `gorm:"column:remark;size:255" json:"remark"`
	CreateTime          string  `gorm:"column:create_time;size:50" json:"createTime"`
}

func (Alarm) TableName() string { return "zws_alarm" }

// EventSource 事件来源。
type EventSource struct {
	ID          int64  `gorm:"column:id;primaryKey;autoIncrement"`
	DeviceID    int64  `gorm:"column:device_id"`
	ChannelID   *int64 `gorm:"column:channel_id"`
	Kind        string `gorm:"column:kind;size:32"`
	ExternalKey string `gorm:"column:external_key;size:255"`
	Enabled     bool   `gorm:"column:enabled"`
	ConfigJSON  string `gorm:"column:config_json;type:text"`
}

func (EventSource) TableName() string { return "zws_event_source" }

// EventSubscription 事件订阅/布防。
type EventSubscription struct {
	ID                 int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DeviceID           int64  `gorm:"column:device_id;uniqueIndex:uk_event_sub,priority:1" json:"deviceId"`
	ChannelID          *int64 `gorm:"column:channel_id;uniqueIndex:uk_event_sub,priority:2" json:"channelId"`
	DeviceInternalCode string `gorm:"column:device_internal_code;size:64" json:"deviceInternalCode"`
	PluginInstanceID   string `gorm:"column:plugin_instance_id;size:128" json:"pluginInstanceId"`
	DeviceUID          string `gorm:"column:device_uid;size:255" json:"deviceUid"`
	EventTypesJSON     string `gorm:"column:event_types;type:text" json:"-"`
	Armed              bool   `gorm:"column:armed;not null;default:0" json:"armed"`
	Status             string `gorm:"column:status;size:16" json:"status"`
	LastError          string `gorm:"column:last_error;size:500" json:"lastError"`
	RemoteID           string `gorm:"column:remote_id;size:128" json:"remoteId"`
	LastArmedAt        int64  `gorm:"column:last_armed_at" json:"lastArmedAt"`
	CreateTime         string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime         string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (EventSubscription) TableName() string { return "zws_event_subscription" }

// CustomEvent 自定义事件（raw 原样保存，与统一事件隔离）。
type CustomEvent struct {
	ID                  int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	EventID             string `gorm:"column:event_id;size:64" json:"eventId"`
	DeviceInternalCode  string `gorm:"column:device_internal_code;size:64" json:"deviceInternalCode"`
	ChannelInternalCode string `gorm:"column:channel_internal_code;size:64" json:"channelInternalCode"`
	DeviceID            string `gorm:"column:device_id;size:64" json:"deviceId"`
	ResourceID          string `gorm:"column:resource_id;size:64" json:"resourceId"`
	MajorType           string `gorm:"column:major_type;size:64" json:"majorType"`
	MinorType           string `gorm:"column:minor_type;size:64" json:"minorType"`
	OccurAt             int64  `gorm:"column:occur_at" json:"occurAt"`
	Level               string `gorm:"column:level;size:32" json:"level"`
	ImageKey            string `gorm:"column:image_key;size:255" json:"imageKey"`
	ImageURL            string `gorm:"column:image_url;size:1024" json:"imageUrl"`
	RawJSON             string `gorm:"column:raw_json;type:longtext" json:"rawJson"`
	CreateTime          string `gorm:"column:create_time;size:50" json:"createTime"`
}

func (CustomEvent) TableName() string { return "zws_custom_event" }

// ===========================================================================
// D. 媒体 / 录像 / 位置
// ===========================================================================

// MediaServer 媒体服务器节点。
type MediaServer struct {
	ID                   string `gorm:"column:id;primaryKey;size:255" json:"id"`
	Name                 string `gorm:"column:name;size:128" json:"name"`
	IP                   string `gorm:"column:ip;size:50" json:"ip"`
	HookIP               string `gorm:"column:hook_ip;size:50" json:"hookIp"`
	SDPIP                string `gorm:"column:sdp_ip;size:50" json:"sdpIp"`
	StreamIP             string `gorm:"column:stream_ip;size:50" json:"streamIp"`
	HTTPPort             int    `gorm:"column:http_port" json:"httpPort"`
	HTTPSSLPort          int    `gorm:"column:http_ssl_port" json:"httpSSlPort"`
	RTMPPort             int    `gorm:"column:rtmp_port" json:"rtmpPort"`
	RTMPSSLPort          int    `gorm:"column:rtmp_ssl_port" json:"rtmpSslPort"`
	RTSPPort             int    `gorm:"column:rtsp_port" json:"rtspPort"`
	RTSPSSLPort          int    `gorm:"column:rtsp_ssl_port" json:"rtspSslPort"`
	SRTPort              int    `gorm:"column:srt_port" json:"srtPort"`
	RTPProxyPort         int    `gorm:"column:rtp_proxy_port" json:"rtpProxyPort"`
	WebRTCPortMin        int    `gorm:"column:webrtc_port_min" json:"webrtcPortMin"`
	WebRTCPortMax        int    `gorm:"column:webrtc_port_max" json:"webrtcPortMax"`
	CapabilitiesReported bool   `gorm:"column:capabilities_reported" json:"capabilitiesReported"`
	EnableRTSP           bool   `gorm:"column:enable_rtsp" json:"enableRtsp"`
	EnableRTMP           bool   `gorm:"column:enable_rtmp" json:"enableRtmp"`
	EnableHLS            bool   `gorm:"column:enable_hls" json:"enableHls"`
	EnableSRT            bool   `gorm:"column:enable_srt" json:"enableSrt"`
	EnableWebRTC         bool   `gorm:"column:enable_webrtc" json:"enableWebrtc"`
	Secret               string `gorm:"column:secret;size:50" json:"secret"`
	Type                 string `gorm:"column:type;size:32" json:"type"`
	Version              string `gorm:"column:version;size:50" json:"version"`
	ServerID             string `gorm:"column:server_id;size:50" json:"serverId"`
	DefaultServer        bool   `gorm:"column:default_server" json:"defaultServer"`
	HookAliveInterval    int    `gorm:"column:hook_alive_interval" json:"hookAliveInterval"`
	RecordPath           string `gorm:"column:record_path;size:255" json:"recordPath"`
	RecordDay            int    `gorm:"column:record_day" json:"recordDay"`
	TranscodeSuffix      string `gorm:"column:transcode_suffix;size:255" json:"transcodeSuffix"`
	AutoConfig           bool   `gorm:"column:auto_config" json:"autoConfig"`
	LastSeen             int64  `gorm:"column:last_seen" json:"lastSeen"`
	CreateTime           string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime           string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (MediaServer) TableName() string { return "zws_media_server" }

// RecordPlan 录像计划。
type RecordPlan struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name       string `gorm:"column:name;size:255" json:"name"`
	Snap       bool   `gorm:"column:snap" json:"snap"`
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (RecordPlan) TableName() string { return "zws_record_plan" }

// RecordPlanItem 录像计划时段。
type RecordPlanItem struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PlanID     int64  `gorm:"column:plan_id" json:"planId"`
	WeekDay    int    `gorm:"column:week_day" json:"weekDay"`
	Start      int    `gorm:"column:start_min" json:"start"`
	Stop       int    `gorm:"column:stop_min" json:"stop"`
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (RecordPlanItem) TableName() string { return "zws_record_plan_item" }

// CloudRecord 云端录像。
type CloudRecord struct {
	ID            int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ChannelID     *int64  `gorm:"column:channel_id;index:idx_cloud_channel_time,priority:1" json:"channelId"`
	App           string  `gorm:"column:app;size:64" json:"app"`
	Stream        string  `gorm:"column:stream;size:255" json:"stream"`
	CallID        string  `gorm:"column:call_id;size:255" json:"callId"`
	MediaServerID string  `gorm:"column:media_server_id;size:255" json:"mediaServerId"`
	StartTime     int64   `gorm:"column:start_time" json:"startTime"`
	EndTime       int64   `gorm:"column:end_time" json:"endTime"`
	FileName      string  `gorm:"column:file_name;size:255" json:"fileName"`
	Folder        string  `gorm:"column:folder;size:255" json:"folder"`
	FilePath      string  `gorm:"column:file_path;size:512" json:"filePath"`
	Collect       bool    `gorm:"column:collect" json:"collect"`
	FileSize      int64   `gorm:"column:file_size" json:"fileSize"`
	TimeLen       float64 `gorm:"column:time_len" json:"timeLen"`
	PlayURL       string  `gorm:"column:play_url;size:512" json:"playUrl"`
	ServerID      string  `gorm:"column:server_id;size:50" json:"serverId"`
}

func (CloudRecord) TableName() string { return "zws_cloud_record" }

// StreamProxy 拉流代理。
type StreamProxy struct {
	ID                      int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name                    string `gorm:"column:name;size:255" json:"name"`
	Type                    string `gorm:"column:type;size:50" json:"type"`
	App                     string `gorm:"column:app;size:64" json:"app"`
	Stream                  string `gorm:"column:stream;size:255" json:"stream"`
	SrcURL                  string `gorm:"column:src_url;size:1024" json:"srcUrl"`
	Timeout                 int    `gorm:"column:timeout" json:"timeout"`
	FFmpegCmdKey            string `gorm:"column:ffmpeg_cmd_key;size:255" json:"ffmpegCmdKey"`
	RTSPType                string `gorm:"column:rtsp_type;size:50" json:"rtspType"`
	MediaServerID           string `gorm:"column:media_server_id;size:255" json:"mediaServerId"`
	EnableAudio             bool   `gorm:"column:enable_audio" json:"enableAudio"`
	EnableMP4               bool   `gorm:"column:enable_mp4" json:"enableMp4"`
	Enable                  bool   `gorm:"column:enable" json:"enable"`
	Pulling                 bool   `gorm:"column:pulling" json:"pulling"`
	StreamKey               string `gorm:"column:stream_key;size:255" json:"streamKey"`
	ServerID                string `gorm:"column:server_id;size:50" json:"serverId"`
	EnableDisableNoneReader bool   `gorm:"column:enable_disable_none_reader" json:"enableDisableNoneReader"`
	RelatesMediaServerID    string `gorm:"column:relates_media_server_id;size:255" json:"relatesMediaServerId"`
	GbCode                  string `gorm:"column:gb_code;size:50" json:"gbCode"`
	GbName                  string `gorm:"column:gb_name;size:255" json:"gbName"`
	CreateTime              string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime              string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (StreamProxy) TableName() string { return "zws_stream_proxy" }

// StreamPush 推流会话。
type StreamPush struct {
	ID               int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	App              string `gorm:"column:app;size:64" json:"app"`
	Stream           string `gorm:"column:stream;size:255" json:"stream"`
	MediaServerID    string `gorm:"column:media_server_id;size:255" json:"mediaServerId"`
	Status           bool   `gorm:"column:status" json:"status"`
	Pushing          bool   `gorm:"column:pushing" json:"pushing"`
	Self             bool   `gorm:"column:self" json:"self"`
	PushTime         string `gorm:"column:push_time;size:50" json:"pushTime"`
	ServerID         string `gorm:"column:server_id;size:50" json:"serverId"`
	StartOfflinePush bool   `gorm:"column:start_offline_push" json:"startOfflinePush"`
	GbCode           string `gorm:"column:gb_code;size:50" json:"gbCode"`
	GbName           string `gorm:"column:gb_name;size:255" json:"gbName"`
	CreateTime       string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime       string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (StreamPush) TableName() string { return "zws_stream_push" }

// MobilePosition 移动位置上报。
type MobilePosition struct {
	ID         int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ChannelID  int64   `gorm:"column:channel_id;index:idx_pos_channel_time,priority:1" json:"channelId"`
	Timestamp  int64   `gorm:"column:timestamp" json:"timestamp"`
	Longitude  float64 `gorm:"column:longitude" json:"longitude"`
	Latitude   float64 `gorm:"column:latitude" json:"latitude"`
	Altitude   float64 `gorm:"column:altitude" json:"altitude"`
	Speed      float64 `gorm:"column:speed" json:"speed"`
	Direction  float64 `gorm:"column:direction" json:"direction"`
	CreateTime string  `gorm:"column:create_time;size:50" json:"createTime"`
}

func (MobilePosition) TableName() string { return "zws_mobile_position" }

// ===========================================================================
// E. 级联
// ===========================================================================

// UpperPlatform 上级国标平台。
type UpperPlatform struct {
	ID                  int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Enable              bool   `gorm:"column:enable" json:"enable"`
	Name                string `gorm:"column:name;size:255" json:"name"`
	ServerGBID          string `gorm:"column:server_gb_id;size:50;uniqueIndex:uk_upper_server_gb" json:"serverGbId"`
	ServerGBDomain      string `gorm:"column:server_gb_domain;size:50" json:"serverGbDomain"`
	ServerIP            string `gorm:"column:server_ip;size:50" json:"serverIp"`
	ServerPort          int    `gorm:"column:server_port" json:"serverPort"`
	DeviceGBID          string `gorm:"column:device_gb_id;size:50" json:"deviceGbId"`
	DeviceIP            string `gorm:"column:device_ip;size:50" json:"deviceIp"`
	DevicePort          string `gorm:"column:device_port;size:50" json:"devicePort"`
	Username            string `gorm:"column:username;size:255" json:"username"`
	Password            string `gorm:"column:password;size:255" json:"password"`
	Expires             string `gorm:"column:expires;size:50" json:"expires"`
	KeepTimeout         string `gorm:"column:keep_timeout;size:50" json:"keepTimeout"`
	Transport           string `gorm:"column:transport;size:50" json:"transport"`
	CivilCode           string `gorm:"column:civil_code;size:50" json:"civilCode"`
	Manufacturer        string `gorm:"column:manufacturer;size:255" json:"manufacturer"`
	Model               string `gorm:"column:model;size:255" json:"model"`
	Address             string `gorm:"column:address;size:255" json:"address"`
	CharacterSet        string `gorm:"column:character_set;size:50" json:"characterSet"`
	PTZ                 bool   `gorm:"column:ptz" json:"ptz"`
	RTCP                bool   `gorm:"column:rtcp" json:"rtcp"`
	Status              bool   `gorm:"column:status" json:"status"`
	CatalogGroup        int    `gorm:"column:catalog_group" json:"catalogGroup"`
	RegisterWay         int    `gorm:"column:register_way" json:"registerWay"`
	Secrecy             int    `gorm:"column:secrecy" json:"secrecy"`
	AsMessageChannel    bool   `gorm:"column:as_message_channel" json:"asMessageChannel"`
	CatalogWithPlatform int    `gorm:"column:catalog_with_platform" json:"catalogWithPlatform"`
	CatalogWithGroup    int    `gorm:"column:catalog_with_group" json:"catalogWithGroup"`
	CatalogWithRegion   int    `gorm:"column:catalog_with_region" json:"catalogWithRegion"`
	AutoPushChannel     bool   `gorm:"column:auto_push_channel" json:"autoPushChannel"`
	SendStreamIP        string `gorm:"column:send_stream_ip;size:50" json:"sendStreamIp"`
	ServerID            string `gorm:"column:server_id;size:50" json:"serverId"`
	CreateTime          string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime          string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (UpperPlatform) TableName() string { return "zws_upper_platform" }

// SubordinatePlatform 下级国标平台。
type SubordinatePlatform struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Enable       bool   `gorm:"column:enable;not null;default:1" json:"enable"`
	Name         string `gorm:"column:name;size:255" json:"name"`
	DeviceGBID   string `gorm:"column:device_gb_id;size:50;uniqueIndex:uk_sub_gb_id" json:"deviceGBId"`
	Password     string `gorm:"column:password;size:64" json:"password"`
	Transport    string `gorm:"column:transport;size:16;default:UDP" json:"transport"`
	Status       bool   `gorm:"column:status;not null;default:0" json:"status"`
	IP           string `gorm:"column:ip;size:64" json:"ip"`
	Port         int    `gorm:"column:port" json:"port"`
	HostAddress  string `gorm:"column:host_address;size:128" json:"hostAddress"`
	Expires      int    `gorm:"column:expires" json:"expires"`
	RegisterCall string `gorm:"column:register_call_id;size:128" json:"registerCallId"`
	ServerID     string `gorm:"column:server_id;size:50" json:"serverId"`
	CreateTime   string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime   string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (SubordinatePlatform) TableName() string { return "zws_subordinate_platform" }

// SubordinateCatalog 下级平台推送来的目录项。
type SubordinateCatalog struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PlatformID   int64  `gorm:"column:platform_id;not null;uniqueIndex:uk_sub_catalog,priority:1" json:"platformId"`
	DeviceID     string `gorm:"column:device_id;size:50;not null;uniqueIndex:uk_sub_catalog,priority:2" json:"deviceId"`
	Name         string `gorm:"column:name;size:255" json:"name"`
	ParentID     string `gorm:"column:parent_id;size:50;index:idx_sub_catalog_parent" json:"parentId"`
	TypeCode     string `gorm:"column:type_code;size:8" json:"typeCode"`
	ItemType     string `gorm:"column:item_type;size:16" json:"itemType"`
	Status       string `gorm:"column:status;size:16" json:"status"`
	CivilCode    string `gorm:"column:civil_code;size:16" json:"civilCode"`
	Manufacturer string `gorm:"column:manufacturer;size:128" json:"manufacturer"`
	Model        string `gorm:"column:model;size:128" json:"model"`
	Orphan       bool   `gorm:"column:orphan;not null;default:0" json:"orphan"`
	RawJSON      string `gorm:"column:raw_json;type:text" json:"rawJson"`
	CreateTime   string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime   string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (SubordinateCatalog) TableName() string { return "zws_subordinate_catalog" }

// PlatformCatalogSnapshot 向上级推送目录快照。
type PlatformCatalogSnapshot struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PlatformID int64  `gorm:"column:platform_id;not null;uniqueIndex:uk_plat_catalog_snap,priority:1" json:"platformId"`
	DeviceID   string `gorm:"column:device_id;size:50;not null;uniqueIndex:uk_plat_catalog_snap,priority:2" json:"deviceId"`
	Name       string `gorm:"column:name;size:255" json:"name"`
	ParentID   string `gorm:"column:parent_id;size:50" json:"parentId"`
	ItemType   string `gorm:"column:item_type;size:16" json:"itemType"`
	Status     string `gorm:"column:status;size:16" json:"status"`
	Hash       string `gorm:"column:hash;size:64" json:"hash"`
}

func (PlatformCatalogSnapshot) TableName() string { return "zws_platform_catalog_snapshot" }

// PlatformGroupShare 上级平台按业务分组共享。
type PlatformGroupShare struct {
	ID         int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PlatformID int64 `gorm:"column:platform_id;not null;uniqueIndex:uk_plat_group,priority:1" json:"platformId"`
	GroupID    int64 `gorm:"column:group_id;not null;uniqueIndex:uk_plat_group,priority:2" json:"groupId"`
}

func (PlatformGroupShare) TableName() string { return "zws_platform_group_share" }

// PlatformChannel 通道对外映射（级联 overrides）。
type PlatformChannel struct {
	ID            int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PlatformID    int64  `gorm:"column:platform_id;not null;uniqueIndex:uk_plat_channel,priority:1" json:"platformId"`
	ChannelID     int64  `gorm:"column:channel_id;not null;uniqueIndex:uk_plat_channel,priority:2" json:"channelId"`
	CustomGBCode  string `gorm:"column:custom_gb_code;size:50" json:"customGbCode"`
	OverridesJSON string `gorm:"column:overrides_json;type:longtext" json:"overridesJson"`
	CreateTime    string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime    string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (PlatformChannel) TableName() string { return "zws_platform_channel" }

// RegionCodeMapping 区划节点对外编码映射。
type RegionCodeMapping struct {
	RegionID   int64  `gorm:"column:region_id;primaryKey" json:"regionId"`
	GBCode     string `gorm:"column:gb_code;size:20" json:"gbCode"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (RegionCodeMapping) TableName() string { return "zws_region_code_mapping" }

// GBCodeSeq 国标码序号分配。
type GBCodeSeq struct {
	Prefix string `gorm:"column:prefix;primaryKey;size:14" json:"prefix"`
	Next   int64  `gorm:"column:next;not null;default:0" json:"next"`
}

func (GBCodeSeq) TableName() string { return "zws_gb_code_seq" }

// ===========================================================================
// F. 接入 / 插件
// ===========================================================================

// Agent 接入侧 zero-agent 实例。
type Agent struct {
	ID         string `gorm:"column:id;primaryKey;size:128"`
	Hostname   string `gorm:"column:hostname;size:255"`
	IP         string `gorm:"column:ip;size:64"`
	Version    string `gorm:"column:version;size:64"`
	Endpoint   string `gorm:"column:endpoint;size:255"`
	Status     string `gorm:"column:status;size:32"`
	LastSeen   int64  `gorm:"column:last_seen"`
	CreateTime string `gorm:"column:create_time;size:50"`
	UpdateTime string `gorm:"column:update_time;size:50"`
}

func (Agent) TableName() string { return "zws_agent" }

// Plugin 是 sidecar 插件实例注册信息。
type Plugin struct {
	ID               string `gorm:"column:id;primaryKey;size:128"`
	PluginID         string `gorm:"column:plugin_id;size:128"`
	Vendor           string `gorm:"column:vendor;size:128"`
	Version          string `gorm:"column:version;size:64"`
	Protocols        string `gorm:"column:protocols;type:text"`
	Capabilities     string `gorm:"column:capabilities;type:text"`
	StreamTransports string `gorm:"column:stream_transports;type:text"`
	ConfigSchema     string `gorm:"column:config_schema;type:longtext"`
	Endpoint         string `gorm:"column:endpoint;size:255"`
	Online           bool   `gorm:"column:online"`
	LastSeen         int64  `gorm:"column:last_seen"`
	CreateTime       string `gorm:"column:create_time;size:50"`
	UpdateTime       string `gorm:"column:update_time;size:50"`
}

func (Plugin) TableName() string { return "zws_plugin" }

// AgentPlugin 是 Agent 上报的插件清单与运行状态。
type AgentPlugin struct {
	ID               string `gorm:"column:id;primaryKey;size:160"`
	AgentID          string `gorm:"column:agent_id;size:128;index"`
	PluginID         string `gorm:"column:plugin_id;size:128"`
	Name             string `gorm:"column:name;size:255"`
	Vendor           string `gorm:"column:vendor;size:128"`
	Version          string `gorm:"column:version;size:64"`
	Protocols        string `gorm:"column:protocols;type:text"`
	Capabilities     string `gorm:"column:capabilities;type:text"`
	StreamTransports string `gorm:"column:stream_transports;type:text"`
	EventTypes       string `gorm:"column:event_types;type:text"`
	ConfigSchema     string `gorm:"column:config_schema;type:longtext"`
	Enabled          bool   `gorm:"column:enabled"`
	DesiredState     string `gorm:"column:desired_state;size:32"`
	ActualState      string `gorm:"column:actual_state;size:32"`
	Source           string `gorm:"column:source;size:32"`
	LastError        string `gorm:"column:last_error;type:text"`
	LastSeen         int64  `gorm:"column:last_seen"`
	CreateTime       string `gorm:"column:create_time;size:50"`
	UpdateTime       string `gorm:"column:update_time;size:50"`
}

func (AgentPlugin) TableName() string { return "zws_agent_plugin" }

// PluginPackage 是平台登记的插件安装包（.zwplugin）。
type PluginPackage struct {
	ID               string `gorm:"column:id;primaryKey;size:64"`
	PluginID         string `gorm:"column:plugin_id;size:64;uniqueIndex:uk_plugin_pkg_version,priority:1"`
	Version          string `gorm:"column:version;size:64;uniqueIndex:uk_plugin_pkg_version,priority:2"`
	Name             string `gorm:"column:name;size:255"`
	Vendor           string `gorm:"column:vendor;size:128"`
	ObjectKey        string `gorm:"column:object_key;size:512"`
	SHA256           string `gorm:"column:sha256;size:64"`
	SizeBytes        int64  `gorm:"column:size_bytes"`
	ManifestJSON     string `gorm:"column:manifest_json;type:longtext"`
	ConfigSchema     string `gorm:"column:config_schema;type:longtext"`
	Protocols        string `gorm:"column:protocols;type:text"`
	Capabilities     string `gorm:"column:capabilities;type:text"`
	StreamTransports string `gorm:"column:stream_transports;type:text"`
	EventTypes       string `gorm:"column:event_types;type:text"`
	Architectures    string `gorm:"column:architectures;type:text"`
	Status           string `gorm:"column:status;size:32"`
	ErrorMsg         string `gorm:"column:error_msg;type:text"`
	CreateTime       string `gorm:"column:create_time;size:50"`
	UpdateTime       string `gorm:"column:update_time;size:50"`
}

func (PluginPackage) TableName() string { return "zws_plugin_package" }

// PluginInstall 是插件包在某 Agent 上的安装任务/状态。
type PluginInstall struct {
	ID         string `gorm:"column:id;primaryKey;size:64"`
	AgentID    string `gorm:"column:agent_id;size:128;index"`
	PackageID  string `gorm:"column:package_id;size:64"`
	PluginID   string `gorm:"column:plugin_id;size:64"`
	Version    string `gorm:"column:version;size:64"`
	Status     string `gorm:"column:status;size:32"`
	LastError  string `gorm:"column:last_error;type:text"`
	CreateTime string `gorm:"column:create_time;size:50"`
	UpdateTime string `gorm:"column:update_time;size:50"`
}

func (PluginInstall) TableName() string { return "zws_plugin_install" }

// ===========================================================================
// G. 用户 / 权限
// ===========================================================================

type UserRole struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name       string `gorm:"column:name;size:50" json:"name"`
	Authority  string `gorm:"column:authority;size:512" json:"authority"`
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (UserRole) TableName() string { return "zws_user_role" }

type User struct {
	ID         int64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username   string   `gorm:"column:username;size:255;uniqueIndex:uk_user_username" json:"username"`
	Password   string   `gorm:"column:password;size:255" json:"password,omitempty"`
	RoleID     *int64   `gorm:"column:role_id" json:"roleId"`
	CreateTime string   `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string   `gorm:"column:update_time;size:50" json:"updateTime"`
	Role       UserRole `gorm:"foreignKey:RoleID;references:ID" json:"role"`
}

func (User) TableName() string { return "zws_user" }

type UserAPIKey struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID     *int64 `gorm:"column:user_id" json:"userId"`
	App        string `gorm:"column:app;size:255" json:"app"`
	APIKey     string `gorm:"column:api_key;type:text" json:"apiKey"`
	ExpiredAt  int64  `gorm:"column:expired_at" json:"expiredAt"`
	Remark     string `gorm:"column:remark;size:255" json:"remark"`
	Enable     bool   `gorm:"column:enable" json:"enable"`
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (UserAPIKey) TableName() string { return "zws_user_api_key" }

// AccessToken 对外接入密钥。
type AccessToken struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name       string `gorm:"column:name;size:128" json:"name"`
	Token      string `gorm:"column:token;size:128;uniqueIndex:uk_access_token_token" json:"token"`
	Enabled    bool   `gorm:"column:enabled" json:"enabled"`
	Remark     string `gorm:"column:remark;size:255" json:"remark"`
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (AccessToken) TableName() string { return "zws_access_token" }

// ===========================================================================
// H. 系统配置
// ===========================================================================

// GbSipConfig 本平台唯一国标 SIP 配置。
type GbSipConfig struct {
	ID                 int64  `gorm:"column:id;primaryKey" json:"id"`
	IP                 string `gorm:"column:ip;size:64;not null" json:"ip"`
	Port               int    `gorm:"column:port;not null" json:"port"`
	Domain             string `gorm:"column:domain;size:32;not null" json:"domain"`
	DeviceID           string `gorm:"column:device_id;size:32;not null" json:"deviceId"`
	Password           string `gorm:"column:password;size:64;not null" json:"password"`
	Alarm              bool   `gorm:"column:alarm;not null;default:0" json:"alarm"`
	RequirePreRegister bool   `gorm:"column:require_pre_register;not null;default:1" json:"requirePreRegister"`
	Transport          string `gorm:"column:transport;size:16;not null;default:UDP" json:"transport"`
	StreamMode         string `gorm:"column:stream_mode;size:16;not null;default:UDP" json:"streamMode"`
	CreateTime         string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime         string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (GbSipConfig) TableName() string { return "zws_gb_sip_config" }

// ObjectStoreConfig 对象存储对接配置。
type ObjectStoreConfig struct {
	ID         int64  `gorm:"column:id;primaryKey" json:"id"`
	Enabled    bool   `gorm:"column:enabled;not null;default:0" json:"enabled"`
	Provider   string `gorm:"column:provider;size:32;not null;default:noop" json:"provider"`
	Endpoint   string `gorm:"column:endpoint;size:255" json:"endpoint"`
	Region     string `gorm:"column:region;size:64" json:"region"`
	Bucket     string `gorm:"column:bucket;size:128" json:"bucket"`
	AccessKey  string `gorm:"column:access_key;size:128" json:"accessKey"`
	SecretKey  string `gorm:"column:secret_key;size:256" json:"secretKey"`
	UseSSL     bool   `gorm:"column:use_ssl;not null;default:0" json:"useSSL"`
	PathStyle  bool   `gorm:"column:path_style;not null;default:1" json:"pathStyle"`
	PublicBase string `gorm:"column:public_base;size:255" json:"publicBase"`
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (ObjectStoreConfig) TableName() string { return "zws_object_store_config" }

// MQTTConfig 消息队列对接配置。
type MQTTConfig struct {
	ID             int64  `gorm:"column:id;primaryKey" json:"id"`
	Enabled        bool   `gorm:"column:enabled;not null;default:0" json:"enabled"`
	Broker         string `gorm:"column:broker;size:255" json:"broker"`
	ClientIDPrefix string `gorm:"column:client_id_prefix;size:64" json:"clientIdPrefix"`
	Username       string `gorm:"column:username;size:128" json:"username"`
	Password       string `gorm:"column:password;size:256" json:"password"`
	TopicPrefix    string `gorm:"column:topic_prefix;size:128;default:zws/events" json:"topicPrefix"`
	QoS            int    `gorm:"column:qos;default:1" json:"qos"`
	KeepAliveSec   int    `gorm:"column:keepalive_sec;default:30" json:"keepAliveSec"`
	TLSEnabled     bool   `gorm:"column:tls_enabled;not null;default:0" json:"tlsEnabled"`
	CACert         string `gorm:"column:ca_cert;type:text" json:"caCert"`
	ClientCert     string `gorm:"column:client_cert;type:text" json:"clientCert"`
	ClientKey      string `gorm:"column:client_key;type:text" json:"clientKey"`
	CreateTime     string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime     string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (MQTTConfig) TableName() string { return "zws_mqtt_config" }

// ===========================================================================
// I. 审计
// ===========================================================================

// AuditLog 操作审计日志。
type AuditLog struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID     *int64 `gorm:"column:user_id;index:idx_audit_user_time,priority:1" json:"userId"`
	Username   string `gorm:"column:username;size:255" json:"username"`
	Action     string `gorm:"column:action;size:64" json:"action"`
	TargetType string `gorm:"column:target_type;size:32;index:idx_audit_target,priority:1" json:"targetType"`
	TargetID   string `gorm:"column:target_id;size:64;index:idx_audit_target,priority:2" json:"targetId"`
	Detail     string `gorm:"column:detail;type:longtext" json:"detail"`
	IP         string `gorm:"column:ip;size:64" json:"ip"`
	CreateTime string `gorm:"column:create_time;size:50;index:idx_audit_user_time,priority:2" json:"createTime"`
}

func (AuditLog) TableName() string { return "zws_audit_log" }
