package onvifapp

import "strings"

// resolvePreferredPlayer 按实测编码选择播放器（与前端 playerStrategy.js 对齐）。
func resolvePreferredPlayer(videoCodec, audioCodec string, hasAudio bool) string {
	video := normalizeVideoCodec(videoCodec)
	if video == "H265" {
		return "h265web"
	}
	// WebRTC 修好后：Opus/PCMU 或无声均优先 webRTC
	_ = audioCodec
	_ = hasAudio
	return "jessibuca"
}

func normalizeAudioCodec(raw string) string {
	raw = stringsToUpperTrim(raw)
	switch raw {
	case "", "-":
		return ""
	case "MPEG4-GENERIC", "AAC", "MP4A-LATM":
		return "AAC"
	default:
		return raw
	}
}

func stringsToUpperTrim(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
