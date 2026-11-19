package civilcode

import "testing"

func TestAncestors(t *testing.T) {
	Configure("../../configs/civilCode.csv")
	chain := Ancestors("110101")
	if len(chain) != 3 {
		t.Fatalf("expected 3 levels (11/1101/110101), got %d: %+v", len(chain), chain)
	}
	if chain[0].DeviceID != "11" || chain[1].DeviceID != "1101" || chain[2].DeviceID != "110101" {
		t.Fatalf("unexpected chain: %+v", chain)
	}
	if chain[2].Name != "东城区" {
		t.Fatalf("unexpected name: %q", chain[2].Name)
	}
	if got := Ancestors("999999"); got != nil {
		t.Fatalf("unknown code should return nil, got %+v", got)
	}
}
