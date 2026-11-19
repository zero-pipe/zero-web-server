package shared

import "testing"

func TestDefaultChannelName(t *testing.T) {
	cases := map[string]string{
		DefaultChannelName("门口摄像机", 1):  "门口摄像机_01",
		DefaultChannelName("门口摄像机", 10): "门口摄像机_10",
		DefaultChannelName("", 2):       "通道_02",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	}
}
