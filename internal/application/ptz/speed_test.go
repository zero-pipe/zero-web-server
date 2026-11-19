package ptzapp

import "testing"

func TestNormalizeSpeed(t *testing.T) {
	if normalizeSpeed(0, 255) != 0 || normalizeSpeed(-5, 255) != 0 {
		t.Fatal("non-positive speed should be 0")
	}
	if got := normalizeSpeed(255, 255); got != 1 {
		t.Fatalf("max speed should be 1, got %v", got)
	}
	if got := normalizeSpeed(510, 255); got != 1 {
		t.Fatalf("over max should clamp to 1, got %v", got)
	}
	if got := normalizeSpeed(16, 16); got != 1 {
		t.Fatalf("zoom max should be 1, got %v", got)
	}
}
