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
- 菜单层级：`parent_id` + `depth`（根=1，子=父+1），`depth` 由 `MenuService` 从父级派生并拒绝成环/自引用；前端 `flatMenusToTreeByDepth` **先按 `parent_id` 建树**，取不到父节点才退回 `depth` + path 前缀（前缀比较前先剥掉 `scheme://host`，否则外链/iframe 菜单的绝对 URL 会被挂到树根）；**depth 写错就等于多级菜单变平铺**
- 骨架菜单种子（`defaultMenuSeeds`）对骨架内的 code 是**定义**的权威来源（对齐 Django 版 `update_or_create`），重跑 init-data 会覆盖其名称/路径/权限码/RBAC 规则；但**启停（`is_active`）与可见性（`is_visible`）不覆盖**：那是运维状态，种子未显式声明（`menuSeed.IsActive` / `IsVisible`，nil=不表态）就保留库里现值，否则每次 init-data 都会把应急停用/隐藏的菜单静默刷回启用可见。改种子 code 必须在 `legacyMenuCodeRenames` 登记，否则历史库会多出一份重复菜单。`permission_code` 必须等于前端 `router/index.ts` 的 `meta.permission`
- 新增模型/字段后必须同步 `internal/database/sqlite.go` 的 SQLite DDL（`Models()` 与守卫测试会校验覆盖完整性）；**已存在的本地库不会自动加列**（`CREATE TABLE IF NOT EXISTS` 不给旧表加列），新增列要同时在 `migrateSQLite` 的补列清单里登记（`ensureSQLiteColumn`）
- **角色名分两套**：`name` 是英文标识（权限判定、`super_admin` 常量、表单提交都用它），`display_name` 只给界面展示；`UserDTO.role_labels` 与 `roles` 一一对应，展示用 labels、提交用 roles，两者不能混用

### 序列化

- **不要直接返回 model**，用 `model.ToUserDTO()` / `model.ToServiceComponentDTO()`
- 前端依赖 `role_names: string[]`、`role_labels: string[]`、`is_online`、`status` 等计算字段
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
- **规则的解析与校验只有 `internal/rbacrule` 一处**：middleware 用它匹配、`MenuService` 用它校验写入（非法规则直接 400）。各写一份的下场是「接口校验通过、运行时永不匹配」——配了权限仍然 403 且无任何报错
- **菜单 `path` 为空时其 `allowed_paths` 不参与授权**（查权限前先按 `path != ''` 收菜单集合）：想用菜单控制接口权限就必须给它一个路径
- 菜单可见性同一条“全放行”约定：`MenuService.MenusByUser(uid, isSuperuser)` 对超管返回全部启用菜单（`/accounts/users/me/`、`/menu/user_tree/` 均传入 `isSuperuser(c)`）

### 调度器

- gocron v2 + `WithDistributedLocker(gocron-redis-lock)`
- **任务级锁**（非 leader 选举）：多实例并行跑不同任务；锁 key = **数据库任务 ID**（`gocron.WithName(job.ID)` —— 用任务名会让同名任务共用一把锁，一次执行被静默吞掉）
- 锁 TTL 取 `scheduler.lock_ttl_sec`（<10s 或未配则 30s）+ `WithAutoExtendDuration` 自动续期；拿不到锁时 gocron 是**静默 Skip**，靠 `AfterLockError` 监听补一条 failed job_logs
- `Reload` **全程持锁**且**先读库再动现有任务**：否则并发重载会留下清不掉的孤儿 job（任务双跑），一次 DB 抖动会清空全部任务
- reload_pending 消费后重载失败必须**放回标记**，否则全量任务停摆且不再重试
- `Stop` 先取消任务 context（`jobCtx`）再 `ShutdownWithContext`：否则 SIGTERM 后 shell 子进程变孤儿继续跑
- handler 注册表：`service.RegisterHandler("jobs.TaskName", func)`（在 init-data 或启动时注册）
- 独立调度器进程 = 同一二进制加 `--scheduler`（需另一份配置换端口，否则与 web 进程抢 :8000）；`scripts/build.sh` 未单独产出 scheduler 二进制
- `m.jobIDs` 有 mutex 保护
- **触发配置（trigger_config）的解析/校验只有 `internal/trigger` 一处**：service 写入前校验（非法直接 400）、scheduler 用它建 gocron 定义。契约：cron `{"expr":"*/5 * * * *"}`（6 段带秒）、interval `{"minutes":5}`、date `{"datetime":"2006-01-02 15:04:05"}`；同时兼容裸 cron 串与 `run_date` 历史数据。任何新增的解析点都必须接它 —— 各写一份的下场是接口 201、任务永不注册

### 文件中心

- 上传**流式**消费 multipart part（`c.Request.MultipartReader()`）：`c.FormFile` 会把整份 body 先落到系统临时目录，100MB 上限挡不住写满 /tmp
- 落盘名 = UUID + 扩展名白名单（不含 svg），原始文件名只入库（已去目录/控制字符）
- **归属隔离**：非超管只能列举/下载/删除 `uploaded_by = 自己` 的记录；取文件统一走鉴权端点 `GET /files/records/:id/download/`，不要挂匿名静态目录
- 权限归 `system:file` 菜单（`/api/v1/files/*`），不要寄生在其它菜单上
- 删除先删记录再删盘（反之会留下谁也清不掉的孤儿文件）；`mime_type` 用内容嗅探而非客户端声明的 Content-Type

### 监控与历史采样

- `monitor_samples` 采样**不依赖调度器**：每次 `/monitor/resources/`（含仪表盘）按 1 分钟最小间隔补一条，历史查询时顺手清理 7 天前的数据；采样失败只记 warn，不影响实时接口
- `/monitor/resources/history/` 在 Go 侧分桶聚合（不写方言时间函数），range=1h/6h/24h/7d、interval=1m/5m/15m/1h（缺省按窗口推导）；**空桶必须补齐并返回 null**（前端按索引等距画图，缺桶会让时间轴压缩失真），前端在 null 处断开折线
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
- 全局主题（不分用户）整份 JSON 存配置中心 `group=theme`、key=`theme.settings`，由 `ThemeService` 读写：`GET /common/theme/` 为公共端点（登录页也要应用主题），`POST` 与导入导出走 RBAC（同挂在 `system-config` 菜单，新增路径需重跑 init-data 才会刷新种子）。字段覆盖式合并 + 整体校验，未知字段直接 400
- 字典导入导出：`ConfigService.ImportFile/ExportFile`。导入按 key 整条覆盖、整批校验后单事务写入；加密配置不导出、导入时拒绝覆盖。`ConfigRepo.Create` 必须经过回写，别删：`is_active` 带 `default:true`，GORM 会把停用项写成启用
- 由专用服务维护的配置分组（主题 `group=theme`、站点信息 `group=site`）必须登记进 `serviceOwnedGroups`：字典导出跳过它们、导入拒绝写入。否则通用导入只校验「值是合法 JSON」，能把 `theme.settings` 写成非法布局 → 前端三个布局分支全不成立，整站白屏
- 密码策略单例 `password_policies` ID=1：默认值只在 `model.DefaultPasswordPolicy()` 定义一处，init-data 用同一份播种（否则读取时会打 record not found 日志）

## 测试

`go test ./...` 覆盖：jwt、middleware（RBAC/限流）、service（auth/user/role/menu/config/job/notification/rbac 集成）、model、captcha、crypto、errors、database（SQLite DDL 覆盖守卫）、repository（调度器心跳/重载标记）。

## 安全基线（改动时必须维持）

- **Refresh 必须校验 last_logout**（与 JWTAuth 同标准），否则登出二次失效被绕过；WS 日志流也走同一判定（`jwt.RevokedByLogout`）
- **jobs 写操作 / run_once / set_upgrade / menu 写（create+update+delete+register） / policy 写** 挂在 router 的 admin 组（JWT+RBAC+RequireSuperuser），新增高危端点一律进该组
- **组件端点**（/cluster/components/*）依赖 `security.component_secret`（X-Component-Token，恒时比较）；生产必须配置
- **登录**：验证码开关存配置中心 `site.login_captcha_required`（默认开启；站点设置页可改，登录时实时读，SetValues 清缓存所以即时生效）；验证码一次性，答案优先存 Redis、未配置 Redis 时降级到 `captcha_codes` 表（本地调试无需 Redis；两者都没有才 fail-closed）；登录锁定常开（不随 mode）
- **service 层禁止 map 直通 GORM Updates**（mass assignment），一律白名单 struct
- **改密成功会更新 last_logout 吊销全部旧 token**；创建/重置密码走 `validatePasswordPolicy`；**管理员重置用户密码同样吊销该用户旧 token**（`UserRepo.SaveAndMaybeRevoke`，与密码写入同事务）
- **通知已读按用户隔离**：真值在 `notification_reads`，`notifications.is_read` 只是按当前用户计算出的响应字段，不要当列读写
- **超管账号不可被普通管理员改动/删除**（改密、禁用、删除一律 403）
- **最后一个启用中的超管不能被降级/停用/删除**：`UserRepo.SaveAndMaybeRevoke` / `SoftDelete` 传 `guardSuperuser`，校验写在写之后（`ensureSuperuserRemains`），并发下两个超管互相降级也不会双双放行；一旦清零，后台就没人能管，只能改库
- **超管账号的角色只读**：权限来自 `is_superuser`，角色仅用于展示，`UserService.Update` 静默忽略超管的 `roles` 变更（报错会让前端改个描述也保存不了）；`init-data` 给内置超管补绑 `super_admin` 只为列表展示一致
- **LIKE 搜索词必须过 `repository.EscapeLike`，并在 SQL 里附 `ESCAPE '\\'`**：PostgreSQL 的 LIKE 默认把反斜杠当转义符，SQLite 没有，漏了 ESCAPE 会在本地把 `super_admin` 这类带下划线的搜索词变成 0 条（线上 PG 正常，很难复现）
- **写端点必须显式审计**：各 handler 在业务成功后调用 `auditRecordDiff` / `auditOperation`，新增记录新快照、修改记录前后快照、删除保留旧快照；执行类操作写明具体动作和提交/执行结果。使用安全 DTO，敏感数据须脱敏，禁止用请求/响应拦截器推测审计内容。动作 create/update/delete/execute 与前端下拉一致。

## 已知技术债

- shell 任务无沙箱（服务进程权限执行），生产需加固或限超管使用
- 调度器独立进程模式下依赖 scheduler_heartbeats.reload_pending 轮询（10s）感知任务变更
