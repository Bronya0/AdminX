# DjangoAdminX Go Backend

DjangoAdminX 的 Go 语言重写实现。目标是逐步对齐 [`../backend-django/`](../backend-django/) 的功能，提供更高性能、更低资源占用的后端服务。

## 当前状态

🚧 **骨架阶段** —— 仅有目录结构和占位代码，未实现任何业务逻辑。

## 目录结构

```
backend-go/
├── go.mod                  # 模块: djangoadminx
├── cmd/
│   └── server/
│       └── main.go         # 服务入口
├── internal/               # 私有业务代码（不可被外部 import）
│   ├── config/             # 配置加载
│   ├── handler/            # HTTP handler（控制器层）
│   ├── service/            # 业务逻辑（服务层）
│   ├── repository/         # 数据访问（仓储层）
│   ├── model/              # 数据模型 / 实体
│   └── middleware/         # 中间件（JWT、RBAC、限流、审计）
├── pkg/                    # 可复用包（可被外部 import）
├── configs/
│   └── config.example.yaml # 配置示例
└── docs/                   # 文档
```

## 功能模块映射（规划中）

| Django 模块 | Go 对应 | 优先级 |
|------------|---------|--------|
| accounts（用户/角色/JWT） | internal/handler + service | P0 |
| menu（动态菜单） | internal/handler + service | P0 |
| config_center（配置中心） | internal/handler + service | P1 |
| audit（审计日志） | internal/middleware + service | P1 |
| jobs（定时任务 + HA） | internal/service + 调度器锁 | P1 |
| monitor（系统监控） | internal/handler | P2 |
| notification（通知中心） | internal/handler + service | P2 |
| file_center（文件中心） | internal/handler + service | P2 |
| common（RBAC/限流/Channel Layer） | internal/middleware + pkg | P0 |

## 开发命令

```bash
# 构建（当前仅打印版本信息）
cd backend-go
go build -o bin/server ./cmd/server
./bin/server

# 带版本注入构建
go build -ldflags "-X main.version=1.0.0" -o bin/server ./cmd/server

# 运行测试（待实现）
go test ./...

# 格式化与检查
go fmt ./...
go vet ./...
```

## 技术选型（待定）

以下为候选方案，正式开发时确定：

- **Web 框架**: Gin / Echo / 标准库 net/http
- **ORM**: GORM / sqlx
- **配置**: viper
- **日志**: zap / slog
- **JWT**: golang-jwt/jwt
- **Redis**: go-redis
- **WebSocket**: gorilla/websocket / nhooyr/websocket
- **定时任务**: robfig/cron + Redis 分布式锁（对齐 Django 的 Active-Standby HA 方案）

## 与前端协作

前端 [`../adminx-ui/`](../adminx-ui/) 与具体后端实现解耦，仅通过 HTTP/WebSocket + JWT 交互。
Go 后端实现 API 兼容后，前端无需改动即可切换。
