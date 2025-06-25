package sipinfra

import "strings"

// NormalizeStreamMode returns the RTP media transport for INVITE SDP.
// This is independent of SIP signaling transport (device.Transport: UDP/TCP on port 5060).
// Empty or unknown values default to UDP.
func NormalizeStreamMode(mode string) string {
	switch strings.ToUpper(strings.TrimSpace(mode)) {
	case "TCP-ACTIVE":
		return "TCP-ACTIVE"
	case "TCP-PASSIVE":
		return "TCP-PASSIVE"
	case "UDP":
		return "UDP"
	default:
		return "UDP"
	}
}
