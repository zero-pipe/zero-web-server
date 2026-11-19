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
