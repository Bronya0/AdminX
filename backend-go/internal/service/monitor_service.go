package service

import (
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// MonitorService 系统资源监控（gopsutil）。
type MonitorService struct{}

func NewMonitorService() *MonitorService { return &MonitorService{} }

// Resources 系统资源快照。
type Resources struct {
	CPUUsage    float64 `json:"cpu_usage"`     // %
	CPUCount    int     `json:"cpu_count"`
	MemTotal    uint64  `json:"mem_total"`     // bytes
	MemUsed     uint64  `json:"mem_used"`
	MemPercent  float64 `json:"mem_percent"`
	DiskTotal   uint64  `json:"disk_total"`    // /
	DiskUsed    uint64  `json:"disk_used"`
	DiskPercent float64 `json:"disk_percent"`
	NetSent     uint64  `json:"net_sent"`      // bytes (累计)
	NetRecv     uint64  `json:"net_recv"`
	GoRoutines  int     `json:"go_routines"`
}

// GetResources 获取系统资源快照。
func (s *MonitorService) GetResources() (*Resources, error) {
	r := &Resources{GoRoutines: runtime.NumGoroutine()}

	// CPU
	if percents, err := cpu.Percent(0, false); err == nil && len(percents) > 0 {
		r.CPUUsage = percents[0]
	}
	if counts, err := cpu.Counts(true); err == nil {
		r.CPUCount = counts
	}

	// 内存
	if vm, err := mem.VirtualMemory(); err == nil {
		r.MemTotal = vm.Total
		r.MemUsed = vm.Used
		r.MemPercent = vm.UsedPercent
	}

	// 磁盘
	if du, err := disk.Usage("/"); err == nil {
		r.DiskTotal = du.Total
		r.DiskUsed = du.Used
		r.DiskPercent = du.UsedPercent
	}

	// 网络
	if counters, err := net.IOCounters(false); err == nil && len(counters) > 0 {
		r.NetSent = counters[0].BytesSent
		r.NetRecv = counters[0].BytesRecv
	}

	return r, nil
}

// NetStat 网络连接统计。
type NetStat struct {
	Established int `json:"established"`
	Listen      int `json:"listen"`
	TimeWait    int `json:"time_wait"`
	Total       int `json:"total"`
}

// GetNetStat 获取网络连接统计。
func (s *MonitorService) GetNetStat() *NetStat {
	stat := &NetStat{}
	if conns, err := net.Connections("all"); err == nil {
		stat.Total = len(conns)
		for _, c := range conns {
			switch c.Status {
			case "ESTABLISHED":
				stat.Established++
			case "LISTEN":
				stat.Listen++
			case "TIME_WAIT":
				stat.TimeWait++
			}
		}
	}
	return stat
}
