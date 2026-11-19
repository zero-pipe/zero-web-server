package upstreamapp

import (
	"testing"

	domaintree "zero-web-server/internal/domain/tree"
)

func TestRegionNodeID(t *testing.T) {
	cases := map[string]string{
		"530100": "53010000002160000000",
		"53":     "53000000002160000000",
		"5301":   "53010000002160000000",
		"34020000001320000001": "34020000001320000001", // 已是 20 位则原样返回
	}
	for in, want := range cases {
		if got := regionNodeID(in); got != want {
			t.Errorf("regionNodeID(%q)=%q want %q", in, got, want)
		}
	}
	if regionNodeID("") != "" || regionNodeID("123456789") != "" {
		t.Error("empty/oversized codes must map to empty")
	}
}

func TestCivilCodeOf(t *testing.T) {
	for _, ok := range []string{"53", "5301", "530100", "53010000"} {
		if civilCodeOf(ok) != ok {
			t.Errorf("civilCodeOf(%q) should keep valid code", ok)
		}
	}
	for _, bad := range []string{"", "5", "530", "5301000", "34020000001320000001"} {
		if civilCodeOf(bad) != "" {
			t.Errorf("civilCodeOf(%q) should drop invalid code", bad)
		}
	}
}

func TestItemTypeOfID(t *testing.T) {
	if itemTypeOfID("34020000001320000001") != "channel" {
		t.Error("132 should be channel")
	}
	if itemTypeOfID("34020000002150000001") != "node" {
		t.Error("215 should be node")
	}
	if itemTypeOfID("53010000002160000000") != "node" {
		t.Error("216 should be node")
	}
	if itemTypeOfID("bad") != "node" {
		t.Error("unknown id should be node")
	}
}

func TestExpandGroupSubtree(t *testing.T) {
	groups := []domaintree.Node{
		{DeviceID: "g215", Name: "业务分组", ParentDeviceID: ""},
		{DeviceID: "g216a", Name: "虚拟组织A", ParentDeviceID: "g215"},
		{DeviceID: "g216b", Name: "虚拟组织B", ParentDeviceID: "g215"},
		{DeviceID: "gother", Name: "无关", ParentDeviceID: ""},
	}
	got := expandGroupSubtree([]string{"g215"}, groups)
	for _, want := range []string{"g215", "g216a", "g216b"} {
		if _, ok := got[want]; !ok {
			t.Fatalf("subtree missing %s: %v", want, got)
		}
	}
	if _, ok := got["gother"]; ok {
		t.Fatal("unrelated group must not be included")
	}
	if d := dedupeInts([]int{1, 2, 1, 3}); len(d) != 3 {
		t.Fatalf("dedupe failed: %v", d)
	}
}
