# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

本仓库为 monorepo：Django 后端在 `backend-django/`，Go 后端骨架在 `backend-go/`，前端在 `adminx-ui/`。

```bash
# Backend (Django) — 先 cd backend-django
cd backend-django
python manage.py migrate                # DB migrations
python manage.py init_data --superuser --username admin --password admin123  # Seed superuser
python manage.py run_scheduler          # Start APScheduler standalone process
python manage.py runserver              # Dev server (settings: config.settings.dev)

# Redis (optional - see README "启用 Redis" section)
# Just set REDIS_URL in .env, no code changes needed. Auto-detected at startup.

# Tests
python manage.py test                   # All tests (97 total)
python manage.py test djangoadminx.accounts    # Single module

# Frontend (adminx-ui/)
cd adminx-ui && npm run dev             # Vite dev server (port 5173, proxies /api to :8000)
cd adminx-ui && npm run build           # Production build
cd adminx-ui && npm run type-check      # Vue/TS type checking (skip build)

# Backend (Go skeleton)
cd backend-go && go build ./cmd/server  # Build skeleton (placeholder only)
```

## Project Structure & File Index

本仓库为 monorepo：Django 后端在 `backend-django/`，Go 后端骨架在 `backend-go/`，前端在 `adminx-ui/`。下方文件索引默认相对于 `backend-django/`（如 `accounts/models.py` 实际位于 `backend-django/djangoadminx/accounts/models.py`）。

### Backend Django (`backend-django/djangoadminx/`)

#### accounts — 用户认证管理
| 文件 | 说明 |
|------|------|
| `accounts/models.py` | User（软删除 UUID PK）、Role（name 唯一, is_system）、LoginLock、UserLoginLog、BusinessPermission、BusinessCommand |
| `accounts/serializers.py` | 用户/角色/权限/登录日志的 DRF 序列化器 |
| `accounts/views.py` | 登录/登出/JWT/用户/角色/权限/业务命令 ViewSet + APIView |
| `accounts/urls.py` | login、logout、introspect、6 个 ViewSet 路由 |
| `accounts/permissions.py` | RBACPermission — 基于菜单 allowed_paths 的路径白名单校验 |
| `accounts/sso.py` | CAS 单点登录 (django-cas-ng) |

#### audit — 审计日志
| 文件 | 说明 |
|------|------|
| `audit/models.py` | AuditLog 模型（记录核心模型的增删改） |
| `audit/views.py` | AuditLogViewSet 只读，支持过滤/日期范围搜索 |
| `audit/urls.py` | 路由注册 |
| `audit/mixins.py` | AuditLogMixin — ViewSet 中显式记录审计日志 |
| `audit/signals.py` | signal 自动拦截模型增删改写入 AuditLog |
| `audit/utils.py` | 审计工具（序列化、获取操作人/IP） |

#### captcha — 验证码
| 文件 | 说明 |
|------|------|
| `captcha/views.py` | 生成和校验图片验证码 API |
| `captcha/urls.py` | 路由注册 |

#### cluster — 集群管理
| 文件 | 说明 |
|------|------|
| `cluster/models.py` | ClusterNode（主/从、在线/离线） |
| `cluster/views.py` | ClusterNodeViewSet CRUD + 心跳检测 |
| `cluster/urls.py` | 路由注册 |

#### common — 公共模块
| 文件 | 说明 |
|------|------|
| `common/views.py` | 仪表盘统计、健康检查、站点信息、日志 tail、NTP 同步等通用 API |
| `common/urls.py` | dashboard、health、site-info、log、cache、ntp 路由 |
| `common/middleware.py` | RequestContextMiddleware（thread-local）+ 请求耗时日志 |
| `common/exceptions.py` | 全局异常处理器，统一 {code, msg, data} |
| `common/scheduler.py` | APScheduler 管理器（cron/interval/date + 跨进程心跳协调） |
| `common/crypto_utils.py` | 加解密工具（SM4、AES、MD5、SHA256、HMAC） |
| `common/renderers.py` | StandardJsonRenderer — 所有响应统一 {code, msg, data} |
| `common/pagination.py` | StandardPagination — 统一分页 |
| `common/consumers.py` | WebSocket LogConsumer — 实时推送日志 |
| `common/cache_views.py` | 缓存统计和清空 API |
| `common/ntp.py` | NTP 时间同步客户端 |
| `common/management/commands/init_data.py` | 初始化角色/菜单/配置/超级用户 |
| `common/management/commands/run_scheduler.py` | 独立进程运行 APScheduler |

#### config_center — 配置中心
| 文件 | 说明 |
|------|------|
| `config_center/models.py` | Config（5 种值类型 + 加密开关 + Fernet 缓存） |
| `config_center/views.py` | ConfigViewSet CRUD + 分组查询 + 按 key 读取 + NTP 同步 |
| `config_center/serializers.py` | 显示值/加密值处理 |
| `config_center/urls.py` | 路由注册 |

#### data_center — 数据中心
| 文件 | 说明 |
|------|------|
| `data_center/views.py` | 通用 Excel 导入导出（openpyxl，模型白名单） |
| `data_center/urls.py` | 路由注册 |

#### file_center — 文件中心
| 文件 | 说明 |
|------|------|
| `file_center/models.py` | FileRecord（本地存储统一管理） |
| `file_center/views.py` | 文件上传 API + FileRecord CRUD |
| `file_center/urls.py` | 路由注册 |

#### menu — 动态菜单
| 文件 | 说明 |
|------|------|
| `menu/models.py` | Menu（treebeard + path 白名单 + permission_code） |
| `menu/views.py` | MenuViewSet CRUD + 当前用户菜单树 + 移动（受限） |
| `menu/serializers.py` | Menu 序列化器 + 树形序列化器（code 前缀匹配） |
| `menu/urls.py` | 路由注册 |

#### monitor — 系统监控
| 文件 | 说明 |
|------|------|
| `monitor/views.py` | 系统资源实时信息 + 网络连接统计 API |
| `monitor/urls.py` | 路由注册 |
| `monitor/utils.py` | SystemMonitor（CPU、内存、磁盘、网络、进程 IO） |

#### notification — 通知中心
| 文件 | 说明 |
|------|------|
| `notification/models.py` | Notification、WebhookConfig、WebhookLog |
| `notification/views.py` | 通知只读+标记已读、Webhook 配置 CRUD |
| `notification/urls.py` | 路由注册 |
| `notification/webhook.py` | 信号驱动的 Webhook 外发（HMAC 签名） |

#### policy — 密码策略
| 文件 | 说明 |
|------|------|
| `policy/models.py` | PasswordPolicy（单例）、PasswordHistory |
| `policy/views.py` | 获取密码策略 + 修改密码 API |
| `policy/urls.py` | 路由注册 |

#### webservice — 定时任务 (API prefix: `/api/v1/jobs/`)
| 文件 | 说明 |
|------|------|
| `jobs/models.py` | ScheduleJob、JobLog、SchedulerHeartbeat |
| `jobs/views.py` | ScheduleJobViewSet CRUD + 执行/重载、JobLogViewSet 只读 |
| `jobs/urls.py` | 路由注册 |
| `jobs/tasks.py` | 任务处理函数（NTP 同步、清理日志等） |

#### tests — 测试（97 total）
| 文件 | 说明 |
|------|------|
| `tests/base.py` | AdminXTestCase 基类（JWT 认证辅助） |
| `tests/factories.py` | UserFactory、RoleFactory、ConfigFactory |
| `tests/accounts/test_auth.py` | 登录/登出/刷新/用户 CRUD |
| `tests/accounts/test_login_lock.py` | 登录锁定 |
| `tests/accounts/test_permission.py` | RBACPermission 路径白名单 |
| `tests/audit/test_audit.py` | 审计 signals + mixins |
| `tests/captcha/test_captcha.py` | 验证码 API |
| `tests/cluster/test_cluster.py` | 集群节点 CRUD |
| `tests/common/test_crypto.py` | 加解密工具 |
| `tests/common/test_renderer.py` | Renderer + 异常处理器 |
| `tests/common/test_scheduler.py` | 调度心跳 + 重载 |
| `tests/config_center/test_config.py` | 配置中心 CRUD + 加密 |
| `tests/monitor/test_monitor.py` | 系统监控 API |
| `tests/notification/test_notification.py` | 通知 API |
| `tests/policy/test_policy.py` | 密码策略 + 修改密码 |
| `tests/webservice/test_job.py` | 任务模型 + CRUD |
| `tests/webservice/test_job_views.py` | 任务执行/状态/重载 |

#### config/ — 项目配置（`backend-django/config/`）
| 文件 | 说明 |
|------|------|
| `config/settings/base.py` | 基础配置（应用、数据库、缓存、JWT、CORS、安全等） |
| `config/settings/dev.py` | 开发环境（SQLite、DEBUG=True、CORS 全开放） |
| `config/settings/prod.py` | 生产环境（PostgreSQL、强制校验、HSTS、强制 Redis） |

#### 根目录（`backend-django/`）
- `manage.py` — 入口（默认 config.settings.dev）
- `requirements.txt` — 依赖清单
- `start-linux.sh` / `stop-linux.sh` / `start-win.bat` — 启停脚本
- `.env.example` — 环境变量示例
- `deploy/` — Docker / nginx / systemd / supervisor 配置
- `scripts/deploy.sh` — 一键部署脚本

---

### Frontend (`adminx-ui/`)

#### 入口 & 配置
| 文件 | 说明 |
|------|------|
| `main.ts` | Vue 应用入口（Pinia、Router、Ant Design Vue） |
| `App.vue` | 根组件（a-config-provider + router-view + 主题/中文） |
| `types/index.ts` | 所有 TS 接口（ApiResponse、User、Role、Menu、Config 等） |
| `router/index.ts` | 路由定义 + 导航守卫（权限/redirect/home_page） |

#### 状态管理（Pinia）
| 文件 | 说明 |
|------|------|
| `stores/index.ts` | Pinia 实例 + pinia-plugin-persistedstate |
| `stores/user.ts` | 用户状态（token、信息、权限、菜单、主题、登录/登出） |

#### API 层
| 文件 | 说明 |
|------|------|
| `api/auth.ts` | 登录/登出/刷新 + 用户/角色/权限/登录日志 CRUD |
| `api/common.ts` | 站点信息、组件健康检查 |
| `api/menu.ts` | 菜单管理 CRUD |
| `api/config.ts` | 配置中心 CRUD |
| `api/audit.ts` | 审计日志查询 |
| `api/cluster.ts` | 集群节点 CRUD |
| `api/dashboard.ts` | 仪表盘统计 |
| `api/monitor.ts` | 系统资源/网络监控 |
| `api/notification.ts` | 通知列表/已读 |
| `api/webhook.ts` | Webhook 配置 CRUD + 日志 |
| `api/webservice.ts` | 定时任务 CRUD + 执行日志 |

#### 工具函数
| 文件 | 说明 |
|------|------|
| `utils/request.ts` | Axios 封装（JWT + Token 刷新 + 全局错误） |
| `utils/format.ts` | 日期格式化 ISO → "YYYY-MM-DD HH:mm:ss" |
| `utils/menuTree.ts` | 扁平菜单转树形 + 路由侧边栏生成 |
| `utils/iconResolver.ts` | 字符串 → @ant-design/icons-vue 组件 |

#### 布局 & 通用组件
| 文件 | 说明 |
|------|------|
| `layouts/AdminLayout.vue` | 主布局（左侧/顶部/混合 + 标签页 + 页头） |
| `layouts/SideBar.vue` | 侧边栏（Logo、菜单树、折叠） |
| `layouts/TopNav.vue` | 顶部导航栏 |
| `components/BreadcrumbNav.vue` | 面包屑导航 |
| `components/IdleWatcher.vue` | 空闲超时监测（弹窗 + 自动登出） |
| `components/SidebarMenuItem.vue` | 菜单递归渲染 |

#### 页面组件
| 文件 | 说明 |
|------|------|
| `views/login/LoginView.vue` | 登录页（验证码、记住密码） |
| `views/dashboard/DashboardView.vue` | 仪表盘（统计卡片、最近日志） |
| `views/system/UserList.vue` | 用户管理（搜索 CRUD + 角色分配 + home_page） |
| `views/system/RoleList.vue` | 角色管理（搜索 CRUD + 菜单权限树 + is_system 只读） |
| `views/system/PermissionList.vue` | 权限管理（菜单树 + 接口权限双标签页） |
| `views/system/ThemeSettings.vue` | 主题设置（布局/颜色/暗黑模式） |
| `views/config/ConfigCenter.vue` | 配置中心容器（字典 + 主题双标签页） |
| `views/config/ConfigList.vue` | 字典配置列表（搜索 CRUD + 加密值） |
| `views/audit/AuditLogList.vue` | 审计日志列表（搜索/筛选） |
| `views/audit/LoginLogList.vue` | 登录日志列表（按用户/IP/状态筛选） |
| `views/cluster/ClusterNodes.vue` | 节点管理（概览 + CRUD + 状态） |
| `views/monitor/SystemMonitor.vue` | 系统资源监控（CPU/内存/磁盘图表） |
| `views/monitor/ComponentStatus.vue` | 组件健康检查（DB/Redis/调度器） |
| `views/notification/NotificationCenter.vue` | 通知中心（列表 + Webhook 配置） |
| `views/scheduler/ScheduleJobList.vue` | 定时任务（CRUD + 日志 + 暂停/恢复） |
| `views/NotFound.vue` | 404 页面 |

#### 根目录配置文件
- `package.json` — npm 依赖和脚本
- `vite.config.ts` — Vite（端口 5173、/api 代理、@ 别名）
- `tsconfig.json` / `tsconfig.app.json` / `tsconfig.node.json` — TypeScript 配置

---

## Architecture & Key Patterns

### View Pattern
- Prefer `viewsets.ModelViewSet` for CRUD.
- Custom endpoints via `@action(detail=True/false)` decorator.
- Permissions: `[IsAuthenticated, RBACPermission]` for business views, `[IsAdminUser]` for admin-only.
- `search_fields`, `ordering_fields`, `filterset_class` for query capability declaration.
- Exceptions handled globally — views only contain business logic.

### Unified Response Format
- All responses: `{code: int, msg: string, data: ...}` — handled by `StandardJsonRenderer`.
- All exceptions wrapped by global handler in `common/exceptions.py`.
- Custom `StandardPagination` returns `{count, next, previous, results}`.
- **Testing note**: `StandardJsonRenderer` forces all status_code to 200. Tests must call `response.render()` then parse JSON to check `data["code"]` instead of `response.status_code`.

### Config Center
- KV config with 5 types: `string`, `int`, `bool`, `json`, `options` (option lists).
  - `encrypted` type removed — encryption is now an independent toggle via `is_encrypted` boolean.
- **Encryption**: Each config has an `is_encrypted` switch. When enabled, the value is Fernet-encrypted in the database.
  - API responses return `value: ""`, `display_value: "********"`, `is_encrypted: true`
  - Internal reads (`parse_value()`, `get_value()`, `get_by_group()`) return the decrypted plaintext automatically.
  - **FERNET_KEY must be configured** in `.env` for encryption to work:
    ```
    FERNET_KEY=$(python -c "from cryptography.fernet import Fernet; print(Fernet.generate_key().decode())")
    ```
  - Once enabled, encryption **cannot be disabled** on an existing config.
  - If FERNET_KEY is missing, a warning is logged and values are stored as plaintext (not silently dropped).
- Read via `Config.get_value("KEY", default=...)` or `Config.get_by_group("group_name")`.
- Cached 1h in Redis; auto-cleared on save/delete. Falls back to LocMemCache if Redis unavailable.

### Config Center Integration in Apps
- Business option lists stored in config center (`group` + `options` type).
- Example: post status options stored as `group="post_status"`, read via `Config.get_by_group("post_status")`.

### Scheduler
- APScheduler runs as standalone process (`python manage.py run_scheduler`).
- Job definitions stored in DB (`ScheduleJob` model), supports `cron` / `interval` / `date` triggers.
- **Cross-process communication via database** (no Redis dependency):
  - **Heartbeat**: scheduler process updates `SchedulerHeartbeat.last_heartbeat` every 10s via `update_or_create`; web workers check via `is_alive()` (30s timeout threshold).
  - **Reload notifications**: CRUD signals + reload button set `SchedulerHeartbeat.reload_pending=True`; scheduler process polls every 10s and calls `reload_all()`.
  - See [`SchedulerManager`](backend-django/djangoadminx/common/scheduler.py) and [`SchedulerHeartbeat`](backend-django/djangoadminx/jobs/models.py) for implementation.
- **Anti-pattern (fixed)**: Before this approach, signals and reload called `reload_job()`/`reload_all()` directly on `SchedulerManager` singleton — this only affected the calling process (one Gunicorn worker) and was silently ignored by the actual scheduler process. **Never rely on in-process state for cross-process coordination.**
- **Shell commands**: Use `shlex.split()` + `subprocess.run(shell=False)` to prevent command injection.
- **Security**: `ScheduleJobViewSet` requires `IsAdminUser` permission.

### Encryption (backend-django/djangoadminx/common/crypto_utils.py)
- SM4 (ECB/CBC), AES (CBC/GCM), MD5, SHA256, HMAC-SHA256.
- Each function has `_b64` variant for base64 I/O.

### Frontend Patterns
- API layer: `adminx-ui/src/api/*.ts` — typed functions calling `request` instance.
- State: Pinia stores (user store persisted via `pinia-plugin-persistedstate`).
- Router guards: `router/index.ts` checks `userStore.isLoggedIn` and redirects to `/login`.
- TypeScript: one `types/index.ts` with all API interfaces.
- All API calls use `/api/v1/` prefix (proxied by Vite in dev).
- **Double error prevention**: Avoid `message.error()` in catch blocks — the global axios interceptor already shows errors. Use meaningful generic messages only for client-side validation.
- **PATCH for partial updates**: Use `patchRole` (not `updateRole`) when only changing `menus` to avoid required field validation errors.
- **Type conventions**:
  - `id` is `number` for models with auto-increment PK (Config, Menu)
  - `id` is `string` for models with UUID PK (User, Role, etc.)
  - `Config.display_value` can be `string | any[]` (options type returns array)
  - `Config.encrypted_value` field removed from frontend type

### Testing Patterns
- Use `APITestCase` from `rest_framework.test`.
- Set up shared data in `setUpTestData` classmethods, auth in `setUp`.
- Always call `response.render()` and parse JSON: `json.loads(response.content)`.
- Assert on `data["code"]` (not response.status_code) due to `StandardJsonRenderer`.
- **80 total tests**: accounts (auth, login_lock, permission), audit, captcha, cluster, config_center, notification, policy, scheduler, common (crypto, renderer, scheduler).

### Business Container Development
Business services run in separate containers and authenticate via JWT introspection:
1. Write business service in any language/framework
2. Use `POST /api/v1/accounts/introspect/` to validate tokens
3. Platform returns `user_id`, `roles`, `permissions` for local authorization
4. Register menus/routes via Config Center or platform registration API

### Built-in Roles (等保2.0 三权分立)
- **超级管理员** (`super_admin`) — all permissions, all menus
- **安全管理员** (`security_admin`) — config center (security-related), monitor, cluster, audit logs (read-only)
- **审计管理员** (`audit_admin`) — audit logs, login logs (read-only)
- **普通用户** (`user`) — basic access only

Roles and their permission/menu bindings are initialized in `init_data.py`. Run `python manage.py init_data` to apply.

## Known Bugs & Limitations

### Menu treebeard `path` field conflict
`Menu.path` field (route path) shadows treebeard MP_Node's internal `path` column. Tree structure is rebuilt on the frontend from flat menu data (by path prefix matching in `menuTree.ts`). Treebeard's `move()` / `add_child()` operations are unsafe — they will corrupt the route path. The `move` endpoint returns an error explaining this limitation. Menu creation in `init_data.py` uses direct ORM with manually set `depth`/`numchild` values.

### Audit Signals
- **UPDATE auditing via signals**: Uses `pre_save` + `post_save` pair to capture old/new values correctly. The `pre_save` signal snapshots the pre-save state; `post_save` computes the diff. `AuditLogMixin` (used directly by ViewSets) captures values before save via `perform_update`, so models using the mixin are unaffected.
- **SELECT/query auditing**: Not implemented (only DML: create/update/delete).
- **Audit log retention**: No automatic cleanup of old records.

### Data Import
- Data import uses `update_or_create` with per-row savepoints, so failures in one row don't roll back successful rows.
- Previously used `transaction.atomic()` wrapping all rows, making error reporting misleading (all rolled back but reported "success: N").

### Password Policy
- Password expiry reference date uses `date_joined` as fallback (not `last_login`) to avoid resetting the expiry clock on login.
- `PasswordPolicy` check is enforced on user creation via `UserCreateSerializer.create()`.
- `create_superuser()` uses `BaseUserManager.create_user()` (not the non-existent `all_objects.create_superuser`).

### JWT & Refresh Tokens
- `BLACKLIST_AFTER_ROTATION = True` requires `rest_framework_simplejwt.token_blacklist` in INSTALLED_APPS (added).
- Token blacklist tables must be migrated: `python manage.py migrate token_blacklist`.
- Access token: 30 min (configurable via `JWT_ACCESS_EXPIRE`).
- Refresh token: 7 days (configurable via `JWT_REFRESH_EXPIRE`).

### Production Security
- Missing settings validated at startup: `SECRET_KEY`, `ALLOWED_HOSTS` (must be set, not default).
- Production security headers: `SECURE_SSL_REDIRECT`, HSTS (31536000s), `SECURE_PROXY_SSL_HEADER`, `SECURE_REFERRER_POLICY`.
- `SECURE_BROWSER_XSS_FILTER` removed (deprecated in Django 4.0+).
- Console logger defaults to WARNING in production.
- `rest_framework_simplejwt.token_blacklist` in INSTALLED_APPS (requires migration).

## Common Issues

### Login fails with "no such table: token_blacklist_..."
Run `python manage.py migrate token_blacklist` after adding `token_blacklist` to INSTALLED_APPS.

### Config encryption returns "<<解密失败>>"
FERNET_KEY not configured. Set in `.env`:
```
FERNET_KEY=$(python -c "from cryptography.fernet import Fernet; print(Fernet.generate_key().decode())")
```

## note
- 禁止修改虚拟环境源码
