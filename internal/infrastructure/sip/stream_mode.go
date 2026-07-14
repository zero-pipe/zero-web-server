package sipinfra

import "github.com/zero-pipe/gb28181-go/transport"

// NormalizeStreamMode returns the RTP media transport for INVITE SDP.
// This is independent of SIP signaling transport (device.Transport: UDP/TCP on port 5060).
// Empty or unknown values default to UDP.
func NormalizeStreamMode(mode string) string {
	return transport.Normalize(mode)
}
