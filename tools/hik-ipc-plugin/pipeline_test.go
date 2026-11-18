//go:build hik && cgo

package main

import "testing"

func TestNormalizeEventType(t *testing.T) {
	cases := map[string]string{
		"VMD": "motion", "linedetection": "line_crossing",
		"fielddetection": "intrusion", "videoloss": "video_loss",
		"something": "something",
	}
	for in, want := range cases {
		if got := normalizeEventType(in); got != want {
			t.Errorf("%s: got %s want %s", in, got, want)
		}
	}
}

func TestChannelNumber(t *testing.T) {
	if got := channelNumber("1"); got != 101 {
		t.Fatalf("got %d", got)
	}
	if got := channelNumber("2"); got != 201 {
		t.Fatalf("got %d", got)
	}
	if got := channelNumber("bad"); got != 101 {
		t.Fatalf("got %d", got)
	}
}

func TestDigestAuthorization(t *testing.T) {
	auth, err := digestAuthorization(
		`Digest realm="testrealm@host.com", qop="auth", nonce="dcd98b7102dd2f0e8b11d0f600bfb0c093", opaque="5ccc069c403ebaf9f0171e9517f40e41"`,
		"GET", "/dir/index.html", "Mufasa", "Circle Of Life")
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(auth, `username="Mufasa"`) || !containsStr(auth, `uri="/dir/index.html"`) || !containsStr(auth, "response=") {
		t.Fatalf("unexpected auth: %s", auth)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
