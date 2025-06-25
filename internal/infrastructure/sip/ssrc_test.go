package sipinfra

import "testing"

func TestPlaySSRC(t *testing.T) {
	got := PlaySSRC("3402000000", 1)
	want := "0200000001"
	if got != want {
		t.Fatalf("PlaySSRC() = %q, want %q", got, want)
	}
}
