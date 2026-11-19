package asset

import "testing"

func TestDriversRegistry(t *testing.T) {
	drivers := newDrivers(nil, nil, nil, nil)
	for _, proto := range []string{"gb28181", "onvif", "plugin"} {
		d, ok := drivers[proto]
		if !ok {
			t.Fatalf("missing driver for %s", proto)
		}
		if d.Protocol() != proto {
			t.Fatalf("driver %s reports %s", proto, d.Protocol())
		}
	}
}
