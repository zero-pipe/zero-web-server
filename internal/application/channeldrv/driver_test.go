package channeldrv

import "testing"

func TestRegistry(t *testing.T) {
	r := New(Deps{})
	for _, proto := range []string{"gb28181", "onvif", "plugin"} {
		d, ok := r.Get(proto)
		if !ok || d.Protocol() != proto {
			t.Fatalf("driver %s missing or mismatched", proto)
		}
	}
	if _, ok := r.Get("unknown"); ok {
		t.Fatal("unexpected driver for unknown protocol")
	}
}

func TestClamp01(t *testing.T) {
	cases := map[float64]float64{-1: 0, 0: 0, 0.5: 0.5, 1: 1, 2: 1}
	for in, want := range cases {
		if got := clamp01(in); got != want {
			t.Errorf("clamp01(%v)=%v want %v", in, got, want)
		}
	}
}
