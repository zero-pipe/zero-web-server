package persistence

import (
	"strings"
	"testing"

	domainasset "zero-web-server/internal/domain/asset"
)

func TestInferProfileRole(t *testing.T) {
	cases := []struct {
		label string
		index int
		want  string
	}{
		{"主码流", 0, domainasset.ProfileMain},
		{"Main Stream", 1, domainasset.ProfileMain},
		{"子码流", 1, domainasset.ProfileSub},
		{"sub", 1, domainasset.ProfileSub},
		{"third", 2, domainasset.ProfileThird},
		{"custom-a", 1, domainasset.ProfileCustom},
		{"", 0, domainasset.ProfileMain},
	}
	for _, tc := range cases {
		if got := inferProfileRole(tc.label, tc.index); got != tc.want {
			t.Errorf("%q/%d: got %s want %s", tc.label, tc.index, got, tc.want)
		}
	}
}

func TestChannelCaps(t *testing.T) {
	caps := channelCaps(1, true)
	set := map[string]bool{}
	for _, c := range caps {
		set[c] = true
	}
	for _, want := range []string{"play", "snapshot", "ptz", "audio"} {
		if !set[want] {
			t.Errorf("missing capability %s in %v", want, caps)
		}
	}
	if got := channelCaps(0, false); len(got) != 2 {
		t.Errorf("expected play+snapshot, got %v", got)
	}
}

// TestHasPTZ 锁定国标 PTZType 的云台判定口径（2016 A.2.1 g / 2022 A.2.1.9）：
// 只有 3-固定枪机 与 6/7-多目通道没有云台；0 表示未上报，按无云台处理。
func TestHasPTZ(t *testing.T) {
	cases := []struct {
		ptzType int
		want    bool
		desc    string
	}{
		{0, false, "未上报"},
		{1, true, "球机"},
		{2, true, "半球"},
		{3, false, "固定枪机"},
		{4, true, "遥控枪机（与固定枪机的差别就是可否远控）"},
		{5, true, "遥控半球（2022 新增）"},
		{6, false, "多目设备全景/拼接通道"},
		{7, false, "多目设备分割通道"},
	}
	for _, c := range cases {
		if got := hasPTZ(c.ptzType); got != c.want {
			t.Errorf("hasPTZ(%d)=%v, want %v（%s）", c.ptzType, got, c.want, c.desc)
		}
	}
	// 固定枪机不能出现在通道能力位里，否则界面会显示一个按不出反应的云台。
	for _, op := range channelCaps(3, false) {
		if op == "ptz" {
			t.Errorf("固定枪机不应获得 ptz 能力：%v", channelCaps(3, false))
		}
	}
}

func TestPluginOpSupported(t *testing.T) {
	// 未上报声明（nil）→ 维持宽松，全部放行。
	if !pluginOpSupported(nil, "playback") {
		t.Errorf("nil declared should be permissive")
	}
	// 只声明 login/channels/live → 只支持 play/sync，回放/云台/抓拍均不支持。
	declared := map[string]bool{"login": true, "channels": true, "live": true}
	cases := map[string]bool{
		"play": true, "sync": true,
		"playback": false, "ptz": false, "snapshot": false, "alarm": false, "subscribe": false,
	}
	for op, want := range cases {
		if got := pluginOpSupported(declared, op); got != want {
			t.Errorf("op %s: got %v want %v", op, got, want)
		}
	}
	// record_query 亦视为支持回放；arm 视为告警+订阅。
	declared2 := map[string]bool{"record_query": true, "arm": true}
	if !pluginOpSupported(declared2, "playback") {
		t.Errorf("record_query should enable playback")
	}
	if !pluginOpSupported(declared2, "alarm") || !pluginOpSupported(declared2, "subscribe") {
		t.Errorf("arm should enable alarm/subscribe")
	}
	if pluginOpSupported(declared2, "play") {
		t.Errorf("play requires live, not record_query")
	}
}

func TestStableIDDeterministic(t *testing.T) {
	a := stableID("profile", "resource-1", "main")
	b := stableID("profile", "resource-1", "main")
	c := stableID("profile", "resource-1", "sub")
	if a != b {
		t.Fatalf("stableID not deterministic: %s vs %s", a, b)
	}
	if a == c {
		t.Fatalf("stableID collision for different inputs")
	}
}

func TestMergeResourceAttrsPreservesUserEdits(t *testing.T) {
	incoming := `{"ptzType":3,"hasAudio":true,"longitude":1,"latitude":2}`
	if got := mergeResourceAttrs(`{"ptzType":1}`, incoming); got != incoming {
		t.Fatalf("without user edits should take incoming attrs, got %s", got)
	}
	existing := `{"ptzType":1,"longitude":120.5,"latitude":30.5,"mapLevel":12,"tags":"gate","enabled":false,"userEdited":true}`
	got := mergeResourceAttrs(existing, incoming)
	for _, want := range []string{`"longitude":120.5`, `"latitude":30.5`, `"mapLevel":12`, `"tags":"gate"`, `"enabled":false`, `"ptzType":3`} {
		if !strings.Contains(got, want) {
			t.Errorf("merged attrs missing %s: %s", want, got)
		}
	}
}
