package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	MySQL        MySQLConfig        `mapstructure:"mysql"`
	Redis        RedisConfig        `mapstructure:"redis"`
	SIP          SIPConfig          `mapstructure:"sip"`
	GB           GBConfig           `mapstructure:"gb"`
	Media        MediaConfig        `mapstructure:"media"`
	Log          LogConfig          `mapstructure:"log"`
	UserSettings UserSettingsConfig `mapstructure:"user_settings"`
	ONVIF        ONVIFConfig        `mapstructure:"onvif"`
	Event        EventConfig        `mapstructure:"event"`
}

// EventConfig 事件中心行为。
type EventConfig struct {
	// RetentionDays 事件保留天数；0 使用默认 30 天，<0 关闭自动清理。
	RetentionDays int `mapstructure:"retention_days"`
}

// GBConfig 国标接入行为（与 sip 信令配置分离）
type GBConfig struct {
	// RequirePreRegister 为 true 时，未在平台预添加的设备 REGISTER 将被拒绝
	RequirePreRegister bool `mapstructure:"require_pre_register"`
}

// SIPConfig 国标 SIP：yaml 提供默认值；启动时写入库（若库空），之后以库为准，页面可热更新编码等。
type SIPConfig struct {
	// IP 摄像机可达的平台网卡地址，写入 INVITE Contact/Via；空则启动时自动探测。
	IP       string `mapstructure:"ip"`
	Port     int    `mapstructure:"port"`
	Domain   string `mapstructure:"domain"`
	ID       string `mapstructure:"id"`
	Password string `mapstructure:"password"`
	Alarm    bool   `mapstructure:"alarm"`
	// RedirectContact 非空时作为 2022 注册重定向入口，把设备 REGISTER 以 302
	// 指向该 Contact（格式 sip:<服务器编码>@<IP>:<端口>）。
	RedirectContact string `mapstructure:"redirect_contact"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type MySQLConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

func (c MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Database, c.Charset)
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
	PoolSize int    `mapstructure:"pool_size"`
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type MediaConfig struct {
	ID       string         `mapstructure:"id"`
	Type     string         `mapstructure:"type"` // zms | zero-media-server（兼容旧别名）
	IP       string         `mapstructure:"ip"`
	HTTPPort int            `mapstructure:"http_port"`
	Secret   string         `mapstructure:"secret"`
	RTP      MediaRTPConfig `mapstructure:"rtp"`

	// 以下为节点自注册（ZMS 上报）的端口与能力；旧手工节点只填 IP+HTTPPort 时保持全开、端口走默认。
	HTTPSSLPort   int `mapstructure:"http_ssl_port"`
	RTSPPort      int `mapstructure:"rtsp_port"`
	RTSPSSLPort   int `mapstructure:"rtsp_ssl_port"`
	RTMPPort      int `mapstructure:"rtmp_port"`
	RTMPSSLPort   int `mapstructure:"rtmp_ssl_port"`
	SRTPort       int `mapstructure:"srt_port"`
	WebRTCPortMin int `mapstructure:"webrtc_port_min"`
	WebRTCPortMax int `mapstructure:"webrtc_port_max"`
	// CapabilitiesReported 为 true 时，Enable* 才具备权威性；否则视为「全开」以兼容旧数据。
	CapabilitiesReported bool `mapstructure:"capabilities_reported"`
	EnableRTSP           bool `mapstructure:"enable_rtsp"`
	EnableRTMP           bool `mapstructure:"enable_rtmp"`
	EnableHLS            bool `mapstructure:"enable_hls"`
	EnableSRT            bool `mapstructure:"enable_srt"`
	EnableWebRTC         bool `mapstructure:"enable_webrtc"`
}

// RTSPAvailable 该节点是否提供 RTSP 播放（旧节点默认可用）。
func (c MediaConfig) RTSPAvailable() bool {
	return !c.CapabilitiesReported || c.EnableRTSP
}

// RTMPAvailable 该节点是否提供 RTMP 播放（旧节点默认可用）。
func (c MediaConfig) RTMPAvailable() bool {
	return !c.CapabilitiesReported || c.EnableRTMP
}

// HLSAvailable 该节点是否提供 HLS（旧节点默认可用）。
func (c MediaConfig) HLSAvailable() bool {
	return !c.CapabilitiesReported || c.EnableHLS
}

// WebRTCAvailable 该节点是否提供 WebRTC（旧节点默认可用）。
func (c MediaConfig) WebRTCAvailable() bool {
	return !c.CapabilitiesReported || c.EnableWebRTC
}

// RTSPPlayPort RTSP 播放端口（未上报时回退 8554）。
func (c MediaConfig) RTSPPlayPort() int {
	if c.RTSPPort > 0 {
		return c.RTSPPort
	}
	return 8554
}

// RTMPPlayPort RTMP 播放端口（未上报时回退 1935）。
func (c MediaConfig) RTMPPlayPort() int {
	if c.RTMPPort > 0 {
		return c.RTMPPort
	}
	return 1935
}

func (c MediaConfig) BackendType() string {
	switch strings.ToLower(strings.TrimSpace(c.Type)) {
	case "zms", "zero-media-server", "zeromediaserver", "zeromediakit", "mediakit", "zlm":
		return "zms"
	default:
		return "zms"
	}
}

// Configured 配置文件是否声明了媒体节点（已改为可选，节点以数据库为准）。
func (c MediaConfig) Configured() bool {
	return strings.TrimSpace(c.IP) != "" && c.HTTPPort > 0
}

type LogConfig struct {
	Level  string        `mapstructure:"level"`
	Format string        `mapstructure:"format"`
	Output string        `mapstructure:"output"`
	File   LogFileConfig `mapstructure:"file"`
}

type LogFileConfig struct {
	Path       string `mapstructure:"path"`
	MaxSizeMB  int    `mapstructure:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAgeDays int    `mapstructure:"max_age_days"`
	Compress   bool   `mapstructure:"compress"`
}

type MediaRTPConfig struct {
	Enable        bool   `mapstructure:"enable"`
	PortRange     string `mapstructure:"port_range"`
	SendPortRange string `mapstructure:"send_port_range"`
}

func (c MediaConfig) BaseURL() string {
	return fmt.Sprintf("http://%s:%d", c.IP, c.HTTPPort)
}

// SignalingBaseURL WebRTC 信令基址。
// 浏览器必须能访问该 host；禁止用 127.0.0.1（本机环回对远端浏览器无效，ICE 也会对不上）。
// signalingPort>0 时走平台 HTTP 反代到 ZMS；否则直连媒体节点 HTTP。
func (c MediaConfig) SignalingBaseURL(serverPort int) string {
	host := strings.TrimSpace(c.IP)
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	if serverPort <= 0 {
		return fmt.Sprintf("http://%s:%d", host, c.HTTPPort)
	}
	return fmt.Sprintf("http://%s:%d", host, serverPort)
}

// IsLoopbackHost 判断媒体/流 IP 是否为环回，WebRTC 场景应避免。
func IsLoopbackHost(ip string) bool {
	ip = strings.TrimSpace(strings.ToLower(ip))
	return ip == "" || ip == "127.0.0.1" || ip == "localhost" || ip == "::1"
}

type UserSettingsConfig struct {
	PlayTimeout       int    `mapstructure:"play_timeout"`
	RecordInfoTimeout int    `mapstructure:"record_info_timeout"`
	StreamOnDemand    bool   `mapstructure:"stream_on_demand"`
	PushAuthority     bool   `mapstructure:"push_authority"`
	RecordPushLive    bool   `mapstructure:"record_push_live"`
	ServerID          string `mapstructure:"server_id"`
	LoginTimeout      int64  `mapstructure:"login_timeout"`
	JWKFile           string `mapstructure:"jwk_file"`
}

type ONVIFConfig struct {
	DiscoveryTimeout int `mapstructure:"discovery_timeout"`
	ProbeInterval    int `mapstructure:"probe_interval"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// 可选本地覆盖：configs/config.local.yaml（不入库，放密码等）
	localPath := filepath.Join(filepath.Dir(path), "config.local.yaml")
	if st, err := os.Stat(localPath); err == nil && !st.IsDir() {
		v.SetConfigFile(localPath)
		if err := v.MergeInConfig(); err != nil {
			return nil, fmt.Errorf("merge local config %s: %w", localPath, err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}
