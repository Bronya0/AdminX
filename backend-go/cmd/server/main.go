// Package main 是 DjangoAdminX Go 后端的入口。
//
// 当前为骨架占位实现：仅打印版本信息后退出，不启动任何服务。
// 后续将逐步填充 HTTP server、路由、数据库连接、JWT 认证等模块，
// 目标是对齐 backend-django 的功能。
package main

import (
	"fmt"
	"os"
)

// version 在构建时可通过 -ldflags "-X main.version=..." 注入。
var version = "0.0.1-dev"

func main() {
	fmt.Printf("DjangoAdminX Go backend %s\n", version)
	fmt.Println()
	fmt.Println("当前为骨架占位，尚未实现业务逻辑。计划模块：")
	fmt.Println("  - HTTP server (net/http 或 gin/echo)")
	fmt.Println("  - 路由与中间件 (JWT 认证、RBAC、限流)")
	fmt.Println("  - 数据库 (GORM + PostgreSQL)")
	fmt.Println("  - Redis 缓存与调度器 leader 锁")
	fmt.Println("  - WebSocket (实时日志/通知)")
	fmt.Println("  - 审计日志、配置中心、文件中心等业务模块")
	fmt.Println()
	fmt.Println("对应 Django 实现见 ../backend-django/")
	os.Exit(0)
}
