package mediakit

import (
	"strings"
	"testing"

	"zero-web-server/internal/infrastructure/config"
)

func TestBuildPlayURLsForMedia_RegisteredCapabilities(t *testing.T) {
	cfg := config.MediaConfig{
		Type:                 "zms",
		IP:                   "10.0.0.5",
		HTTPPort:             8080,
		RTSPPort:             18554,
		RTMPPort:             11935,
		CapabilitiesReported: true,
		EnableRTSP:           false, // 关闭 RTSP
		EnableRTMP:           true,
		EnableHLS:            false, // 关闭 HLS
	}
	u := BuildPlayURLsForMedia(cfg, "live", "cam1")

	if _, ok := u["hls"]; ok {
		t.Fatalf("HLS disabled but got url: %q", u["hls"])
	}
	if _, ok := u["rtsp"]; ok {
		t.Fatalf("RTSP disabled but got url: %q", u["rtsp"])
	}
	if got, want := u["flv"], "http://10.0.0.5:8080/live/cam1.flv"; got != want {
		t.Fatalf("flv = %q want %q", got, want)
	}
	// RTMP 开启：应使用上报端口而非 1935
	if got, want := u["rtmp"], "rtmp://10.0.0.5:11935/live/cam1"; got != want {
		t.Fatalf("rtmp = %q want %q", got, want)
	}
}

func TestBuildPlayURLsForMedia_LegacyNodeUsesDefaults(t *testing.T) {
	// CapabilitiesReported=false 的旧节点：视为全开，RTSP/RTMP 未上报端口走 8554/1935
	cfg := config.MediaConfig{Type: "zms", IP: "10.0.0.5", HTTPPort: 8080}
	u := BuildPlayURLsForMedia(cfg, "live", "cam1")

	if !strings.HasPrefix(u["rtsp"], "rtsp://10.0.0.5:8554/") {
		t.Fatalf("legacy rtsp = %q", u["rtsp"])
	}
	if !strings.HasPrefix(u["rtmp"], "rtmp://10.0.0.5:1935/") {
		t.Fatalf("legacy rtmp = %q", u["rtmp"])
	}
	if u["hls"] == "" {
		t.Fatalf("legacy node should expose hls")
	}
}

func TestBuildStreamPlayURLs_WebRTCDisabled(t *testing.T) {
	cfg := config.MediaConfig{
		Type: "zms", IP: "10.0.0.5", HTTPPort: 8080,
		CapabilitiesReported: true, EnableHLS: true, EnableRTSP: true, EnableRTMP: true,
		EnableWebRTC: false,
	}
	u := BuildStreamPlayURLs(cfg, "live", "cam1", false, 0)
	if u.Rtc != "" || u.Rtcs != "" {
		t.Fatalf("webrtc disabled but got rtc=%q rtcs=%q", u.Rtc, u.Rtcs)
	}
}
