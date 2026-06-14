// Package logger 提供基于 slog 的结构化日志初始化。
//
// 生产用 JSON 格式便于采集；开发用 text 格式便于阅读。
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Init 根据 level/format 初始化全局 slog logger。
//   - format: "json" → JSON 输出；其他 → text 输出
//   - level: "debug" / "info" / "warn" / "error"
func Init(level, format string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	if strings.ToLower(format) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
