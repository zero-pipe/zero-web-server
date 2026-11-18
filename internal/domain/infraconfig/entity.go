package infraconfig

import "errors"

// ErrNotFound 表示尚未保存 MQTT 配置。
var ErrNotFound = errors.New("mqtt config not found")

// MQTTConfig 是平台消费者与插件发布端共用的消息队列配置（单行）。
type MQTTConfig struct {
	ID             int    `json:"id"`
	Enabled        bool   `json:"enabled"`
	Broker         string `json:"broker"`
	ClientIDPrefix string `json:"clientIdPrefix"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	TopicPrefix    string `json:"topicPrefix"`
	QoS            int    `json:"qos"`
	KeepAliveSec   int    `json:"keepAliveSec"`
	TLSEnabled     bool   `json:"tlsEnabled"`
	CACert         string `json:"caCert"`
	ClientCert     string `json:"clientCert"`
	ClientKey      string `json:"clientKey"`
	CreateTime     string `json:"createTime"`
	UpdateTime     string `json:"updateTime"`
}

// Default 返回未配置时的默认值。
func Default() *MQTTConfig {
	return &MQTTConfig{
		ID:             1,
		Enabled:        false,
		Broker:         "",
		ClientIDPrefix: "zws",
		TopicPrefix:    "zws/events",
		QoS:            1,
		KeepAliveSec:   30,
	}
}

type Repository interface {
	Get() (*MQTTConfig, error)
	Save(row *MQTTConfig) error
}
