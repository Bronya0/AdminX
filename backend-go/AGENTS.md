# AGENTS.md — backend-go

Go 后端项目指南，供 AI 助手和开发者参考。

## 命令

```bash
go build ./...              # 编译检查
go vet ./...                # 静态检查（必须通过）
go test ./...               # 测试
go run ./cmd/server         # 启动 HTTP 服务
go run ./cmd/server --scheduler   # 启动 + 内置调度器
go run ./cmd/init-data      # 初始化种子数据
bash scripts/build.sh       # 构建生产二进制到 bin/
```

## 架构

分层 + 依赖注入，所有依赖通过构造函数传入，无全局变量（handler_registry 例外）：

```
cmd/server/main.go   ← 唯一组装点
  config → database → redis → AES → JWT
  → repository → service → handler → router → gin.Engine
```

### 分层职责

| 层 | 职责 | 禁止 |
|---|---|---|
| `model/` | GORM 模型 + DTO | 业务逻辑、HTTP |
| `repository/` | 数据访问（GORM 封装） | 业务逻辑、返回 HTTP 错误 |
| `service/` | 业务逻辑 + 错误（apperr） | HTTP、gin |
| `handler/` | 绑定参数 → 调 service → 格式化响应 | 业务逻辑 |
| `middleware/` | JWT/RBAC/限流/日志 | 业务逻辑 |
| `pkg/` | 可复用工具（无 internal 依赖） | 依赖 internal |

### 错误处理

- service 层返回 `*apperr.AppError`（携带业务 code）
- handler 层用 `response.Error(c, logger, err)` 统一转换
- **所有响应 HTTP 恒 200**，业务状态在 `code` 字段（对齐 Python 版）

### 统一响应

```go
response.OK(c, data)              // {code:200, msg:"success", data:...}
response.Created(c, data)         // {code:201, ...}
response.Fail(c, 400, "错误信息")  // {code:400, msg:"...", data:null}
response.Paginated(c, count, next, prev, results)  // 分页
```

## 关键约定

### 模型

- UUID 主键：`gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
- 软删除：仅 User 用 `gorm.DeletedAt`（其他模型无软删除）
- jsonb 字段：`datatypes.JSON` + `default:'[]'::jsonb`（必须带 `::jsonb` cast）
- M2M：`gorm:"many2many:table_name;"`，GORM 自动建中间表

### 序列化

- **不要直接返回 model**，用 `model.ToUserDTO()` / `model.ToServiceComponentDTO()`
- 前端依赖 `role_names: string[]`、`is_online`、`status` 等计算字段
- 密码字段 `json:"-"` 永不序列化

### 认证

- JWT HS256，access(30min) + refresh(7d)
- token_type 是独立 claims 字段（不用 Subject）
- logout 校验 refresh 归属 + last_logout 二次失效
- 黑名单存 Redis set（key: `jwt:blacklist`）

### RBAC

- superuser 全放行
- 普通用户：查 user→role→menu→allowed_paths，glob 匹配（`path.Match`）
- 规则格式：`METHOD:/api/path/*` 或 `/api/path`

### 调度器

- gocron v2 + `WithDistributedLocker(gocron-redis-lock)`
- **任务级锁**（非 leader 选举）：多实例并行跑不同任务
- handler 注册表：`service.RegisterHandler("jobs.TaskName", func)`（在 init-data 或启动时注册）
- `m.jobIDs` 有 mutex 保护

### WebSocket

- Hub 模式，broadcast 用 **写锁**（Lock，非 RLock）
- `closeSend()` 幂等关闭 channel（防 double close panic）
- Redis pub/sub channel `ws:logs` 跨实例广播

### 配置

- viper：yaml 文件 + `DJA_` 前缀环境变量覆盖
- AES-GCM 密钥 base64 编码 32 字节，留空则加密功能禁用（不报错）
- 加密配置明文 **不缓存** Redis（安全）

## 测试

当前无测试。建议优先补：
- `internal/jwt/` 生成/解析/黑名单
- `internal/middleware/rbac.go` glob 匹配
- `internal/service/auth_service.go` 登录流程

## 已知技术债

- `executeShell` 用 `strings.Fields` 切割命令，参数含空格/引号会出错（应引入 shlex）
- shell 任务无沙箱（服务进程权限执行），生产需加固
- 缺少结构化测试
- 验证码只返回文本，前端需自行渲染图片
