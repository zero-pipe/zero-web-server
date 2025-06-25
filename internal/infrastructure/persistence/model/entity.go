package model

type UserRole struct {
	ID         int    `gorm:"column:id;primaryKey"`
	Name       string `gorm:"column:name"`
	Authority  string `gorm:"column:authority"`
	CreateTime string `gorm:"column:create_time"`
	UpdateTime string `gorm:"column:update_time"`
}

func (UserRole) TableName() string { return "wvp_user_role" }

type User struct {
	ID         int    `gorm:"column:id;primaryKey"`
	Username   string `gorm:"column:username"`
	Password   string `gorm:"column:password"`
	RoleID     int    `gorm:"column:role_id"`
	CreateTime string `gorm:"column:create_time"`
	UpdateTime string `gorm:"column:update_time"`
	PushKey    string `gorm:"column:push_key"`
	Role       UserRole `gorm:"foreignKey:RoleID;references:ID"`
}

func (User) TableName() string { return "wvp_user" }

type OnvifDevice struct {
	ID            int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Name          string `gorm:"column:name"`
	IP            string `gorm:"column:ip"`
	Port          int    `gorm:"column:port"`
	Username      string `gorm:"column:username"`
	Password      string `gorm:"column:password"`
	Manufacturer  string `gorm:"column:manufacturer"`
	Model         string `gorm:"column:model"`
	Firmware      string `gorm:"column:firmware"`
	SerialNumber  string `gorm:"column:serial_number"`
	HardwareID    string `gorm:"column:hardware_id"`
	DeviceURI     string `gorm:"column:device_uri"`
	MediaURI      string `gorm:"column:media_uri"`
	PTZURI        string `gorm:"column:ptz_uri"`
	OnLine        bool   `gorm:"column:on_line"`
	DiscoveryMode int    `gorm:"column:discovery_mode"`
	MediaServerID string `gorm:"column:media_server_id"`
	CustomName    string `gorm:"column:custom_name"`
	ServerID      string `gorm:"column:server_id"`
	CreateTime    string `gorm:"column:create_time"`
	UpdateTime    string `gorm:"column:update_time"`
}

func (OnvifDevice) TableName() string { return "wvp_onvif_device" }

type OnvifChannel struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement"`
	DeviceID     int64  `gorm:"column:device_id"`
	ProfileToken string `gorm:"column:profile_token"`
	Name         string `gorm:"column:name"`
	VideoSource  string `gorm:"column:video_source"`
	EncoderToken string `gorm:"column:encoder_token"`
	Resolution   string `gorm:"column:resolution"`
	Codec        string `gorm:"column:codec"`
	HasAudio     bool   `gorm:"column:has_audio"`
	HasPTZ       bool   `gorm:"column:has_ptz"`
	StreamURI    string `gorm:"column:stream_uri"`
	Status       string `gorm:"column:status"`
	CreateTime   string `gorm:"column:create_time"`
	UpdateTime   string `gorm:"column:update_time"`
}

func (OnvifChannel) TableName() string { return "wvp_onvif_channel" }

type MediaServer struct {
	ID                string `gorm:"column:id;primaryKey"`
	IP                string `gorm:"column:ip"`
	HookIP            string `gorm:"column:hook_ip"`
	SDPIP             string `gorm:"column:sdp_ip"`
	StreamIP          string `gorm:"column:stream_ip"`
	HTTPPort          int    `gorm:"column:http_port"`
	HTTPSSLPort       int    `gorm:"column:http_ssl_port"`
	RTMPPort          int    `gorm:"column:rtmp_port"`
	RTSPPort          int    `gorm:"column:rtsp_port"`
	RTPProxyPort      int    `gorm:"column:rtp_proxy_port"`
	Secret            string `gorm:"column:secret"`
	Type              string `gorm:"column:type"`
	DefaultServer     bool   `gorm:"column:default_server"`
	HookAliveInterval int    `gorm:"column:hook_alive_interval"`
	CreateTime        string `gorm:"column:create_time"`
	UpdateTime        string `gorm:"column:update_time"`
	ServerID          string `gorm:"column:server_id"`
}

func (MediaServer) TableName() string { return "wvp_media_server" }

type GBDevice struct {
	ID                       int    `gorm:"column:id;primaryKey;autoIncrement"`
	DeviceID                 string `gorm:"column:device_id"`
	Name                     string `gorm:"column:name"`
	Manufacturer             string `gorm:"column:manufacturer"`
	Model                    string `gorm:"column:model"`
	Firmware                 string `gorm:"column:firmware"`
	Transport                string `gorm:"column:transport"`
	StreamMode               string `gorm:"column:stream_mode"`
	OnLine                   bool   `gorm:"column:on_line"`
	IP                       string `gorm:"column:ip"`
	Port                     int    `gorm:"column:port"`
	Expires                  int    `gorm:"column:expires"`
	HostAddress              string `gorm:"column:host_address"`
	Charset                  string `gorm:"column:charset"`
	MediaServerID            string `gorm:"column:media_server_id"`
	CustomName               string `gorm:"column:custom_name"`
	SDPIP                    string `gorm:"column:sdp_ip"`
	LocalIP                  string `gorm:"column:local_ip"`
	Password                 string `gorm:"column:password"`
	HeartBeatInterval        int    `gorm:"column:heart_beat_interval"`
	HeartBeatCount           int    `gorm:"column:heart_beat_count"`
	SubscribeCycleForCatalog        int    `gorm:"column:subscribe_cycle_for_catalog"`
	SubscribeCycleForMobilePosition int    `gorm:"column:subscribe_cycle_for_mobile_position"`
	MobilePositionSubmissionInterval int   `gorm:"column:mobile_position_submission_interval"`
	SubscribeCycleForAlarm          int    `gorm:"column:subscribe_cycle_for_alarm"`
	ServerID                 string `gorm:"column:server_id"`
	CreateTime               string `gorm:"column:create_time"`
	UpdateTime               string `gorm:"column:update_time"`
}

func (GBDevice) TableName() string { return "wvp_device" }

type GBDeviceChannel struct {
	ID              int     `gorm:"column:id;primaryKey;autoIncrement"`
	DeviceID        string  `gorm:"column:device_id"`
	DataType        int     `gorm:"column:data_type"`
	DataDeviceID    int     `gorm:"column:data_device_id"`
	GBDeviceID      string  `gorm:"column:gb_device_id"`
	Name            string  `gorm:"column:name"`
	Manufacturer    string  `gorm:"column:manufacturer"`
	Model           string  `gorm:"column:model"`
	Parental        int     `gorm:"column:parental"`
	ParentID        string  `gorm:"column:parent_id"`
	Status          string  `gorm:"column:status"`
	Longitude       float64 `gorm:"column:longitude"`
	Latitude        float64 `gorm:"column:latitude"`
	PTZType         int     `gorm:"column:ptz_type"`
	HasAudio        bool    `gorm:"column:has_audio"`
	SubCount        int     `gorm:"column:sub_count"`
	RecordPlanID    int     `gorm:"column:record_plan_id"`
	CreateTime      string  `gorm:"column:create_time"`
	UpdateTime      string  `gorm:"column:update_time"`
}

func (GBDeviceChannel) TableName() string { return "wvp_device_channel" }

type Alarm struct {
	ID          int     `gorm:"column:id;primaryKey;autoIncrement"`
	ChannelID   int     `gorm:"column:channel_id"`
	Description string  `gorm:"column:description"`
	SnapPath    string  `gorm:"column:snap_path"`
	RecordPath  string  `gorm:"column:record_path"`
	Longitude   float64 `gorm:"column:longitude"`
	Latitude    float64 `gorm:"column:latitude"`
	AlarmType   int     `gorm:"column:alarm_type"`
	AlarmTime   int64   `gorm:"column:alarm_time"`
}

func (Alarm) TableName() string { return "wvp_alarm" }

type MobilePosition struct {
	ID         int     `gorm:"column:id;primaryKey;autoIncrement"`
	ChannelID  int     `gorm:"column:channel_id"`
	Timestamp  int64   `gorm:"column:timestamp"`
	Longitude  float64 `gorm:"column:longitude"`
	Latitude   float64 `gorm:"column:latitude"`
	Altitude   float64 `gorm:"column:altitude"`
	Speed      float64 `gorm:"column:speed"`
	Direction  float64 `gorm:"column:direction"`
	CreateTime string  `gorm:"column:create_time"`
}

func (MobilePosition) TableName() string { return "wvp_mobile_position" }

type Platform struct {
	ID              int    `gorm:"column:id;primaryKey;autoIncrement"`
	Enable          bool   `gorm:"column:enable"`
	Name            string `gorm:"column:name"`
	ServerGBID      string `gorm:"column:server_gb_id"`
	ServerGBDomain  string `gorm:"column:server_gb_domain"`
	ServerIP        string `gorm:"column:server_ip"`
	ServerPort      int    `gorm:"column:server_port"`
	DeviceGBID      string `gorm:"column:device_gb_id"`
	DeviceIP        string `gorm:"column:device_ip"`
	DevicePort      string `gorm:"column:device_port"`
	Username        string `gorm:"column:username"`
	Password        string `gorm:"column:password"`
	Expires         string `gorm:"column:expires"`
	KeepTimeout     string `gorm:"column:keep_timeout"`
	Transport       string `gorm:"column:transport"`
	Status          bool   `gorm:"column:status"`
	AutoPushChannel bool   `gorm:"column:auto_push_channel"`
	ServerID        string `gorm:"column:server_id"`
	CreateTime      string `gorm:"column:create_time"`
	UpdateTime      string `gorm:"column:update_time"`
}

func (Platform) TableName() string { return "wvp_platform" }

type PlatformChannel struct {
	ID              int    `gorm:"column:id;primaryKey;autoIncrement"`
	PlatformID      int    `gorm:"column:platform_id"`
	DeviceChannelID int    `gorm:"column:device_channel_id"`
	CustomDeviceID  string `gorm:"column:custom_device_id"`
	CustomName      string `gorm:"column:custom_name"`
}

func (PlatformChannel) TableName() string { return "wvp_platform_channel" }

type StreamPush struct {
	ID               int    `gorm:"column:id;primaryKey;autoIncrement"`
	App              string `gorm:"column:app"`
	Stream           string `gorm:"column:stream"`
	MediaServerID    string `gorm:"column:media_server_id"`
	ServerID         string `gorm:"column:server_id"`
	PushTime         string `gorm:"column:push_time"`
	Status           bool   `gorm:"column:status"`
	UpdateTime       string `gorm:"column:update_time"`
	CreateTime       string `gorm:"column:create_time"`
	Pushing          bool   `gorm:"column:pushing"`
	StartOfflinePush bool   `gorm:"column:start_offline_push"`
}

func (StreamPush) TableName() string { return "wvp_stream_push" }

type StreamProxy struct {
	ID                      int    `gorm:"column:id;primaryKey;autoIncrement"`
	Type                    string `gorm:"column:type"`
	App                     string `gorm:"column:app"`
	Stream                  string `gorm:"column:stream"`
	SrcURL                  string `gorm:"column:src_url"`
	Timeout                 int    `gorm:"column:timeout"`
	FFmpegCmdKey            string `gorm:"column:ffmpeg_cmd_key"`
	RTSPType                string `gorm:"column:rtsp_type"`
	MediaServerID           string `gorm:"column:media_server_id"`
	EnableAudio             bool   `gorm:"column:enable_audio"`
	EnableMP4               bool   `gorm:"column:enable_mp4"`
	Pulling                 bool   `gorm:"column:pulling"`
	Enable                  bool   `gorm:"column:enable"`
	CreateTime              string `gorm:"column:create_time"`
	Name                    string `gorm:"column:name"`
	UpdateTime              string `gorm:"column:update_time"`
	StreamKey               string `gorm:"column:stream_key"`
	ServerID                string `gorm:"column:server_id"`
	EnableDisableNoneReader bool   `gorm:"column:enable_disable_none_reader"`
	RelatesMediaServerID    string `gorm:"column:relates_media_server_id"`
}

func (StreamProxy) TableName() string { return "wvp_stream_proxy" }

type CloudRecord struct {
	ID            int     `gorm:"column:id;primaryKey;autoIncrement"`
	App           string  `gorm:"column:app"`
	Stream        string  `gorm:"column:stream"`
	CallID        string  `gorm:"column:call_id"`
	StartTime     int64   `gorm:"column:start_time"`
	EndTime       int64   `gorm:"column:end_time"`
	MediaServerID string  `gorm:"column:media_server_id"`
	ServerID      string  `gorm:"column:server_id"`
	FileName      string  `gorm:"column:file_name"`
	Folder        string  `gorm:"column:folder"`
	FilePath      string  `gorm:"column:file_path"`
	Collect       bool    `gorm:"column:collect"`
	FileSize      int64   `gorm:"column:file_size"`
	TimeLen       float64 `gorm:"column:time_len"`
}

func (CloudRecord) TableName() string { return "wvp_cloud_record" }

type RecordPlan struct {
	ID         int    `gorm:"column:id;primaryKey;autoIncrement"`
	Snap       bool   `gorm:"column:snap"`
	Name       string `gorm:"column:name"`
	CreateTime string `gorm:"column:create_time"`
	UpdateTime string `gorm:"column:update_time"`
}

func (RecordPlan) TableName() string { return "wvp_record_plan" }

type RecordPlanItem struct {
	ID         int    `gorm:"column:id;primaryKey;autoIncrement"`
	Start      int    `gorm:"column:start"`
	Stop       int    `gorm:"column:stop"`
	WeekDay    int    `gorm:"column:week_day"`
	PlanID     int    `gorm:"column:plan_id"`
	CreateTime string `gorm:"column:create_time"`
	UpdateTime string `gorm:"column:update_time"`
}

func (RecordPlanItem) TableName() string { return "wvp_record_plan_item" }
