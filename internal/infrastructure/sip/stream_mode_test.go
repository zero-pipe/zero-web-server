package sipinfra

import "testing"

func TestNormalizeStreamMode(t *testing.T) {
	if got := NormalizeStreamMode(""); got != "UDP" {
		t.Fatalf("empty -> %q", got)
	}
	if got := NormalizeStreamMode("TCP-PASSIVE"); got != "TCP-PASSIVE" {
		t.Fatalf("TCP-PASSIVE -> %q", got)
	}
	if got := NormalizeStreamMode("udp"); got != "UDP" {
		t.Fatalf("udp -> %q", got)
	}
}
