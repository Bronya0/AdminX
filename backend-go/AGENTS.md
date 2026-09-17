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
- 菜单层级：`parent_id` + `depth`（根=1，子=父+1），`depth` 由 `MenuService` 从父级派生并拒绝成环/自引用；前端 `flatMenusToTreeByDepth` 按 `depth` + path 前缀建树，**depth 写错就等于多级菜单变平铺**
- 骨架菜单种子（`defaultMenuSeeds`）对骨架内的 code 是**权威来源**（对齐 Django 版 `update_or_create`），重跑 init-data 会覆盖其名称/路径/权限码/RBAC 规则；改种子 code 必须在 `legacyMenuCodeRenames` 登记，否则历史库会多出一份重复菜单。`permission_code` 必须等于前端 `router/index.ts` 的 `meta.permission`
- 新增模型/字段后必须同步 `internal/database/sqlite.go` 的 SQLite DDL（`Models()` 与守卫测试会校验覆盖完整性）

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
- 菜单可见性同一条“全放行”约定：`MenuService.MenusByUser(uid, isSuperuser)` 对超管返回全部启用菜单（`/accounts/users/me/`、`/menu/user_tree/` 均传入 `isSuperuser(c)`）

### 调度器

- gocron v2 + `WithDistributedLocker(gocron-redis-lock)`
- **任务级锁**（非 leader 选举）：多实例并行跑不同任务
- handler 注册表：`service.RegisterHandler("jobs.TaskName", func)`（在 init-data 或启动时注册）
- `m.jobIDs` 有 mutex 保护

### 监控与历史采样

- `monitor_samples` 采样**不依赖调度器**：每次 `/monitor/resources/`（含仪表盘）按 1 分钟最小间隔补一条，历史查询时顺手清理 7 天前的数据；采样失败只记 warn，不影响实时接口
- `/monitor/resources/history/` 在 Go 侧分桶聚合（不写方言时间函数），range=1h/6h/24h/7d、interval=1m/5m/15m/1h（缺省按窗口推导）
- `SetReloadPending` **不得写 `last_heartbeat`**：心跳只能由调度器进程自己写，否则 web 进程一调 reload，组件状态页会把未启动的调度器误报为在线

### WebSocket

- Hub 模式，broadcast 用 **写锁**（Lock，非 RLock）
- `closeSend()` 幂等关闭 channel（防 double close panic）
- Redis pub/sub channel `ws:logs` 跨实例广播

### 配置

- viper：yaml 文件 + `DJA_` 前缀环境变量覆盖
- `database.driver`：`postgres`（默认/生产，AutoMigrate 建表）/ `sqlite`（本地开发，显式 DDL 建表、连接池固定 1）
- **原始 SQL 必须方言无关**：时间由 Go 侧传参，不用 `NOW()` / `INTERVAL`；不用 `WITH ... UPDATE ... RETURNING`（SQLite 的 CTE 不支持数据修改语句）
- AES-GCM 密钥 base64 编码 32 字节，留空则加密功能禁用（不报错）
- 加密配置明文 **不缓存** Redis（安全）
- 站点信息（站点名/描述/Logo/主题色/空闲超时/版本）存配置中心 `group=site`、key 固定 `site.*`：`GET /common/site-info/` 为公共端点（登录页需要），`POST` 走 RBAC（默认绑在 `system-config` 菜单）；写键值走 `ConfigService.SetValue`，别直接建 repo 行
- 密码策略单例 `password_policies` ID=1：默认值只在 `model.DefaultPasswordPolicy()` 定义一处，init-data 用同一份播种（否则读取时会打 record not found 日志）

## 测试

`go test ./...` 覆盖：jwt、middleware（RBAC/限流）、service（auth/user/role/menu/config/job/notification/rbac 集成）、model、captcha、crypto、errors、database（SQLite DDL 覆盖守卫）、repository（调度器心跳/重载标记）。

## 安全基线（改动时必须维持）

- **Refresh 必须校验 last_logout**（与 JWTAuth 同标准），否则登出二次失效被绕过
- **jobs 写操作 / run_once / set_upgrade / menu create+register / policy 写** 挂在 router 的 admin 组（JWT+RBAC+RequireSuperuser），新增高危端点一律进该组
- **组件端点**（/cluster/components/*）依赖 `security.component_secret`（X-Component-Token，恒时比较）；生产必须配置
- **登录**：`security.login_captcha_required` 开启后强制验证码（一次性、Redis 存储）；登录锁定常开（不随 mode）
- **service 层禁止 map 直通 GORM Updates**（mass assignment），一律白名单 struct
- **改密成功会更新 last_logout 吊销全部旧 token**；创建/重置密码走 `validatePasswordPolicy`
- **LIKE 搜索词必须过 `repository.EscapeLike`**

## 已知技术债

- shell 任务无沙箱（服务进程权限执行），生产需加固或限超管使用
- 调度器独立进程模式下依赖 scheduler_heartbeats.reload_pending 轮询（10s）感知任务变更
