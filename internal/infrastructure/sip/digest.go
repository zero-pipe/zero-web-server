package sipinfra

import (
	"net/http"

	"github.com/zero-pipe/gb28181-go/digest"
)

type DigestChallenge = digest.Challenge

func NewDigestChallenge(realm string) DigestChallenge {
	return digest.NewChallenge(realm)
}

func ParseAuthorization(header string) map[string]string {
	return digest.ParseAuthorization(header)
}

func VerifyDigest(authHeader, method, uri, username, password, realm, nonce string) bool {
	return digest.Verify(authHeader, method, uri, username, password, realm, nonce)
}

func WWWAuthenticateHeader(ch DigestChallenge) http.Header {
	return digest.WWWAuthenticateHeader(ch)
}
