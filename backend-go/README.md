# AdminX Go Backend

AdminX 的 Go 语言实现，与 [`../backend-django/`](../backend-django/) API 完全兼容，前端 [`../adminx-ui/`](../adminx-ui/) 无需改动即可切换。

## 当前状态

✅ **核心模块全部完成**（67 个 Go 文件，~7400 行），`go build` + `go vet` 通过。

## 技术栈

| 库 | 用途 |
|---|---|
| [gin](https://github.com/gin-gonic/gin) | HTTP 框架 |
| [gorm](https://gorm.io) + postgres driver | ORM |
| [gocron v2](https://github.com/go-co-op/gocron) + [gocron-redis-lock](https://github.com/go-co-op/gocron-redis-lock) | 定时任务 + 任务级分布式锁（HA） |
| [golang-jwt/v5](https://github.com/golang-jwt/jwt) | JWT 认证（HS256 + 黑名单） |
| [go-redis/v9](https://github.com/redis/go-redis) | Redis 客户端 |
| [viper](https://github.com/spf13/viper) | 配置管理（yaml + 环境变量） |
| [gorilla/websocket](https://github.com/gorilla/websocket) | WebSocket 实时推送 |
| [gopsutil](https://github.com/shirou/gopsutil) | 系统资源监控 |
| `crypto/aes` + `crypto/cipher` | AES-GCM 配置加密 |
| `golang.org/x/crypto/bcrypt` | 密码哈希 |
| `log/slog` | 结构化日志（标准库） |

## 目录结构

```
backend-go/
├── cmd/
│   ├── server/main.go          # 服务入口（HTTP + 可选调度器）
│   └── init-data/main.go       # 种子数据（超级用户 + 默认角色 + 菜单）
├── internal/
│   ├── config/                 # viper 配置加载
│   ├── database/               # GORM + Postgres + AutoMigrate
│   ├── redis/                  # go-redis 初始化
│   ├── model/                  # GORM 模型（17 个）+ DTO
│   ├── repository/             # 数据访问层（11 个 repo）
│   ├── service/                # 业务逻辑（11 个 service + handler 注册表）
│   ├── handler/                # HTTP 处理器（14 个）
│   ├── middleware/             # JWT / RBAC / 限流 / 请求日志
│   ├── jwt/                    # JWT 签发/解析/黑名单
│   ├── router/                 # 路由注册（50+ 端点）
│   ├── scheduler/              # gocron 调度器 + Redis 分布式锁
│   ├── websocket/              # Hub + Redis pub/sub 跨实例广播
│   ├── captcha/                # 图片验证码
│   └── audit/                  # （预留）
├── pkg/
│   ├── response/               # 统一 {code,msg,data} 响应
│   ├── pagination/             # count/next/previous/results 分页
│   ├── logger/                 # slog 初始化
│   ├── crypto/                 # AES-GCM + bcrypt
│   └── errors/                 # AppError 错误类型
├── configs/
│   └── config.example.yaml     # 配置示例
├── go.mod / go.sum
└── README.md
```

## 已实现模块

| 模块 | 端点 | 说明 |
|---|---|---|
| **认证** | `/accounts/login/` `logout/` `refresh/` `introspect/` | JWT + 登录锁定 + 黑名单 + last_logout 二次失效 |
| **用户/角色/菜单** | `/accounts/users/` `/accounts/roles/` `/menu/` | 完整 CRUD + RBAC |
| **配置中心** | `/config/` + `by_group/` `get_value/` `groups/` | 5 种值类型 + AES-GCM 加密 + Redis 缓存 |
| **审计日志** | `/audit/` | create/update/delete 记录 |
| **定时任务** | `/jobs/` + `run_once/` `status/` `logs/` | gocron + Redis 任务级锁 HA |
| **通知中心** | `/notification/messages/` `webhooks/` | webhook HMAC-SHA256 签名外发 |
| **集群管理** | `/cluster/nodes/` `components/` | 节点 CRUD + 业务组件注册/心跳 |
| **文件中心** | `/files/upload/` `records/` | 本地存储 |
| **系统监控** | `/monitor/resources/` `netstat/` | CPU/内存/磁盘/网络（gopsutil） |
| **密码策略** | `/policy/policy/` `change-password/` | 策略校验 + 历史防重用 |
| **验证码** | `/captcha/captcha/` `verify/` | Redis 存储 |
| **通用** | `/common/health/` `dashboard/stats/` | 健康检查 + 仪表盘 |
| **WebSocket** | `/ws/log/` | Hub + Redis pub/sub 跨实例广播 |

## 快速开始

### 前置依赖

- Go 1.22+
- PostgreSQL 14+
- Redis 6+

### 配置

```bash
cp configs/config.example.yaml configs/config.yaml
# 编辑 config.yaml，填入数据库 DSN、Redis URL、JWT secret、AES key
```

### 初始化数据

```bash
go run ./cmd/init-data --username admin --password admin123
# 创建超级管理员 + 4 个默认角色（三权分立）+ 基础菜单
```

### 启动服务

```bash
# 仅 HTTP 服务
go run ./cmd/server

# HTTP + 内置调度器（--scheduler）
go run ./cmd/server --scheduler

# 或用脚本
bash scripts/dev.sh    # 开发（go run，热重载自行配 air）
bash scripts/build.sh  # 构建二进制到 bin/
```

启动后访问 `http://localhost:8000/adminx/api/v1/common/health/`。

### 前端联调

前端 `adminx-ui/` 的 vite dev proxy 默认指向 `:9999`。修改 `adminx-ui/vite.config.ts` 的 proxy target 为 `http://localhost:8000`，或把 Go 服务端口改为 9999。

## 架构设计

### 与 Python 实现的差异

| 维度 | Python 版 | Go 版 |
|---|---|---|
| 数据库 | 独立（不共享 schema） | 独立（AutoMigrate 从零建表） |
| 密码哈希 | pbkdf2_sha256 | bcrypt |
| 配置加密 | Fernet | AES-GCM（标准库） |
| 调度器 HA | Redis leader 锁（进程级） | gocron 任务级锁（多实例并行不同任务） |
| 响应格式 | `{code,msg,data}` HTTP 200 | 完全一致 |
| 路由前缀 | `/adminx/api/v1/` | 完全一致 |

### HA 部署

```
Nginx (负载均衡)
  ├── Go 实例 1 (HTTP + 调度器)
  ├── Go 实例 2 (HTTP)
  └── Go 实例 N (HTTP)
        │
   PostgreSQL + Redis (共享)
```

- HTTP 层无状态，任意实例可处理请求
- 调度器用 gocron `WithDistributedLocker`：所有实例都加载任务，但同一任务同时只在一个实例执行（Redis 锁）
- WebSocket 通过 Redis pub/sub 跨实例广播

### 关键安全设计

- JWT 防 `alg:none` 攻击（强制 HMAC 算法校验）
- 登出后 access token 通过 `last_logout` 二次失效（即使未过期）
- logout 校验 refresh token 归属（防 DoS）
- RBAC glob 路径匹配（对齐 Python 版 `fnmatch`）
- 加密配置明文不缓存到 Redis
- 限流：anon(30/min) / user / introspect(300/min) 三级

## 开发命令

```bash
go build ./...              # 编译全部
go vet ./...                # 静态检查
go test ./...               # 测试（待补充）
go fmt ./...                # 格式化
go run ./cmd/server         # 运行
go run ./cmd/init-data      # 初始化数据
```
