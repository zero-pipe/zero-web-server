package onvifapp

import (
	"zero-web-server/internal/port"
)

const minPlayBytesSpeed int64 = 2048

// isSuspiciousVideoSize 海康 RTSP sprop 常见占位分辨率（如 208x64 / 208x96）。
func isSuspiciousVideoSize(width, height int) bool {
	if width == 0 || height == 0 {
		return true
	}
	return width < 320 || height < 240
}

// isStreamReadyForPlay 判断 ZMS 流是否已有可播数据（避免复用半就绪/僵尸流）。
func isStreamReadyForPlay(info *port.StreamProbe, streamChannel string) bool {
	_ = streamChannel
	if info == nil || !info.Video {
		return false
	}
	if info.BytesSpeed >= minPlayBytesSpeed {
		return true
	}
	if !isSuspiciousVideoSize(info.Width, info.Height) && info.BytesSpeed > 0 {
		return true
	}
	return false
}

func waitStreamReadyAttempts(streamChannel string) int {
	if streamChannel == "101" {
		return 75
	}
	return 45
}
