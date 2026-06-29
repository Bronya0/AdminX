# AdminX

> 企业级 Admin 框架底座 — RBAC 权限、动态菜单、JWT 认证、配置中心、定时任务、集群管理、系统监控、文件管理、加密工具。

![alt text](cover.jpg)

## 技术栈

| 层 | 技术 |
| --- | --- |
| 后端 | **Go 1.24** + **Gin** + **GORM** |
| 前端 | **Vue 3** + **TypeScript** + **Ant Design Vue 4** |
| 数据库 | PostgreSQL 16 / SQLite (dev) |
| 缓存 | Redis 7 |
| 认证 | SimpleJWT（HS256，Access 30min，Refresh 7d） |
| 加密 | AES-256-GCM（配置加密） |
| 调度 | gocron v2 |

## 快速开始

```bash
# 克隆
git clone https://github.com/Bronya0/AdminX.git
cd AdminX

# ── 后端 ─────────────────────────────────────
cd backend-go
cp configs/config.example.yaml configs/config.yaml
# 编辑 configs/config.yaml 填入数据库配置

# 初始化数据
go run ./cmd/init-data --username admin --password admin123

# 启动服务
go run ./cmd/server

# ── 前端 ─────────────────────────────────────
cd ../adminx-ui
npm install
npm run dev   # http://localhost:5173
```

## 项目结构

```
AdminX/
├── adminx-ui/                     # 前端（Vue3 + Ant Design Vue）
├── backend-go/                    # Go 后端（Gin + GORM）
│   ├── cmd/server/                #   服务入口
│   ├── cmd/init-data/             #   种子数据初始化
│   ├── internal/
│   │   ├── model/                 #   GORM 模型
│   │   ├── repository/            #   数据访问层
│   │   ├── service/               #   业务逻辑层
│   │   ├── handler/               #   HTTP 处理器
│   │   ├── middleware/            #   JWT/RBAC/限流/日志中间件
│   │   ├── router/                #   路由注册
│   │   ├── jwt/                   #   JWT 签发和验证
│   │   ├── scheduler/             #   调度器（gocron v2）
│   │   ├── websocket/             #   WebSocket Hub
│   │   ├── config/                #   Viper 配置管理
│   │   ├── captcha/               #   验证码
│   │   ├── database/              #   GORM 数据库连接
│   │   └── redis/                 #   Redis 客户端
│   ├── pkg/                       #   可复用包
│   └── configs/                   #   配置示例
├── business-service/              # 示例业务服务（FastAPI，独立部署）
├── README.md
└── AGENTS.md
```

## API 概览

| 模块 | 端点 | 说明 |
|------|------|------|
| **Auth** | `POST /api/v1/accounts/login/` | JWT 登录（含登录锁定） |
| | `POST /api/v1/accounts/logout/` | 登出（黑名单 refresh token） |
| | `POST /api/v1/accounts/refresh/` | 刷新 token |
| | `POST /api/v1/accounts/introspect/` | Token 校验（业务容器接入） |
| **用户** | `GET/POST/PUT/DELETE /api/v1/accounts/users/` | 用户 CRUD |
| | `GET /api/v1/accounts/users/me/` | 当前用户信息 + 菜单 |
| **角色** | `GET/POST/PUT/DELETE /api/v1/accounts/roles/` | 角色 CRUD（绑定权限+菜单） |
| **权限** | `GET /api/v1/accounts/permissions/` | 权限列表 |
| **菜单** | `GET/POST/PUT/DELETE /api/v1/menu/` | 菜单 CRUD |
| | `GET /api/v1/menu/tree/` | 当前用户菜单树 |
| **配置中心** | `GET/POST/PUT/DELETE /api/v1/config/` | 动态配置 CRUD |
| | `GET /api/v1/config/by_group/` | 按分组批量查询 |
| **监控** | `GET /api/v1/monitor/resources/` | 系统资源实时数据 |
| **集群** | `GET/POST/PUT/DELETE /api/v1/cluster/nodes/` | 集群节点管理 |
| | `GET /api/v1/cluster/nodes/overview/` | 集群概览 |
| **定时任务** | `GET/POST/PUT/DELETE /api/v1/jobs/` | 任务 CRUD |
| | `POST /api/v1/jobs/{id}/run_once/` | 立即执行 |
| | `GET /api/v1/jobs/status/` | 调度器状态 |
| **任务日志** | `GET /api/v1/job-logs/` | 任务执行历史 |
| **文件** | `POST /api/v1/files/upload/` | 文件上传 |
| | `GET /api/v1/files/records/` | 文件记录 |
| **验证码** | `GET /api/v1/captcha/` | 图形验证码 |
| **审计日志** | `GET /api/v1/audit/logs/` | 操作审计 |
| **通知** | `GET/POST /api/v1/notification/` | 通知中心 + Webhook |
| **健康检查** | `GET /api/v1/common/health/` | 健康检查 |
| **站点信息** | `GET /api/v1/common/site-info/` | 站点信息 |

## 业务容器开发

业务功能运行在独立容器中，通过 JWT Token Introspection 接入：

1. 使用任意语言/框架编写业务服务
2. 将 token 发至 `POST /api/v1/accounts/introspect/`
3. 平台返回 `user_id`, `username`, `roles`, `permissions`
4. 业务服务据此执行本地鉴权

示例见 `business-service/`（FastAPI 实现）。

## License

MIT
