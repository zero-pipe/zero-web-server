package ipcdevice

import "testing"

func TestPickTransport(t *testing.T) {
	cases := []struct {
		name       string
		transports []string
		want       string
	}{
		{"prefer rtsp", []string{"rtp_push", "rtsp_url"}, "rtsp_url"},
		{"rtp only", []string{"rtp_push"}, "rtp_push"},
		{"empty defaults rtsp", nil, "rtsp_url"},
		{"unknown defaults rtsp", []string{"webrtc"}, "rtsp_url"},
	}
	for _, tc := range cases {
		if got := pickTransport(tc.transports); got != tc.want {
			t.Errorf("%s: got %s want %s", tc.name, got, tc.want)
		}
	}
}
