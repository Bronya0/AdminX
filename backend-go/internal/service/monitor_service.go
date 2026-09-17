package service

import (
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"gorm.io/gorm"

	apperr "adminx/pkg/errors"

	"adminx/internal/model"
	"adminx/internal/repository"
)

// 历史采样参数。
const (
	// sampleInterval 最小采样间隔（前端标注「历史采样: 1 分钟」）。
	sampleInterval = time.Minute
	// sampleRetention 历史保留窗口。
	sampleRetention = 7 * 24 * time.Hour
)

// MonitorService 系统资源监控（gopsutil）+ 历史采样。
type MonitorService struct {
	repo *repository.MonitorRepo
	log  *slog.Logger

	// 上次累计磁盘 IO 计数：gopsutil 给的是累计值，读写速率需要相邻两次的差。
	mu         sync.Mutex
	lastIOAt   time.Time
	lastReadB  uint64
	lastWriteB uint64
}

// NewMonitorService 构造。repo 为 nil 时不做历史采样（单测/无库场景）；logger 可为 nil。
func NewMonitorService(repo *repository.MonitorRepo, logger *slog.Logger) *MonitorService {
	return &MonitorService{repo: repo, log: logger}
}

// CPUInfo CPU 信息。
type CPUInfo struct {
	Percent float64 `json:"percent"`
	Count   int     `json:"count"`
}

// MemoryInfo 内存信息（bytes）。
type MemoryInfo struct {
	Total     uint64  `json:"total"`
	Available uint64  `json:"available"`
	Used      uint64  `json:"used"`
	Free      uint64  `json:"free"`
	Percent   float64 `json:"percent"`
}

// DiskInfo 单个分区。
type DiskInfo struct {
	Device     string  `json:"device"`
	Mountpoint string  `json:"mountpoint"`
	Fstype     string  `json:"fstype"`
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Free       uint64  `json:"free"`
	Percent    float64 `json:"percent"`
}

// DiskIOInfo 磁盘累计 IO 计数。
type DiskIOInfo struct {
	ReadCount  uint64 `json:"read_count"`
	WriteCount uint64 `json:"write_count"`
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
	ReadTime   uint64 `json:"read_time"`
	WriteTime  uint64 `json:"write_time"`
}

// NetworkIOInfo 网络累计计数。
type NetworkIOInfo struct {
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	Errin       uint64 `json:"errin"`
	Errout      uint64 `json:"errout"`
	Dropin      uint64 `json:"dropin"`
	Dropout     uint64 `json:"dropout"`
}

// Resources 系统资源快照（结构与字段名对齐前端 types 的 SystemResources）。
type Resources struct {
	CPU       CPUInfo       `json:"cpu"`
	Memory    MemoryInfo    `json:"memory"`
	Disk      []DiskInfo    `json:"disk"`
	DiskIO    DiskIOInfo    `json:"disk_io"`
	NetworkIO NetworkIOInfo `json:"network_io"`
	LoadAvg   []float64     `json:"load_avg"` // Windows 无 load 概念时为 null
	Timestamp int64         `json:"timestamp"` // Unix 秒
}

// NetStat 网络连接统计（键名对齐前端 NetstatInfo）。
type NetStat struct {
	Listen      int `json:"LISTEN"`
	Established int `json:"ESTABLISHED"`
	TimeWait    int `json:"TIME_WAIT"`
	CloseWait   int `json:"CLOSE_WAIT"`
	Other       int `json:"OTHER"`
}

// GetResources 采集实时资源快照（不写历史采样；采样见 CollectResources）。
func (s *MonitorService) GetResources() *Resources {
	res := &Resources{
		Disk:      []DiskInfo{},
		Timestamp: time.Now().Unix(),
	}

	if percents, err := cpu.Percent(0, false); err == nil && len(percents) > 0 {
		res.CPU.Percent = round2(percents[0])
	}
	if counts, err := cpu.Counts(true); err == nil {
		res.CPU.Count = counts
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		res.Memory = MemoryInfo{
			Total: vm.Total, Available: vm.Available, Used: vm.Used, Free: vm.Free,
			Percent: round2(vm.UsedPercent),
		}
	}
	res.Disk = diskInfos()
	res.DiskIO = diskIOInfo()
	res.NetworkIO = networkIOInfo()
	if avg, err := load.Avg(); err == nil && avg != nil {
		res.LoadAvg = []float64{round2(avg.Load1), round2(avg.Load5), round2(avg.Load15)}
	}
	return res
}

// CollectResources 采集快照，并按最小间隔补一条历史采样。
// 采样失败不回滚（实时数据优先），但会记 warn 日志——否则 schema/写库问题会静默丢历史。
func (s *MonitorService) CollectResources() *Resources {
	res := s.GetResources()
	if s.repo == nil {
		return res
	}
	if err := s.recordSample(res); err != nil && s.log != nil {
		s.log.Warn("资源采样写入失败，历史趋势会缺数据", "error", err)
	}
	return res
}

// GetNetStat 网络连接统计（按 TCP 状态归类，其余计入 OTHER）。
func (s *MonitorService) GetNetStat() *NetStat {
	stat := &NetStat{}
	conns, err := net.Connections("all")
	if err != nil {
		return stat
	}
	for _, c := range conns {
		switch c.Status {
		case "LISTEN":
			stat.Listen++
		case "ESTABLISHED":
			stat.Established++
		case "TIME_WAIT":
			stat.TimeWait++
		case "CLOSE_WAIT":
			stat.CloseWait++
		default:
			stat.Other++
		}
	}
	return stat
}

// HistoryPoint 历史趋势点（timestamp 为 Unix 秒，前端按秒渲染）。
type HistoryPoint struct {
	Timestamp     int64   `json:"timestamp"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	DiskReadMBps  float64 `json:"disk_read_mbps"`
	DiskWriteMBps float64 `json:"disk_write_mbps"`
}

// History 资源历史趋势（按区间聚合为等间隔桶，桶内取均值）。
type History struct {
	Range    string         `json:"range"`
	Interval string         `json:"interval"`
	From     int64          `json:"from"`
	To       int64          `json:"to"`
	Points   []HistoryPoint `json:"points"`
}

// History 返回时间窗内的资源趋势。range: 1h/6h/24h/7d；interval: 1m/5m/15m/1h（缺省按窗口推导）。
func (s *MonitorService) History(rangeStr, intervalStr string) (*History, error) {
	window, err := parseHistoryRange(rangeStr)
	if err != nil {
		return nil, err
	}
	interval, step, err := parseHistoryInterval(intervalStr, window)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	from := now.Add(-window)

	if s.repo == nil {
		return &History{Range: rangeStr, Interval: interval, From: from.Unix(), To: now.Unix(), Points: []HistoryPoint{}}, nil
	}
	// 历史查询频率低，顺手清理窗口外的过期采样
	_ = s.repo.DeleteBefore(now.Add(-sampleRetention))

	samples, err := s.repo.ListBetween(from, now)
	if err != nil {
		return nil, apperr.Wrap(500, "读取资源历史失败", err)
	}

	// Go 侧分桶聚合（postgres / sqlite 通用，不依赖方言时间函数）
	type bucket struct {
		cpu, mem, disk, read, write float64
		count                       int
	}
	buckets := make(map[int64]*bucket, len(samples))
	stepSec := int64(step.Seconds())
	for i := range samples {
		key := samples[i].CreatedAt.Unix() / stepSec * stepSec
		b := buckets[key]
		if b == nil {
			b = &bucket{}
			buckets[key] = b
		}
		b.cpu += samples[i].CPUPercent
		b.mem += samples[i].MemoryPercent
		b.disk += samples[i].DiskPercent
		b.read += samples[i].DiskReadMBps
		b.write += samples[i].DiskWriteMBps
		b.count++
	}

	points := make([]HistoryPoint, 0, len(buckets))
	for ts, b := range buckets {
		if b.count == 0 {
			continue
		}
		n := float64(b.count)
		points = append(points, HistoryPoint{
			Timestamp:     ts,
			CPUPercent:    round2(b.cpu / n),
			MemoryPercent: round2(b.mem / n),
			DiskPercent:   round2(b.disk / n),
			DiskReadMBps:  round2(b.read / n),
			DiskWriteMBps: round2(b.write / n),
		})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Timestamp < points[j].Timestamp })

	return &History{
		Range:    rangeStr,
		Interval: interval,
		From:     from.Unix(),
		To:       now.Unix(),
		Points:   points,
	}, nil
}

// recordSample 落一条采样：距上次采样不足 sampleInterval 时跳过；同时更新 IO 速率基准。
func (s *MonitorService) recordSample(res *Resources) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	last, err := s.repo.Latest()
	if err == nil && last != nil && now.Sub(last.CreatedAt) < sampleInterval {
		return nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	// 磁盘读写速率：相邻两次累计计数的差 / 间隔
	var readMBps, writeMBps float64
	if !s.lastIOAt.IsZero() {
		elapsed := now.Sub(s.lastIOAt).Seconds()
		if elapsed > 0 {
			readMBps = counterRateMBps(s.lastReadB, res.DiskIO.ReadBytes, elapsed)
			writeMBps = counterRateMBps(s.lastWriteB, res.DiskIO.WriteBytes, elapsed)
		}
	}
	s.lastIOAt = now
	s.lastReadB = res.DiskIO.ReadBytes
	s.lastWriteB = res.DiskIO.WriteBytes

	return s.repo.Create(&model.MonitorSample{
		CPUPercent:    res.CPU.Percent,
		MemoryPercent: res.Memory.Percent,
		DiskPercent:   diskUsagePercent(res.Disk),
		DiskReadMBps:  readMBps,
		DiskWriteMBps: writeMBps,
	})
}

// parseHistoryRange range 参数 → 时间窗。
func parseHistoryRange(rangeStr string) (time.Duration, error) {
	switch rangeStr {
	case "", "1h":
		return time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "24h":
		return 24 * time.Hour, nil
	case "7d":
		return 7 * 24 * time.Hour, nil
	}
	return 0, apperr.New(400, "不支持的 range（可选 1h / 6h / 24h / 7d）")
}

// parseHistoryInterval interval 参数 → (区间名, 时长)。
// 未指定时按窗口推导，保证点数在图表可读范围内（≤ ~170 点）。
func parseHistoryInterval(intervalStr string, window time.Duration) (string, time.Duration, error) {
	switch intervalStr {
	case "":
		// 按窗口推导
	case "1m":
		return intervalStr, time.Minute, nil
	case "5m":
		return intervalStr, 5 * time.Minute, nil
	case "15m":
		return intervalStr, 15 * time.Minute, nil
	case "1h":
		return intervalStr, time.Hour, nil
	default:
		return "", 0, apperr.New(400, "不支持的 interval（可选 1m / 5m / 15m / 1h）")
	}

	switch {
	case window <= time.Hour:
		return "1m", time.Minute, nil
	case window <= 6*time.Hour:
		return "5m", 5 * time.Minute, nil
	case window <= 24*time.Hour:
		return "15m", 15 * time.Minute, nil
	default:
		return "1h", time.Hour, nil
	}
}

// diskInfos 分区列表（跳过读取失败的分区，按挂载点去重并排序）。
func diskInfos() []DiskInfo {
	parts, err := disk.Partitions(false)
	if err != nil {
		return []DiskInfo{}
	}
	out := make([]DiskInfo, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, p := range parts {
		if seen[p.Mountpoint] {
			continue
		}
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil || usage == nil || usage.Total == 0 {
			continue
		}
		seen[p.Mountpoint] = true
		out = append(out, DiskInfo{
			Device:     p.Device,
			Mountpoint: p.Mountpoint,
			Fstype:     p.Fstype,
			Total:      usage.Total,
			Used:       usage.Used,
			Free:       usage.Free,
			Percent:    round2(usage.UsedPercent),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Mountpoint < out[j].Mountpoint })
	return out
}

// diskIOInfo 汇总所有设备的磁盘累计 IO 计数。
func diskIOInfo() DiskIOInfo {
	counters, err := disk.IOCounters()
	if err != nil {
		return DiskIOInfo{}
	}
	var info DiskIOInfo
	for _, c := range counters {
		info.ReadCount += c.ReadCount
		info.WriteCount += c.WriteCount
		info.ReadBytes += c.ReadBytes
		info.WriteBytes += c.WriteBytes
		info.ReadTime += c.ReadTime
		info.WriteTime += c.WriteTime
	}
	return info
}

// networkIOInfo 网络累计计数（false = 汇总所有网卡）。
func networkIOInfo() NetworkIOInfo {
	counters, err := net.IOCounters(false)
	if err != nil || len(counters) == 0 {
		return NetworkIOInfo{}
	}
	c := counters[0]
	return NetworkIOInfo{
		BytesSent:   c.BytesSent,
		BytesRecv:   c.BytesRecv,
		PacketsSent: c.PacketsSent,
		PacketsRecv: c.PacketsRecv,
		Errin:       c.Errin,
		Errout:      c.Errout,
		Dropin:      c.Dropin,
		Dropout:     c.Dropout,
	}
}

// diskUsagePercent 分区整体使用率（与前端 getDiskUsagePercent 同口径：Σused / Σtotal）。
func diskUsagePercent(disks []DiskInfo) float64 {
	var total, used uint64
	for _, d := range disks {
		total += d.Total
		used += d.Used
	}
	if total == 0 {
		return 0
	}
	return round2(float64(used) / float64(total) * 100)
}

// counterRateMBps 累计计数器差值 → MB/s（计数器回绕/重启导致倒退时按 0 处理）。
func counterRateMBps(prev, cur uint64, elapsedSec float64) float64 {
	if cur < prev || elapsedSec <= 0 {
		return 0
	}
	return round2(float64(cur-prev) / elapsedSec / 1024 / 1024)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
