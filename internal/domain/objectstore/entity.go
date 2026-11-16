package objectstore

import (
	"errors"
	"time"

	"zero-web-server/internal/port"
)

// ErrNotFound 表示未配置对象存储。
var ErrNotFound = errors.New("object store config not found")

// Config 对象存储配置（领域实体）。
type Config struct {
	ID         int    `json:"id"`
	Enabled    bool   `json:"enabled"`
	Provider   string `json:"provider"`
	Endpoint   string `json:"endpoint"`
	Region     string `json:"region"`
	Bucket     string `json:"bucket"`
	AccessKey  string `json:"accessKey"`
	SecretKey  string `json:"secretKey"`
	UseSSL     bool   `json:"useSSL"`
	PathStyle  bool   `json:"pathStyle"`
	PublicBase string `json:"publicBase"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

// ToPortConfig 转换为端口配置视图。
func (c *Config) ToPortConfig() port.ObjectStoreConfig {
	if c == nil {
		return port.ObjectStoreConfig{Provider: "noop"}
	}
	return port.ObjectStoreConfig{
		Enabled: c.Enabled, Provider: c.Provider, Endpoint: c.Endpoint,
		Region: c.Region, Bucket: c.Bucket, AccessKey: c.AccessKey,
		SecretKey: c.SecretKey, UseSSL: c.UseSSL, PathStyle: c.PathStyle,
		PublicBase: c.PublicBase,
	}
}

// Default 返回未配置时的默认值。
func Default() *Config {
	now := time.Now().Format("2006-01-02 15:04:05")
	return &Config{ID: 1, Enabled: false, Provider: "noop", PathStyle: true, CreateTime: now, UpdateTime: now}
}

type Repository interface {
	Get() (*Config, error)
	Save(row *Config) error
}
