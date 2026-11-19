// Package mediaurl 提供与流媒体实现无关的播放/推流地址拼装与日志辅助。
package mediaurl

import (
	"fmt"
	"strings"

	applog "zero-web-server/pkg/log"
)

// LiveApp 统一直播 app：所有接入协议（GB28181 / ONVIF / 插件）的直播流都落在 live/<设备内码>_<通道内码>。
const LiveApp = "live"

// LiveStream 统一流标识：<设备内码>_<通道内码>。
// 设备内码/通道内码是平台归一化身份；协议侧标识（国标码、通道号、ProfileToken）不参与流命名。
// 任一内码缺失时返回空串，调用方必须拒绝播放，禁止回退协议标识。
func LiveStream(deviceInternalCode, channelInternalCode string) string {
	d := strings.TrimSpace(deviceInternalCode)
	c := strings.TrimSpace(channelInternalCode)
	if d == "" || c == "" {
		return ""
	}
	return d + "_" + c
}

// PlaybackStream 回放流标识：<设备内码>_<通道内码>_playback。
func PlaybackStream(deviceInternalCode, channelInternalCode string) string {
	if base := LiveStream(deviceInternalCode, channelInternalCode); base != "" {
		return base + "_playback"
	}
	return ""
}

// DownloadStream 下载流标识：<设备内码>_<通道内码>_download。
func DownloadStream(deviceInternalCode, channelInternalCode string) string {
	if base := LiveStream(deviceInternalCode, channelInternalCode); base != "" {
		return base + "_download"
	}
	return ""
}

// BuildGB28181PushURL 国标收流地址摘要（UDP 或 TCP 被动 listen 端口）。
func BuildGB28181PushURL(sdpIP string, port int, tcpMode int) string {
	switch tcpMode {
	case 1:
		return fmt.Sprintf("tcp://%s:%d (RTP/PS RFC4571, camera -> zero-media-server listen)", sdpIP, port)
	case 2:
		return "tcp://camera:port (RTP/PS RFC4571, zero-media-server -> camera after 200 OK)"
	default:
		return fmt.Sprintf("udp://%s:%d (RTP/PS, camera -> zero-media-server)", sdpIP, port)
	}
}

// BuildWebRTCURLs 拼装 WebRTC 播放/推流信令地址。
func BuildWebRTCURLs(baseURL, app, stream string, push bool) (rtc, rtcs string) {
	typ := "play"
	if push {
		typ = "push"
	}
	q := fmt.Sprintf("app=%s&stream=%s&type=%s", app, stream, typ)
	rtc = fmt.Sprintf("%s/index/api/webrtc?%s", strings.TrimRight(baseURL, "/"), q)
	rtcs = strings.Replace(rtc, "http://", "https://", 1)
	return rtc, rtcs
}

// LogPlayStreamPaths 以 debug 级别打印推/拉流地址，便于排障。
func LogPlayStreamPaths(tag, app, stream, pushURL string, urls map[string]string) {
	applog.Debugf("[%s] app=%s stream=%s", tag, app, stream)
	if pushURL != "" {
		applog.Debugf("[%s] push  %s", tag, pushURL)
	}
	if urls == nil {
		return
	}
	if v := urls["flv"]; v != "" {
		applog.Debugf("[%s] pull  http-flv : %s", tag, v)
	}
	if v := urls["ws"]; v != "" {
		applog.Debugf("[%s] pull  ws-flv  : %s", tag, v)
	}
	if v := urls["hls"]; v != "" {
		applog.Debugf("[%s] pull  hls     : %s", tag, v)
	}
	if v := urls["rtsp"]; v != "" {
		applog.Debugf("[%s] pull  rtsp    : %s", tag, v)
	}
	if v := urls["rtmp"]; v != "" {
		applog.Debugf("[%s] pull  rtmp    : %s", tag, v)
	}
	rtc, rtcs := BuildWebRTCURLs("", app, stream, false)
	if base := urls["flv"]; base != "" {
		if i := strings.Index(base, "://"); i > 0 {
			if j := strings.Index(base[i+3:], "/"); j >= 0 {
				rtcBase := base[:i+3+j]
				rtc, rtcs = BuildWebRTCURLs(rtcBase, app, stream, false)
			}
		}
	}
	if rtc != "" {
		applog.Debugf("[%s] pull  webrtc  : %s", tag, rtc)
	}
	if rtcs != "" && rtcs != rtc {
		applog.Debugf("[%s] pull  webrtcs : %s", tag, rtcs)
	}
}
