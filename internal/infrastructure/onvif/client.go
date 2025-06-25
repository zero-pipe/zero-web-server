package onvifinfra

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	onviflib "github.com/0x524a/onvif-go"
	"github.com/0x524a/onvif-go/discovery"
)

type ClientFactory struct {
	timeout time.Duration
}

func NewClientFactory(timeoutSec int) *ClientFactory {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	return &ClientFactory{timeout: time.Duration(timeoutSec) * time.Second}
}

func (f *ClientFactory) NewClient(endpoint, username, password string) (*onviflib.Client, error) {
	opts := []onviflib.ClientOption{
		onviflib.WithTimeout(f.timeout),
		onviflib.WithInsecureSkipVerify(),
	}
	if username != "" {
		opts = append(opts, onviflib.WithCredentials(username, password))
	}
	return onviflib.NewClient(endpoint, opts...)
}

func (f *ClientFactory) Initialize(ctx context.Context, endpoint, username, password string) (*onviflib.Client, error) {
	client, err := f.NewClient(endpoint, username, password)
	if err != nil {
		return nil, err
	}
	if err := client.Initialize(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

func Discover(ctx context.Context, timeoutSec int) ([]*discovery.Device, error) {
	if timeoutSec <= 0 {
		timeoutSec = 5
	}
	return discovery.Discover(ctx, time.Duration(timeoutSec)*time.Second)
}

func ParseEndpoint(raw string) (host string, port int, endpoint string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, "", fmt.Errorf("empty endpoint")
	}

	if !strings.HasPrefix(raw, "http") {
		if !strings.Contains(raw, ":") {
			raw = raw + ":80"
		}
		raw = "http://" + raw + "/onvif/device_service"
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", 0, "", err
	}

	host = u.Hostname()
	port = 80
	if u.Port() != "" {
		fmt.Sscanf(u.Port(), "%d", &port)
	}
	return host, port, raw, nil
}
