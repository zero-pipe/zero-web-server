package channelapp

import "testing"

func TestGBGroupTypeCode(t *testing.T) {
	if gbGroupTypeCode("34020000002150000001") != "215" {
		t.Fatal("215 expected")
	}
	if gbGroupTypeCode("34020000002160000001") != "216" {
		t.Fatal("216 expected")
	}
	if gbGroupTypeCode("34020000001320000001") != "" {
		t.Fatal("camera is not a group node")
	}
}
