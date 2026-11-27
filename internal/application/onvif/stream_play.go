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

// isStreamReadyForPlay 判断 ZMS 流是否真的可播（避免复用卡死的僵尸流）。
func isStreamReadyForPlay(info *port.StreamProbe, streamChannel string) bool {
	_ = streamChannel
	if info == nil || !info.Video {
		return false
	}
	// 码率足够 → 就绪
	if info.BytesSpeed >= minPlayBytesSpeed {
		return true
	}
	// 码率偏低但有真实帧在走(fps>0) → 也算就绪（容忍慢启动 / 低码率子码流）
	if info.VideoFps > 0 {
		return true
	}
	// bytes>0 但 fps=0 且码率很低 = 拉流卡死的僵尸流：判不就绪，交给上层重建。
	// （旧逻辑只要分辨率正常且 bytes>0 就判就绪，会把卡死流反复复用到播放器 → 黑屏）
	return false
}

func waitStreamReadyAttempts(streamChannel string) int {
	if streamChannel == "101" {
		return 75
	}
	return 45
}
