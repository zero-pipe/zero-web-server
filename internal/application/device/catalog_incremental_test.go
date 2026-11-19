package deviceapp

import (
	"testing"

	domainchannel "zero-web-server/internal/domain/channel"
	"zero-web-server/internal/gbsip"
)

func ch(gb, name, status string) *domainchannel.Channel {
	return &domainchannel.Channel{DeviceID: "dev", GBDeviceID: gb, Name: name, Status: status}
}

func byGB(m map[string]*domainchannel.Channel, list []*domainchannel.Channel) {
	for _, c := range list {
		m[c.GBDeviceID] = c
	}
}

func TestMergeCatalogIncrementalUpdateKeepsOthers(t *testing.T) {
	current := []*domainchannel.Channel{ch("34020000001310000001", "门口_01", "ON"), ch("34020000001310000002", "门口_02", "ON")}
	items := []gbsip.CatalogItem{{DeviceID: "34020000001310000002", Event: "UPDATE", Status: "OFF"}}

	merged, stats := mergeCatalogIncremental(current, items, "dev", "门口", "2026-01-01 00:00:00")

	if stats.removed != 0 || stats.added != 0 || stats.updated != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if len(merged) != 2 {
		t.Fatalf("expected both channels preserved, got %d", len(merged))
	}
	m := map[string]*domainchannel.Channel{}
	byGB(m, merged)
	if m["34020000001310000001"].Status != "ON" {
		t.Fatal("untouched channel should keep status ON")
	}
	if m["34020000001310000002"].Status != "OFF" {
		t.Fatal("notified channel should switch to OFF")
	}
	if m["34020000001310000002"].Name != "门口_02" {
		t.Fatal("user channel name must be preserved on update")
	}
}

func TestMergeCatalogIncrementalAddAndDel(t *testing.T) {
	current := []*domainchannel.Channel{ch("34020000001310000001", "门口_01", "ON")}
	items := []gbsip.CatalogItem{
		{DeviceID: "34020000001310000001", Event: "DEL"},
		{DeviceID: "34020000001310000009", Event: "ADD", Status: "ON"},
	}

	merged, stats := mergeCatalogIncremental(current, items, "dev", "门口", "2026-01-01 00:00:00")

	if stats.removed != 1 || stats.added != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if len(merged) != 1 || merged[0].GBDeviceID != "34020000001310000009" {
		t.Fatalf("expected only the added channel, got %+v", merged)
	}
	if merged[0].Name != "门口_channel_2" {
		t.Fatalf("new channel should use next default name, got %q", merged[0].Name)
	}
}

func TestMergeCatalogIncrementalStatusOnly(t *testing.T) {
	current := []*domainchannel.Channel{ch("34020000001310000001", "门口_01", "ON")}
	items := []gbsip.CatalogItem{{DeviceID: "34020000001310000001", Event: "VLOST"}}

	merged, stats := mergeCatalogIncremental(current, items, "dev", "门口", "2026-01-01 00:00:00")
	if stats.updated != 1 || len(merged) != 1 {
		t.Fatalf("expected pure status update, got stats=%+v n=%d", stats, len(merged))
	}
	if merged[0].Name != "门口_01" {
		t.Fatal("status-only notify must not rename the channel")
	}
}

func TestMergeCatalogIncrementalEmptyNotifyKeepsAll(t *testing.T) {
	current := []*domainchannel.Channel{ch("34020000001310000001", "门口_01", "ON")}
	merged, _ := mergeCatalogIncremental(current, []gbsip.CatalogItem{{DeviceID: "34020000001310000077", Event: "DEL"}}, "dev", "门口", "now")
	if len(merged) != 1 || merged[0].GBDeviceID != "34020000001310000001" {
		t.Fatal("deleting an unknown channel must not clear the set")
	}
}
