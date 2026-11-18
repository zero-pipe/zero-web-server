package eventsub

import (
	"strings"
	"testing"

	domainplugin "zero-web-server/internal/domain/plugin"
)

func TestSupportsAlarm(t *testing.T) {
	if !supportsAlarm(&domainplugin.Plugin{Capabilities: []string{"live", "arming"}}) {
		t.Fatal("arming should be supported")
	}
	if !supportsAlarm(&domainplugin.Plugin{Capabilities: []string{"alarm"}}) {
		t.Fatal("alarm should be supported")
	}
	if supportsAlarm(&domainplugin.Plugin{Capabilities: []string{"live", "ptz"}}) {
		t.Fatal("live/ptz only should not be supported")
	}
}

func TestPublisherClientID(t *testing.T) {
	id := publisherClientID("hik-ipc", "192.168.1.64:8000")
	if strings.ContainsAny(id, ":. ") {
		t.Fatalf("client id contains unsafe chars: %s", id)
	}
	if !strings.HasSuffix(id, "-publisher") {
		t.Fatalf("unexpected client id: %s", id)
	}
}
