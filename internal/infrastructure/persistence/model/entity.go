package model

type UserRole struct {
	ID         int    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name       string `gorm:"column:name;size:50" json:"name"`
	Authority  string `gorm:"column:authority;size:512" json:"authority"` // 菜单权限：* 或 JSON 数组
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (UserRole) TableName() string { return "zws_user_role" }

type User struct {
	ID         int      `gorm:"column:id;primaryKey;autoIncrement"`
	Username   string   `gorm:"column:username;size:255"`
	Password   string   `gorm:"column:password;size:255"`
	RoleID     int      `gorm:"column:role_id"`
	CreateTime string   `gorm:"column:create_time;size:50"`
	UpdateTime string   `gorm:"column:update_time;size:50"`
	PushKey    string   `gorm:"column:push_key;size:50"`
	Role       UserRole `gorm:"foreignKey:RoleID;references:ID"`
}

func (User) TableName() string { return "zws_user" }

type OnvifDevice struct {
	ID            int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Name          string `gorm:"column:name;size:255"`
	IP            string `gorm:"column:ip;size:64"`
	Port          int    `gorm:"column:port"`
	Username      string `gorm:"column:username;size:128"`
	Password      string `gorm:"column:password;size:128"`
	Manufacturer  string `gorm:"column:manufacturer;size:255"`
	Model         string `gorm:"column:model;size:255"`
	Firmware      string `gorm:"column:firmware;size:255"`
	SerialNumber  string `gorm:"column:serial_number;size:255"`
	HardwareID    string `gorm:"column:hardware_id;size:255"`
	DeviceURI     string `gorm:"column:device_uri;size:512"`
	MediaURI      string `gorm:"column:media_uri;size:512"`
	PTZURI        string `gorm:"column:ptz_uri;size:512"`
	OnLine        bool   `gorm:"column:on_line"`
	DiscoveryMode int    `gorm:"column:discovery_mode"`
	MediaServerID string `gorm:"column:media_server_id;size:255"`
	CustomName    string `gorm:"column:custom_name;size:255"`
	ServerID      string `gorm:"column:server_id;size:50"`
	CreateTime    string `gorm:"column:create_time;size:50"`
	UpdateTime    string `gorm:"column:update_time;size:50"`
}

func (OnvifDevice) TableName() string { return "zws_onvif_device" }

type OnvifChannel struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement"`
	DeviceID     int64  `gorm:"column:device_id"`
	ProfileToken string `gorm:"column:profile_token;size:255"`
	Name         string `gorm:"column:name;size:255"`
	VideoSource  string `gorm:"column:video_source;size:255"`
	EncoderToken string `gorm:"column:encoder_token;size:255"`
	Resolution   string `gorm:"column:resolution;size:64"`
	Codec        string `gorm:"column:codec;size:64"`
	HasAudio     bool   `gorm:"column:has_audio"`
	HasPTZ       bool   `gorm:"column:has_ptz"`
	StreamURI    string `gorm:"column:stream_uri;size:1024"`
	Status       string `gorm:"column:status;size:50"`
	CreateTime   string `gorm:"column:create_time;size:50"`
	UpdateTime   string `gorm:"column:update_time;size:50"`
}

func (OnvifChannel) TableName() string { return "zws_onvif_channel" }

type MediaServer struct {
	ID                string `gorm:"column:id;primaryKey;size:255" json:"id"`
	IP                string `gorm:"column:ip;size:50" json:"ip"`
	HookIP            string `gorm:"column:hook_ip;size:50" json:"hookIp"`
	SDPIP             string `gorm:"column:sdp_ip;size:50" json:"sdpIp"`
	StreamIP          string `gorm:"column:stream_ip;size:50" json:"streamIp"`
	HTTPPort          int    `gorm:"column:http_port" json:"httpPort"`
	HTTPSSLPort       int    `gorm:"column:http_ssl_port" json:"httpSSlPort"`
	RTMPPort          int    `gorm:"column:rtmp_port" json:"rtmpPort"`
	RTSPPort          int    `gorm:"column:rtsp_port" json:"rtspPort"`
	RTPProxyPort      int    `gorm:"column:rtp_proxy_port" json:"rtpProxyPort"`
	Secret            string `gorm:"column:secret;size:50" json:"secret"`
	Type              string `gorm:"column:type;size:50" json:"type"`
	DefaultServer     bool   `gorm:"column:default_server" json:"defaultServer"`
	HookAliveInterval int    `gorm:"column:hook_alive_interval" json:"hookAliveInterval"`
	CreateTime        string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime        string `gorm:"column:update_time;size:50" json:"updateTime"`
	ServerID          string `gorm:"column:server_id;size:50" json:"serverId"`
}

func (MediaServer) TableName() string { return "zws_media_server" }

type GBDevice struct {
	ID                               int    `gorm:"column:id;primaryKey;autoIncrement"`
	DeviceID                         string `gorm:"column:device_id;size:50"`
	Name                             string `gorm:"column:name;size:255"`
	Manufacturer                     string `gorm:"column:manufacturer;size:255"`
	Model                            string `gorm:"column:model;size:255"`
	Firmware                         string `gorm:"column:firmware;size:255"`
	Transport                        string `gorm:"column:transport;size:50"`
	StreamMode                       string `gorm:"column:stream_mode;size:50"`
	OnLine                           bool   `gorm:"column:on_line"`
	IP                               string `gorm:"column:ip;size:50"`
	Port                             int    `gorm:"column:port"`
	Expires                          int    `gorm:"column:expires"`
	HostAddress                      string `gorm:"column:host_address;size:50"`
	Charset                          string `gorm:"column:charset;size:50"`
	MediaServerID                    string `gorm:"column:media_server_id;size:50"`
	CustomName                       string `gorm:"column:custom_name;size:255"`
	SDPIP                            string `gorm:"column:sdp_ip;size:50"`
	LocalIP                          string `gorm:"column:local_ip;size:50"`
	Password                         string `gorm:"column:password;size:255"`
	HeartBeatInterval                int    `gorm:"column:heart_beat_interval"`
	HeartBeatCount                   int    `gorm:"column:heart_beat_count"`
	SubscribeCycleForCatalog         int    `gorm:"column:subscribe_cycle_for_catalog"`
	SubscribeCycleForMobilePosition  int    `gorm:"column:subscribe_cycle_for_mobile_position"`
	MobilePositionSubmissionInterval int    `gorm:"column:mobile_position_submission_interval"`
	SubscribeCycleForAlarm           int    `gorm:"column:subscribe_cycle_for_alarm"`
	ServerID                         string `gorm:"column:server_id;size:50"`
	CreateTime                       string `gorm:"column:create_time;size:50"`
	UpdateTime                       string `gorm:"column:update_time;size:50"`
}

func (GBDevice) TableName() string { return "zws_device" }

type GBDeviceChannel struct {
	ID                int     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DeviceID          string  `gorm:"column:device_id;size:50" json:"deviceId"`
	Name              string  `gorm:"column:name;size:255" json:"name"`
	Manufacturer      string  `gorm:"column:manufacturer;size:255" json:"manufacturer"`
	Model             string  `gorm:"column:model;size:255" json:"model"`
	Owner             string  `gorm:"column:owner;size:50" json:"owner"`
	CivilCode         string  `gorm:"column:civil_code;size:50" json:"civilCode"`
	Block             string  `gorm:"column:block;size:50" json:"block"`
	Address           string  `gorm:"column:address;size:50" json:"address"`
	Parental          int     `gorm:"column:parental" json:"parental"`
	ParentID          string  `gorm:"column:parent_id;size:50" json:"parentId"`
	Status            string  `gorm:"column:status;size:50" json:"status"`
	Longitude         float64 `gorm:"column:longitude" json:"longitude"`
	Latitude          float64 `gorm:"column:latitude" json:"latitude"`
	PTZType           int     `gorm:"column:ptz_type" json:"ptzType"`
	BusinessGroupID   string  `gorm:"column:business_group_id;size:255" json:"businessGroupId"`
	CreateTime        string  `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime        string  `gorm:"column:update_time;size:50" json:"updateTime"`
	SubCount          int     `gorm:"column:sub_count" json:"subCount"`
	HasAudio          bool    `gorm:"column:has_audio" json:"hasAudio"`
	ChannelType       int     `gorm:"column:channel_type;default:0" json:"channelType"`
	MapLevel          int     `gorm:"column:map_level;default:0" json:"mapLevel"`
	GBDeviceID        string  `gorm:"column:gb_device_id;size:50" json:"gbDeviceId"`
	GBName            string  `gorm:"column:gb_name;size:255" json:"gbName"`
	GBManufacturer    string  `gorm:"column:gb_manufacturer;size:255" json:"gbManufacturer"`
	GBModel           string  `gorm:"column:gb_model;size:255" json:"gbModel"`
	GBCivilCode       string  `gorm:"column:gb_civil_code;size:255" json:"gbCivilCode"`
	GBParentID        string  `gorm:"column:gb_parent_id;size:255" json:"gbParentId"`
	GBStatus          string  `gorm:"column:gb_status;size:50" json:"gbStatus"`
	GBLongitude       float64 `gorm:"column:gb_longitude" json:"gbLongitude"`
	GBLatitude        float64 `gorm:"column:gb_latitude" json:"gbLatitude"`
	GBBusinessGroupID string  `gorm:"column:gb_business_group_id;size:50" json:"gbBusinessGroupId"`
	GBPTZType         int     `gorm:"column:gb_ptz_type" json:"gbPtzType"`
	RecordPlanID      int     `gorm:"column:record_plan_id" json:"recordPlanId"`
	DataType          int     `gorm:"column:data_type" json:"dataType"`
	DataDeviceID      int     `gorm:"column:data_device_id" json:"dataDeviceId"`
}

func (GBDeviceChannel) TableName() string { return "zws_device_channel" }

type Alarm struct {
	ID          int     `gorm:"column:id;primaryKey;autoIncrement"`
	ChannelID   int     `gorm:"column:channel_id"`
	Description string  `gorm:"column:description;size:255"`
	SnapPath    string  `gorm:"column:snap_path;size:255"`
	RecordPath  string  `gorm:"column:record_path;size:255"`
	Longitude   float64 `gorm:"column:longitude"`
	Latitude    float64 `gorm:"column:latitude"`
	AlarmType   int     `gorm:"column:alarm_type"`
	AlarmTime   int64   `gorm:"column:alarm_time"`
}

func (Alarm) TableName() string { return "zws_alarm" }

type MobilePosition struct {
	ID         int     `gorm:"column:id;primaryKey;autoIncrement"`
	ChannelID  int     `gorm:"column:channel_id"`
	Timestamp  int64   `gorm:"column:timestamp"`
	Longitude  float64 `gorm:"column:longitude"`
	Latitude   float64 `gorm:"column:latitude"`
	Altitude   float64 `gorm:"column:altitude"`
	Speed      float64 `gorm:"column:speed"`
	Direction  float64 `gorm:"column:direction"`
	CreateTime string  `gorm:"column:create_time;size:50"`
}

func (MobilePosition) TableName() string { return "zws_mobile_position" }

type Platform struct {
	ID              int    `gorm:"column:id;primaryKey;autoIncrement"`
	Enable          bool   `gorm:"column:enable"`
	Name            string `gorm:"column:name;size:255"`
	ServerGBID      string `gorm:"column:server_gb_id;size:50"`
	ServerGBDomain  string `gorm:"column:server_gb_domain;size:50"`
	ServerIP        string `gorm:"column:server_ip;size:50"`
	ServerPort      int    `gorm:"column:server_port"`
	DeviceGBID      string `gorm:"column:device_gb_id;size:50"`
	DeviceIP        string `gorm:"column:device_ip;size:50"`
	DevicePort      string `gorm:"column:device_port;size:50"`
	Username        string `gorm:"column:username;size:255"`
	Password        string `gorm:"column:password;size:255"`
	Expires         string `gorm:"column:expires;size:50"`
	KeepTimeout     string `gorm:"column:keep_timeout;size:50"`
	Transport       string `gorm:"column:transport;size:50"`
	Status          bool   `gorm:"column:status"`
	AutoPushChannel bool   `gorm:"column:auto_push_channel"`
	ServerID        string `gorm:"column:server_id;size:50"`
	CreateTime      string `gorm:"column:create_time;size:50"`
	UpdateTime      string `gorm:"column:update_time;size:50"`
}

func (Platform) TableName() string { return "zws_platform" }

type PlatformChannel struct {
	ID              int    `gorm:"column:id;primaryKey;autoIncrement"`
	PlatformID      int    `gorm:"column:platform_id"`
	DeviceChannelID int    `gorm:"column:device_channel_id"`
	CustomDeviceID  string `gorm:"column:custom_device_id;size:50"`
	CustomName      string `gorm:"column:custom_name;size:255"`
}

func (PlatformChannel) TableName() string { return "zws_platform_channel" }

type StreamPush struct {
	ID               int    `gorm:"column:id;primaryKey;autoIncrement"`
	App              string `gorm:"column:app;size:255"`
	Stream           string `gorm:"column:stream;size:255"`
	MediaServerID    string `gorm:"column:media_server_id;size:50"`
	ServerID         string `gorm:"column:server_id;size:50"`
	PushTime         string `gorm:"column:push_time;size:50"`
	Status           bool   `gorm:"column:status"`
	UpdateTime       string `gorm:"column:update_time;size:50"`
	CreateTime       string `gorm:"column:create_time;size:50"`
	Pushing          bool   `gorm:"column:pushing"`
	StartOfflinePush bool   `gorm:"column:start_offline_push"`
}

func (StreamPush) TableName() string { return "zws_stream_push" }

type StreamProxy struct {
	ID                      int    `gorm:"column:id;primaryKey;autoIncrement"`
	Type                    string `gorm:"column:type;size:50"`
	App                     string `gorm:"column:app;size:255"`
	Stream                  string `gorm:"column:stream;size:255"`
	SrcURL                  string `gorm:"column:src_url;size:1024"`
	Timeout                 int    `gorm:"column:timeout"`
	FFmpegCmdKey            string `gorm:"column:ffmpeg_cmd_key;size:255"`
	RTSPType                string `gorm:"column:rtsp_type;size:50"`
	MediaServerID           string `gorm:"column:media_server_id;size:50"`
	EnableAudio             bool   `gorm:"column:enable_audio"`
	EnableMP4               bool   `gorm:"column:enable_mp4"`
	Pulling                 bool   `gorm:"column:pulling"`
	Enable                  bool   `gorm:"column:enable"`
	CreateTime              string `gorm:"column:create_time;size:50"`
	Name                    string `gorm:"column:name;size:255"`
	UpdateTime              string `gorm:"column:update_time;size:50"`
	StreamKey               string `gorm:"column:stream_key;size:255"`
	ServerID                string `gorm:"column:server_id;size:50"`
	EnableDisableNoneReader bool   `gorm:"column:enable_disable_none_reader"`
	RelatesMediaServerID    string `gorm:"column:relates_media_server_id;size:50"`
}

func (StreamProxy) TableName() string { return "zws_stream_proxy" }

type CloudRecord struct {
	ID            int     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	App           string  `gorm:"column:app;size:255" json:"app"`
	Stream        string  `gorm:"column:stream;size:255" json:"stream"`
	CallID        string  `gorm:"column:call_id;size:255" json:"callId"`
	StartTime     int64   `gorm:"column:start_time" json:"startTime"`
	EndTime       int64   `gorm:"column:end_time" json:"endTime"`
	MediaServerID string  `gorm:"column:media_server_id;size:50" json:"mediaServerId"`
	ServerID      string  `gorm:"column:server_id;size:50" json:"serverId"`
	FileName      string  `gorm:"column:file_name;size:255" json:"fileName"`
	Folder        string  `gorm:"column:folder;size:255" json:"folder"`
	FilePath      string  `gorm:"column:file_path;size:512" json:"filePath"`
	PlayURL       string  `gorm:"column:play_url;size:768" json:"playUrl"`
	Collect       bool    `gorm:"column:collect" json:"collect"`
	FileSize      int64   `gorm:"column:file_size" json:"fileSize"`
	TimeLen       float64 `gorm:"column:time_len" json:"timeLen"`
}

func (CloudRecord) TableName() string { return "zws_cloud_record" }

type RecordPlan struct {
	ID         int    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Snap       bool   `gorm:"column:snap" json:"snap"`
	Name       string `gorm:"column:name;size:255" json:"name"`
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (RecordPlan) TableName() string { return "zws_record_plan" }

type RecordPlanItem struct {
	ID         int    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Start      int    `gorm:"column:start" json:"start"`
	Stop       int    `gorm:"column:stop" json:"stop"`
	WeekDay    int    `gorm:"column:week_day" json:"weekDay"`
	PlanID     int    `gorm:"column:plan_id" json:"planId"`
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (RecordPlanItem) TableName() string { return "zws_record_plan_item" }

// CommonGroup 业务分组（组织树）。
type CommonGroup struct {
	ID             int    `gorm:"column:id;primaryKey;autoIncrement"`
	DeviceID       string `gorm:"column:device_id;size:50;not null"`
	Name           string `gorm:"column:name;size:255;not null"`
	ParentID       *int   `gorm:"column:parent_id"`
	ParentDeviceID string `gorm:"column:parent_device_id;size:50"`
	BusinessGroup  string `gorm:"column:business_group;size:50;not null"`
	CreateTime     string `gorm:"column:create_time;size:50;not null"`
	UpdateTime     string `gorm:"column:update_time;size:50;not null"`
	CivilCode      string `gorm:"column:civil_code;size:50"`
	Alias          string `gorm:"column:alias;size:255"`
}

func (CommonGroup) TableName() string { return "zws_common_group" }

// CommonRegion 行政区划。
type CommonRegion struct {
	ID             int    `gorm:"column:id;primaryKey;autoIncrement"`
	DeviceID       string `gorm:"column:device_id;size:50;not null"`
	Name           string `gorm:"column:name;size:255;not null"`
	ParentID       *int   `gorm:"column:parent_id"`
	ParentDeviceID string `gorm:"column:parent_device_id;size:50"`
	CreateTime     string `gorm:"column:create_time;size:50;not null"`
	UpdateTime     string `gorm:"column:update_time;size:50;not null"`
}

func (CommonRegion) TableName() string { return "zws_common_region" }

// GbSipConfig 本平台唯一国标 SIP 配置（设备接入 / 上下级级联共用）。
type GbSipConfig struct {
	ID         int    `gorm:"column:id;primaryKey" json:"id"`
	IP         string `gorm:"column:ip;size:64;not null" json:"ip"`
	Port       int    `gorm:"column:port;not null" json:"port"`
	Domain     string `gorm:"column:domain;size:32;not null" json:"domain"`
	DeviceID   string `gorm:"column:device_id;size:32;not null" json:"deviceId"`
	Password   string `gorm:"column:password;size:64;not null" json:"password"`
	Alarm      bool   `gorm:"column:alarm;not null;default:0" json:"alarm"`
	CreateTime string `gorm:"column:create_time;size:50" json:"createTime"`
	UpdateTime string `gorm:"column:update_time;size:50" json:"updateTime"`
}

func (GbSipConfig) TableName() string { return "zws_gb_sip_config" }
