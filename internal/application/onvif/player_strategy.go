package onvifapp

import "strings"

// resolvePreferredPlayer 按实测编码选择播放器（与前端 playerStrategy.js 对齐）。
func resolvePreferredPlayer(videoCodec, audioCodec string, hasAudio bool) string {
	video := normalizeVideoCodec(videoCodec)
	if video == "H265" {
		return "h265web"
	}
	if prefersWebRTC(video, audioCodec, hasAudio) {
		return "webRTC"
	}
	return "jessibuca"
}

func prefersWebRTC(videoCodec, audioCodec string, hasAudio bool) bool {
	if normalizeVideoCodec(videoCodec) != "H264" {
		return false
	}
	if !hasAudio {
		return true
	}
	return isWebRTCAudioCodec(audioCodec)
}

func isWebRTCAudioCodec(raw string) bool {
	switch normalizeAudioCodec(raw) {
	case "OPUS", "PCMU", "PCMA":
		return true
	default:
		return false
	}
}

func normalizeAudioCodec(raw string) string {
	raw = stringsToUpperTrim(raw)
	switch raw {
	case "", "-":
		return ""
	case "MPEG4-GENERIC", "AAC", "MP4A-LATM":
		return "AAC"
	case "G711A", "G711", "ALAW":
		return "PCMA"
	case "G711U", "ULAW":
		return "PCMU"
	default:
		if strings.Contains(raw, "711A") {
			return "PCMA"
		}
		if strings.Contains(raw, "711U") {
			return "PCMU"
		}
		return raw
	}
}

func stringsToUpperTrim(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
