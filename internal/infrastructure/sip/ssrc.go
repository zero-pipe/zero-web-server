package sipinfra

import "fmt"

// PlaySSRC returns a GB28181 play SSRC matching WVP SSRCFactory (0 + domain[3:8] + 4-digit seq).
func PlaySSRC(domain string, seq int) string {
	part := domain
	if len(domain) >= 8 {
		part = domain[3:8]
	}
	return fmt.Sprintf("0%s%04d", part, seq%10000)
}
