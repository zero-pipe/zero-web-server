package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	MySQL        MySQLConfig        `mapstructure:"mysql"`
	Redis        RedisConfig        `mapstructure:"redis"`
	SIP          SIPConfig          `mapstructure:"sip"`
	Media        MediaConfig        `mapstructure:"media"`
	Log          LogConfig          `mapstructure:"log"`
	UserSettings UserSettingsConfig `mapstructure:"user_settings"`
	ONVIF        ONVIFConfig        `mapstructure:"onvif"`
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

type SIPConfig struct {
	Port     int    `mapstructure:"port"`
	Domain   string `mapstructure:"domain"`
	ID       string `mapstructure:"id"`
	Password string `mapstructure:"password"`
	Alarm    bool   `mapstructure:"alarm"`
}

type MediaConfig struct {
	ID       string         `mapstructure:"id"`
	Type     string         `mapstructure:"type"` // zeromediakit | zms (legacy zlm accepted)
	IP       string         `mapstructure:"ip"`
	HTTPPort int            `mapstructure:"http_port"`
	Secret   string         `mapstructure:"secret"`
	RTP      MediaRTPConfig `mapstructure:"rtp"`
}

func (c MediaConfig) BackendType() string {
	switch strings.ToLower(strings.TrimSpace(c.Type)) {
	case "zms", "zeromediakit", "mediakit", "zlm":
		return "zeromediakit"
	default:
		return "zeromediakit"
	}
}

type LogConfig struct {
	Level  string         `mapstructure:"level"`
	Format string         `mapstructure:"format"`
	Output string         `mapstructure:"output"`
	File   LogFileConfig  `mapstructure:"file"`
}

type LogFileConfig struct {
	Path       string `mapstructure:"path"`
	MaxSizeMB  int    `mapstructure:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAgeDays int    `mapstructure:"max_age_days"`
	Compress   bool   `mapstructure:"compress"`
}

type MediaRTPConfig struct {
	Enable         bool   `mapstructure:"enable"`
	PortRange      string `mapstructure:"port_range"`
	SendPortRange  string `mapstructure:"send_port_range"`
}

func (c MediaConfig) BaseURL() string {
	return fmt.Sprintf("http://%s:%d", c.IP, c.HTTPPort)
}

// SignalingBaseURL WebRTC 信令走平台 HTTP（反向代理到 zero-media-kit）。
func (c MediaConfig) SignalingBaseURL(serverPort int) string {
	if serverPort <= 0 {
		return c.BaseURL()
	}
	return fmt.Sprintf("http://%s:%d", c.IP, serverPort)
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

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}
