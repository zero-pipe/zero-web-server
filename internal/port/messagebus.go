package port

import "context"

// MQTTBusConfig 是消息总线运行参数（与具体 broker 解耦）。
type MQTTBusConfig struct {
	Broker       string
	ClientID     string
	Username     string
	Password     string
	TopicPrefix  string
	SharedGroup  string
	QoS          byte
	KeepAliveSec int
	TLSEnabled   bool
	CACert       string
	ClientCert   string
	ClientKey    string
}

// MQTTMessageHandler 处理订阅到的消息。
type MQTTMessageHandler func(topic string, payload []byte)

// MessageBus 是消息总线端口（当前用于事件消费）。
type MessageBus interface {
	Start(ctx context.Context, cfg MQTTBusConfig, handler MQTTMessageHandler) error
	Stop() error
	Connected() bool
	LastError() string
	Topic() string
}
