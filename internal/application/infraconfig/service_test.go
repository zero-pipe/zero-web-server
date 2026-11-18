package infraconfig

import (
	"context"
	"errors"
	"testing"

	"zero-web-server/internal/port"
)

type fakeBus struct {
	published []string
	topic     string
}

func (f *fakeBus) Start(ctx context.Context, cfg port.MQTTBusConfig, h port.MQTTMessageHandler) error {
	f.topic = cfg.TopicPrefix + "/#"
	return nil
}
func (f *fakeBus) Stop() error       { return nil }
func (f *fakeBus) Connected() bool   { return true }
func (f *fakeBus) LastError() string { return "" }
func (f *fakeBus) Topic() string     { return f.topic }
func (f *fakeBus) Publish(ctx context.Context, topic string, payload []byte, qos byte) error {
	f.published = append(f.published, topic)
	return nil
}

func TestHandleMessagePublishesDLQ(t *testing.T) {
	bus := &fakeBus{}
	s := &Service{bus: bus, dlqTopic: "zws/events-dlq", qos: 1}
	s.SetMessageHandler(func(topic string, payload []byte) error {
		return errors.New("persist failed")
	})
	s.handleMessage("zws/events/D1", []byte(`{"bad":true}`))
	if len(bus.published) != 1 || bus.published[0] != "zws/events-dlq" {
		t.Fatalf("expected dlq publish, got %#v", bus.published)
	}
}
