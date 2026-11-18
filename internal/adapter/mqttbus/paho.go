// Package mqttbus implements the platform MQTT consumer adapter (paho).
package mqttbus

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"zero-web-server/internal/port"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Bus struct {
	mu        sync.RWMutex
	client    mqtt.Client
	topic     string
	connected bool
	ready     bool
	lastErr   string
}

func New() *Bus { return &Bus{} }

var _ port.MessageBus = (*Bus)(nil)

func (b *Bus) Start(ctx context.Context, cfg port.MQTTBusConfig, handler port.MQTTMessageHandler) error {
	_ = ctx
	if strings.TrimSpace(cfg.Broker) == "" {
		return errors.New("mqtt broker 不能为空")
	}
	_ = b.Stop()

	prefix := strings.Trim(strings.TrimSpace(cfg.TopicPrefix), "/")
	if prefix == "" {
		prefix = "zws/events"
	}
	sharedGroup := strings.TrimSpace(cfg.SharedGroup)
	if sharedGroup == "" {
		sharedGroup = "zws"
	}
	baseTopic := prefix + "/#"
	sharedTopic := fmt.Sprintf("$share/%s/%s", sharedGroup, baseTopic)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.Broker)
	opts.SetClientID(cfg.ClientID)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		opts.SetPassword(cfg.Password)
	}
	keepAlive := cfg.KeepAliveSec
	if keepAlive <= 0 {
		keepAlive = 30
	}
	opts.SetKeepAlive(time.Duration(keepAlive) * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetCleanSession(false)

	if cfg.TLSEnabled {
		tlsCfg, err := buildTLS(cfg)
		if err != nil {
			return err
		}
		opts.SetTLSConfig(tlsCfg)
	}

	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		b.setStatus(false, errString(err))
	})
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		b.setStatus(true, "")
		b.mu.RLock()
		ready := b.ready
		b.mu.RUnlock()
		if ready {
			// Reconnect: re-subscribe explicitly (broker may have dropped session).
			b.subscribe(c, sharedTopic, baseTopic, cfg.QoS, handler)
		}
	})

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(8 * time.Second) {
		return errors.New("mqtt connect timeout")
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("mqtt connect: %w", err)
	}

	b.mu.Lock()
	b.client = client
	b.connected = true
	b.lastErr = ""
	b.mu.Unlock()

	// OnConnect may fire before assignment; subscribe explicitly and record the
	// topic that actually worked.
	topic, err := b.subscribeSync(client, sharedTopic, baseTopic, cfg.QoS, handler)
	if err != nil {
		client.Disconnect(250)
		return err
	}
	b.mu.Lock()
	b.topic = topic
	b.ready = true
	b.mu.Unlock()
	return nil
}

func (b *Bus) subscribe(client mqtt.Client, sharedTopic, baseTopic string, qos byte, handler port.MQTTMessageHandler) {
	_, _ = b.subscribeSync(client, sharedTopic, baseTopic, qos, handler)
}

func (b *Bus) subscribeSync(client mqtt.Client, sharedTopic, baseTopic string, qos byte, handler port.MQTTMessageHandler) (string, error) {
	cb := func(_ mqtt.Client, msg mqtt.Message) {
		if handler != nil {
			handler(msg.Topic(), msg.Payload())
		}
	}
	token := client.Subscribe(sharedTopic, qos, cb)
	if token.WaitTimeout(5*time.Second) && token.Error() == nil {
		return sharedTopic, nil
	}
	// Some brokers do not support shared subscriptions; fall back to plain.
	token = client.Subscribe(baseTopic, qos, cb)
	if token.WaitTimeout(5*time.Second) && token.Error() == nil {
		return baseTopic, nil
	}
	if err := token.Error(); err != nil {
		return "", fmt.Errorf("mqtt subscribe %s: %w", baseTopic, err)
	}
	return "", fmt.Errorf("mqtt subscribe %s timeout", baseTopic)
}

func (b *Bus) Stop() error {
	b.mu.Lock()
	client := b.client
	topic := b.topic
	b.client = nil
	b.connected = false
	b.ready = false
	b.mu.Unlock()
	if client == nil {
		return nil
	}
	if topic != "" && client.IsConnectionOpen() {
		client.Unsubscribe(topic).WaitTimeout(2 * time.Second)
	}
	client.Disconnect(250)
	return nil
}

func (b *Bus) Connected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

func (b *Bus) LastError() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastErr
}

func (b *Bus) Publish(ctx context.Context, topic string, payload []byte, qos byte) error {
	_ = ctx
	b.mu.RLock()
	client := b.client
	connected := b.connected
	b.mu.RUnlock()
	if client == nil || !connected {
		return errors.New("mqtt not connected")
	}
	token := client.Publish(topic, qos, false, payload)
	if !token.WaitTimeout(5 * time.Second) {
		return errors.New("mqtt publish timeout")
	}
	return token.Error()
}

func (b *Bus) Topic() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.topic
}

func (b *Bus) setStatus(connected bool, errMsg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.connected = connected
	b.lastErr = errMsg
}

func buildTLS(cfg port.MQTTBusConfig) (*tls.Config, error) {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if strings.TrimSpace(cfg.CACert) != "" {
		pool := x509.NewCertPool()
		pem, err := os.ReadFile(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("read mqtt ca: %w", err)
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, errors.New("invalid mqtt ca cert")
		}
		tlsCfg.RootCAs = pool
	}
	if strings.TrimSpace(cfg.ClientCert) != "" && strings.TrimSpace(cfg.ClientKey) != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCert, cfg.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("load mqtt client cert: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	return tlsCfg, nil
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
