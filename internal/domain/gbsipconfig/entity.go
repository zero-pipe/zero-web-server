package gbsipconfig

import "errors"

// ErrNotFound 表示尚未初始化 SIP 配置。
var ErrNotFound = errors.New("gb sip config not found")

// Config 本平台唯一国标 SIP 配置（领域实体）。
type Config struct {
	ID                 int    `json:"id"`
	IP                 string `json:"ip"`
	Port               int    `json:"port"`
	Domain             string `json:"domain"`
	DeviceID           string `json:"deviceId"`
	Password           string `json:"password"`
	Alarm              bool   `json:"alarm"`
	RequirePreRegister bool   `json:"requirePreRegister"`
	Transport          string `json:"transport"`
	StreamMode         string `json:"streamMode"`
	CreateTime         string `json:"createTime"`
	UpdateTime         string `json:"updateTime"`
}

type Repository interface {
	Get() (*Config, error)
	Save(row *Config) error
}
