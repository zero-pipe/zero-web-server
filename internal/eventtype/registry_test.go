package eventtype

import "testing"

func TestNormalize(t *testing.T) {
	if got := Normalize("Motion"); got.Code != "motion" {
		t.Fatalf("Normalize(Motion)=%s", got.Code)
	}
	if got := Normalize("unknown-vendor-event"); got.Code != "other" || !got.AlarmEligible {
		t.Fatalf("unknown should fallback to other: %+v", got)
	}
}

func TestAlarmEligibility(t *testing.T) {
	if !IsAlarmEligible("line_crossing") {
		t.Fatal("line_crossing should be alarm eligible")
	}
	if IsAlarmEligible("device_online") {
		t.Fatal("device_online should not be alarm eligible")
	}
}

func TestLegacyMapping(t *testing.T) {
	if got := LegacyAlarmType("motion"); got != 7 {
		t.Fatalf("motion legacy=%d", got)
	}
	if got := FromGBAlarmType(10); got != "line_crossing" {
		t.Fatalf("GB 10=%s", got)
	}
	if got := FromGBAlarmType(1234); got != "other" {
		t.Fatalf("unknown GB=%s", got)
	}
}
