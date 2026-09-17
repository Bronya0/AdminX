package service

import (
	"testing"
	"time"

	"adminx/internal/model"
	"adminx/internal/repository"
)

// TestMonitorService_GetResources 采集结果必须满足前端契约（嵌套结构 + 数组不可为 null）。
func TestMonitorService_GetResources(t *testing.T) {
	svc := NewMonitorService(nil, nil) // 无 repo：只测采集形状
	res := svc.GetResources()

	if res.Timestamp == 0 {
		t.Error("timestamp 缺失（前端按 Unix 秒渲染）")
	}
	if res.CPU.Count == 0 {
		t.Error("cpu.count 应为正数")
	}
	if res.Disk == nil {
		t.Error("disk 必须是数组（null 会让前端表格报错）")
	}
	if res.Memory.Total == 0 {
		t.Error("memory.total 缺失")
	}
}

// TestMonitorService_SampleThrottle 采样节流：最小间隔内重复拉取只落一条。
func TestMonitorService_SampleThrottle(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMonitorService(repository.NewMonitorRepo(db), nil)

	svc.CollectResources()
	svc.CollectResources()

	var count int64
	if err := db.Model(&model.MonitorSample{}).Count(&count).Error; err != nil {
		t.Fatalf("统计采样失败: %v", err)
	}
	if count != 1 {
		t.Errorf("1 分钟内重复采样应只落 1 条，实际 %d 条", count)
	}
}

// TestMonitorService_History 历史趋势：分桶聚合、时间升序、参数校验。
func TestMonitorService_History(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewMonitorRepo(db)
	svc := NewMonitorService(repo, nil)

	now := time.Now()
	for i := 0; i < 5; i++ {
		if err := repo.Create(&model.MonitorSample{
			CPUPercent:    float64(10 * (i + 1)),
			MemoryPercent: 50,
			DiskPercent:   60,
			DiskReadMBps:  1.5,
			DiskWriteMBps: 0.5,
			CreatedAt:     now.Add(-time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("写入采样失败: %v", err)
		}
	}

	h, err := svc.History("1h", "")
	if err != nil {
		t.Fatalf("History 失败: %v", err)
	}
	if h.Interval != "1m" {
		t.Errorf("1h 窗口缺省 interval = %s, want 1m", h.Interval)
	}
	if len(h.Points) != 5 {
		t.Fatalf("每个 1 分钟桶一个点，期望 5 个，实际 %d", len(h.Points))
	}
	for i := 1; i < len(h.Points); i++ {
		if h.Points[i].Timestamp < h.Points[i-1].Timestamp {
			t.Error("历史点未按时间升序")
		}
	}
	if h.Points[0].DiskReadMBps != 1.5 || h.Points[0].MemoryPercent != 50 {
		t.Errorf("聚合值不对: %+v", h.Points[0])
	}

	// 7 天窗口缺省按 1h 聚合
	h7, err := svc.History("7d", "")
	if err != nil {
		t.Fatalf("History(7d) 失败: %v", err)
	}
	if h7.Interval != "1h" {
		t.Errorf("7d 窗口缺省 interval = %s, want 1h", h7.Interval)
	}

	// 参数校验
	if _, err := svc.History("lately", ""); err == nil {
		t.Error("非法 range 应报错")
	}
	if _, err := svc.History("1h", "2m"); err == nil {
		t.Error("非法 interval 应报错")
	}
}
