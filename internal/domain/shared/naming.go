package shared

import (
	"fmt"
	"strings"
)

// DefaultChannelName 生成用户可见的通道默认名称：设备名称_01、设备名称_02。
// 用户可在通道编辑中改名；协议同步不得覆盖已存在的名称。
func DefaultChannelName(deviceName string, index int) string {
	base := strings.TrimSpace(deviceName)
	if base == "" {
		base = "通道"
	}
	if index < 1 {
		index = 1
	}
	return fmt.Sprintf("%s_%02d", base, index)
}
