//go:build hik && cgo

package main

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// isapiClient speaks Digest auth to Hikvision ISAPI over HTTP/HTTPS.
type isapiClient struct {
	base *url.URL
	http *http.Client
}

func newISAPIClient(ip string, port int, https bool, insecure bool, user, pass string) *isapiClient {
	scheme := "http"
	if https {
		scheme = "https"
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
		Proxy:           http.ProxyFromEnvironment,
	}
	return &isapiClient{
		base: &url.URL{Scheme: scheme, Host: fmt.Sprintf("%s:%d", ip, port)},
		http: &http.Client{
			Transport: &digestTransport{user: user, pass: pass, base: transport},
			Timeout:   0, // alertStream is long-lived
		},
	}
}

func (c *isapiClient) url(path string) string {
	u := *c.base
	u.Path = path
	return u.String()
}

func (c *isapiClient) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path), nil)
	if err != nil {
		return nil, err
	}
	return c.http.Do(req)
}

type digestTransport struct {
	user, pass string
	base       http.RoundTripper
}

func (t *digestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}
	challenge := resp.Header.Get("WWW-Authenticate")
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	resp.Body.Close()
	auth, err := digestAuthorization(challenge, req.Method, req.URL.RequestURI(), t.user, t.pass)
	if err != nil {
		return nil, err
	}
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", auth)
	return t.base.RoundTrip(req2)
}

var digestKV = regexp.MustCompile(`([a-zA-Z0-9_-]+)="([^"]*)"`)

func digestAuthorization(challenge, method, uri, user, pass string) (string, error) {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(challenge)), "digest") {
		return "", fmt.Errorf("unsupported auth scheme: %s", challenge)
	}
	kv := map[string]string{}
	for _, m := range digestKV.FindAllStringSubmatch(challenge, -1) {
		kv[strings.ToLower(m[1])] = m[2]
	}
	realm, nonce := kv["realm"], kv["nonce"]
	if realm == "" || nonce == "" {
		return "", fmt.Errorf("invalid digest challenge")
	}
	algorithm := strings.ToUpper(kv["algorithm"])
	if algorithm == "" {
		algorithm = "MD5"
	}
	if algorithm != "MD5" {
		return "", fmt.Errorf("unsupported digest algorithm: %s", algorithm)
	}
	qop := strings.TrimSpace(kv["qop"])
	nc := "00000001"
	cnonce := randomHex(8)
	ha1 := md5Hex(fmt.Sprintf("%s:%s:%s", user, realm, pass))
	ha2 := md5Hex(fmt.Sprintf("%s:%s", method, uri))
	var response string
	if qop != "" {
		qop = "auth"
		response = md5Hex(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2))
	} else {
		response = md5Hex(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))
	}
	parts := []string{
		fmt.Sprintf(`username="%s"`, user),
		fmt.Sprintf(`realm="%s"`, realm),
		fmt.Sprintf(`nonce="%s"`, nonce),
		fmt.Sprintf(`uri="%s"`, uri),
		fmt.Sprintf(`response="%s"`, response),
		"algorithm=MD5",
	}
	if qop != "" {
		parts = append(parts, "qop=auth", "nc="+nc, fmt.Sprintf(`cnonce="%s"`, cnonce))
	}
	if opaque := kv["opaque"]; opaque != "" {
		parts = append(parts, fmt.Sprintf(`opaque="%s"`, opaque))
	}
	return "Digest " + strings.Join(parts, ", "), nil
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func hikISAPITimeout() time.Duration { return 8 * time.Second }
