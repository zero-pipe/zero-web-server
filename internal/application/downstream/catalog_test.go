package downstreamapp

import "testing"

func TestTypeCodeAndClassify(t *testing.T) {
	if typeCodeOf("12345") != "" {
		t.Fatal("non-20-digit must yield empty type code")
	}
	if code := typeCodeOf("34020000001320000001"); code != "132" {
		t.Fatalf("type code=%q", code)
	}
	cases := map[string]string{
		"132": "channel", "131": "channel", "137": "channel",
		"215": "node", "216": "node", "200": "node", "209": "node",
		"":    "node",
	}
	for code, want := range cases {
		if got := classifyItem(code); got != want {
			t.Errorf("classifyItem(%q)=%q want %q", code, got, want)
		}
	}
}

func TestKnownParent(t *testing.T) {
	known := map[string]struct{}{"a": {}, "b": {}}
	if !isKnownParent("34020000002000000002", known, "34020000002000000002") {
		t.Fatal("platform itself is a valid parent")
	}
	if !isKnownParent("a", known, "p") {
		t.Fatal("stored node is a valid parent")
	}
	if isKnownParent("zzz", known, "p") {
		t.Fatal("unknown parent must be rejected")
	}
}
