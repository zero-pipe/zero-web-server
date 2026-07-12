package ops

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

var processStart = time.Now()

// PlatformInfo 平台信息：map[分组]map[字段]string，供 GET /api/server/info。
func PlatformInfo(version, serverID string, serverPort int, scheme, requestHost string) map[string]map[string]string {
	out := map[string]map[string]string{
		"硬件信息": {},
		"操作系统": {},
		"平台信息": {},
		"文档地址": {},
	}

	hw := out["硬件信息"]
	if infos, err := cpu.Info(); err == nil && len(infos) > 0 {
		hw["CPU"] = strings.TrimSpace(infos[0].ModelName)
	} else {
		hw["CPU"] = runtime.GOARCH
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		hw["内存"] = formatByte(vm.Used) + "/" + formatByte(vm.Total)
	} else {
		hw["内存"] = "-"
	}
	hw["制造商"] = "-"
	hw["产品名称"] = runtime.GOOS + "/" + runtime.GOARCH
	hw["网卡"] = localIPv4s()

	osMap := out["操作系统"]
	if hi, err := host.Info(); err == nil {
		osMap["名称"] = strings.TrimSpace(hi.Platform + " " + hi.PlatformVersion)
		osMap["类型"] = hi.OS
	} else {
		osMap["名称"] = runtime.GOOS
		osMap["类型"] = runtime.GOOS
	}

	plat := out["平台信息"]
	plat["版本"] = version
	plat["服务标识"] = serverID
	plat["监听端口"] = fmt.Sprintf("%d", serverPort)
	plat["启动时间"] = processStart.Local().Format("2006-01-02 15:04:05")
	plat["运行时长"] = formatDuration(time.Since(processStart))
	plat["Go版本"] = runtime.Version()
	plat["DOCKER环境"] = "否"
	if _, err := os.Stat("/.dockerenv"); err == nil {
		plat["DOCKER环境"] = "是"
	}

	doc := out["文档地址"]
	doc["项目地址"] = "https://github.com/zero-pipe/zero-web-kit"
	if scheme == "" {
		scheme = "http"
	}
	if requestHost != "" {
		doc["本机地址"] = fmt.Sprintf("%s://%s", scheme, requestHost)
	}
	return out
}

func localIPv4s() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "-"
	}
	var ips []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			ips = append(ips, ip.String())
		}
	}
	if len(ips) == 0 {
		return "-"
	}
	return strings.Join(ips, ",")
}

func formatByte(n uint64) string {
	const unit = 1024.0
	v := float64(n)
	if v < unit {
		return fmt.Sprintf("%.2fB", v)
	}
	v /= unit
	if v < unit {
		return fmt.Sprintf("%.2fKB", v)
	}
	v /= unit
	if v < unit {
		return fmt.Sprintf("%.2fMB", v)
	}
	v /= unit
	if v < unit {
		return fmt.Sprintf("%.2fGB", v)
	}
	return fmt.Sprintf("%.2fTB", v/unit)
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d小时%d分%d秒", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%d分%d秒", m, s)
	}
	return fmt.Sprintf("%d秒", s)
}
