package service

import (
	"log/slog"
	"sync"
	"time"
)

// JobHandler 任务处理函数类型（无参，返回 string 或 error）。
type JobHandler func() (string, error)

// handlerRegistry 全局任务 handler 注册表。
// init-data 注册后，调度器通过 handler 路径字符串查找。
var (
	handlerRegistry = make(map[string]JobHandler)
	registryMu      sync.RWMutex
)

// RegisterHandler 注册任务 handler。
// name 格式建议: "jobs.SampleTask"。
func RegisterHandler(name string, h JobHandler) {
	registryMu.Lock()
	defer registryMu.Unlock()
	handlerRegistry[name] = h
}

// GetHandler 按名称查找 handler，找不到返回 nil。
func GetHandler(name string) JobHandler {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return handlerRegistry[name]
}

// SampleTask 示例任务：返回当前时间（供测试调度器）。
func SampleTask() (string, error) {
	return "executed at " + time.Now().Format(time.RFC3339), nil
}

// init 注册内置任务。
func init() {
	RegisterHandler("jobs.SampleTask", SampleTask)
	RegisterHandler("jobs.NtpSyncPlaceholder", func() (string, error) {
		// NTP 同步占位（实际实现需要 root 权限调用系统命令）
		slog.Default().Info("NTP 同步任务被触发（占位实现）")
		return "ntp sync placeholder", nil
	})
}
