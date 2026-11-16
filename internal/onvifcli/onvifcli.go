// Package onvifcli 提供与具体基础设施无关的 ONVIF 发现/端点解析工具。
package onvifcli

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/0x524a/onvif-go/discovery"
)

// Discover 在局域网内发现 ONVIF 设备。
func Discover(ctx context.Context, timeoutSec int) ([]*discovery.Device, error) {
	if timeoutSec <= 0 {
		timeoutSec = 5
	}
	return discovery.Discover(ctx, time.Duration(timeoutSec)*time.Second)
}

// ParseEndpoint 归一化 ONVIF 设备端点，返回 host、port 与完整 endpoint。
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
