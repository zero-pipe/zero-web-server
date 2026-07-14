package ops

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

const metricsHistory = 30

// Metrics 资源监控 CPU/内存/网络时序 + 磁盘快照（内存环形缓冲）。
type Metrics struct {
	mu sync.RWMutex

	cpu []map[string]any
	mem []map[string]any
	net []map[string]any
	disk []map[string]any
	netTotal float64

	prevNetTime time.Time
	prevNetIn   uint64
	prevNetOut  uint64
}

var DefaultMetrics = NewMetrics()

func NewMetrics() *Metrics {
	return &Metrics{
		cpu:      make([]map[string]any, 0, metricsHistory),
		mem:      make([]map[string]any, 0, metricsHistory),
		net:      make([]map[string]any, 0, metricsHistory),
		disk:     []map[string]any{},
		netTotal: 1000,
	}
}

// Start 后台采样；interval 建议 2s。
func (m *Metrics) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	m.sampleOnce() // 先采一次，避免首屏空图
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				m.sampleOnce()
			}
		}
	}()
}

func (m *Metrics) Snapshot() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]any{
		"cpu":      cloneRows(m.cpu),
		"mem":      cloneRows(m.mem),
		"net":      cloneRows(m.net),
		"disk":     cloneRows(m.disk),
		"netTotal": m.netTotal,
	}
}

func cloneRows(in []map[string]any) []map[string]any {
	out := make([]map[string]any, len(in))
	copy(out, in)
	return out
}

func (m *Metrics) sampleOnce() {
	now := time.Now().Local().Format("2006-01-02 15:04:05")

	cpuFrac := sampleCPU()
	memFrac := sampleMem()
	inMbps, outMbps, total := m.sampleNet()
	diskRows := sampleDisk()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.cpu = appendPoint(m.cpu, map[string]any{"time": now, "data": cpuFrac})
	m.mem = appendPoint(m.mem, map[string]any{"time": now, "data": memFrac})
	m.net = appendPoint(m.net, map[string]any{"time": now, "in": inMbps, "out": outMbps})
	if len(diskRows) > 0 {
		m.disk = diskRows
	}
	if total > 0 {
		m.netTotal = total
	}
}

func appendPoint(buf []map[string]any, point map[string]any) []map[string]any {
	buf = append(buf, point)
	if len(buf) > metricsHistory {
		buf = buf[len(buf)-metricsHistory:]
	}
	return buf
}

func sampleCPU() float64 {
	// Interval=0 使用上次调用以来的差值；首次可能为 0
	vals, err := cpu.Percent(0, false)
	if err != nil || len(vals) == 0 {
		return 0
	}
	v := vals[0] / 100.0
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func sampleMem() float64 {
	vm, err := mem.VirtualMemory()
	if err != nil || vm.Total == 0 {
		return 0
	}
	v := float64(vm.Total-vm.Available) / float64(vm.Total)
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (m *Metrics) sampleNet() (inMbps, outMbps, totalMbps float64) {
	counters, err := net.IOCounters(false)
	if err != nil || len(counters) == 0 {
		return 0, 0, m.netTotal
	}
	c := counters[0]
	now := time.Now()
	if !m.prevNetTime.IsZero() {
		sec := now.Sub(m.prevNetTime).Seconds()
		if sec > 0 {
			var dIn, dOut uint64
			if c.BytesRecv >= m.prevNetIn {
				dIn = c.BytesRecv - m.prevNetIn
			}
			if c.BytesSent >= m.prevNetOut {
				dOut = c.BytesSent - m.prevNetOut
			}
			inMbps = float64(dIn) * 8 / sec / 1e6
			outMbps = float64(dOut) * 8 / sec / 1e6
		}
	}
	m.prevNetTime = now
	m.prevNetIn = c.BytesRecv
	m.prevNetOut = c.BytesSent

	totalMbps = m.netTotal
	if totalMbps <= 0 {
		totalMbps = 1000
	}
	return inMbps, outMbps, totalMbps
}

func sampleDisk() []map[string]any {
	parts, err := disk.Partitions(false)
	if err != nil {
		return nil
	}
	want := map[string]bool{}
	switch runtime.GOOS {
	case "windows":
		want["C:"] = true
	default:
		want["/"] = true
		want["/home"] = true
	}
	out := make([]map[string]any, 0, 2)
	seen := map[string]bool{}
	for _, p := range parts {
		mount := p.Mountpoint
		if runtime.GOOS == "windows" {
			if len(mount) >= 2 && mount[1] == ':' {
				mount = mount[:2]
			}
		}
		if !want[mount] || seen[mount] {
			continue
		}
		u, err := disk.Usage(p.Mountpoint)
		if err != nil || u.Total == 0 {
			continue
		}
		seen[mount] = true
		out = append(out, map[string]any{
			"path": mount,
			"use":  float64(u.Used) / (1024 * 1024 * 1024),
			"free": float64(u.Free) / (1024 * 1024 * 1024),
		})
	}
	return out
}
