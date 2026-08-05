package sipinfra

import "github.com/zero-pipe/gb28181-go/ssrc"

// PlaySSRC returns a GB28181 play SSRC (0 + domain[3:8] + 4-digit seq).
func PlaySSRC(domain string, seq int) string {
	return ssrc.Play(domain, seq)
}
