package deviceapp

import (
	"fmt"
	"strings"
	"time"
	"zero-web-server/internal/port"
)

func nowStr() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// channelDisplayName 统一通道展示名：设备名_channel_N
func channelDisplayName(deviceName string, index int) string {
	base := strings.TrimSpace(deviceName)
	if base == "" {
		base = "device"
	}
	return fmt.Sprintf("%s_channel_%d", base, index)
}

// Ensure Service implements sipinfra.DeviceService
var _ port.DeviceLifecycle = (*Service)(nil)
