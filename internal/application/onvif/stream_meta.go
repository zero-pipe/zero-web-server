package onvifapp

import (
	"regexp"
	"strings"

	domainonvif "zero-web-kit/internal/domain/onvif"
)

// 海康 RTSP 常见路径：/Streaming/Channels/101（主） /102（子）
var hikvisionChannelPathRE = regexp.MustCompile(`(?i)/Streaming/Channels/(\d+)`)

func parseRTSPStreamChannel(streamURI string) string {
	m := hikvisionChannelPathRE.FindStringSubmatch(streamURI)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func streamTypeLabel(channelNo string) string {
	switch channelNo {
	case "101":
		return "主码流"
	case "102":
		return "子码流"
	default:
		return ""
	}
}

func streamChannelLabel(channelNo string) string {
	if label := streamTypeLabel(channelNo); label != "" {
		return label
	}
	if channelNo != "" {
		return "码流 " + channelNo
	}
	return "码流"
}

func enrichChannelMeta(ch *domainonvif.Channel, profileName string) {
	chNo := parseRTSPStreamChannel(ch.StreamURI)
	ch.StreamChannel = chNo
	ch.StreamType = streamTypeLabel(chNo)
	ch.ConfigCodec = normalizeVideoCodec(ch.Codec)

	label := ch.StreamType
	if label == "" {
		label = strings.TrimSpace(profileName)
	}
	if label == "" {
		label = ch.ProfileToken
	} else if profileName != "" {
		pn := strings.TrimSpace(profileName)
		if pn != "" && !strings.EqualFold(pn, label) && !strings.EqualFold(pn, ch.StreamType) {
			label = label + " · " + pn
		}
	}
	ch.Name = label
}

func enrichChannelDisplay(ch *domainonvif.Channel) {
	if ch == nil {
		return
	}
	chNo := parseRTSPStreamChannel(ch.StreamURI)
	if ch.StreamChannel == "" {
		ch.StreamChannel = chNo
	}
	if ch.StreamType == "" {
		ch.StreamType = streamTypeLabel(chNo)
	}
	if ch.ConfigCodec == "" {
		ch.ConfigCodec = normalizeVideoCodec(ch.Codec)
	}
}
