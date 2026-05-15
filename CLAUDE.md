# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Backend
python manage.py runserver              # Dev server (settings: config.settings.dev)
python manage.py migrate                # DB migrations
python manage.py init_data              # Init roles, menus, configs
python manage.py init_data --superuser --username admin --password admin123  # Seed superuser
python manage.py run_scheduler          # Start APScheduler standalone process

# Tests
python manage.py test                   # All tests
python manage.py test apps.demo_blog    # Single module
python manage.py test djangoadminx.accounts    # Framework module

# Frontend (adminx-ui/)
cd adminx-ui && npm run dev             # Vite dev server (port 5173, proxies /api to :8000)
cd adminx-ui && npm run build           # Production build
cd adminx-ui && npm run type-check      # Vue/TS type checking

# Production
bash start-linux.sh                     # Gunicorn deployment (Linux)
start-win.bat                           # Dev startup (Windows)
```

## Project Structure

```
DjangoAdminX/
├── djangoadminx/          # Framework code (reusable admin framework)
│   ├── accounts/          #   User/Role/Permission/JWT auth/login lock
│   ├── menu/              #   Dynamic menu (django-treebeard)
│   ├── config_center/     #   Config center (KV + options + Fernet encryption)
│   ├── monitor/           #   System resource monitor (psutil)
│   ├── cluster/           #   Cluster node management
│   ├── webservice/        #   SOAP/spyne + APScheduler jobs + job logs
│   ├── file_center/       #   File upload (Local/MinIO)
│   ├── data_center/       #   Excel import/export
│   ├── captcha/           #   Captcha
│   ├── audit/             #   Audit log (signals-based)
│   ├── policy/            #   Password policy
│   └── common/            #   Middleware, unified response/exception/pagination, scheduler, SSE logs
├── apps/                  # Business applications (decoupled from framework)
│   ├── demo_blog/         #   Example: Post/Category CRUD + SOAP + tests
│   └── common/            #   crypto_utils (SM4/AES/MD5), redis_utils (cache/lock/ratelimit)
├── adminx-ui/             # Frontend (Vue 3 + Ant Design Vue 4 + Pinia + Vite)
│   └── src/
│       ├── api/           #   API service layer (axios)
│       ├── stores/        #   Pinia stores (persisted to localStorage)
│       ├── router/        #   Vue Router (permission guards)
│       ├── views/         #   Page components (system/, config/, monitor/, cluster/, webservice/)
│       ├── types/         #   TypeScript interfaces
│       └── layouts/       #   AdminLayout (sidebar, header, tabs)
├── config/
│   └── settings/          # base.py / dev.py / prod.py
└── requirements.txt
```

## Architecture & Key Patterns

### Framework/Business Decoupling
- `djangoadminx/` is reusable admin framework — never import from `apps/` into framework code.
- `apps/` is business code — imports framework via `djangoadminx.*`, but not internal details.

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
- KV config with 6 types: `string`, `int`, `bool`, `json`, `encrypted` (Fernet), `options` (option lists).
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
  - See [`SchedulerManager`](djangoadminx/common/scheduler.py) and [`SchedulerHeartbeat`](djangoadminx/webservice/models.py) for implementation.
- **Anti-pattern (fixed)**: Before this approach, signals and reload called `reload_job()`/`reload_all()` directly on `SchedulerManager` singleton — this only affected the calling process (one Gunicorn worker) and was silently ignored by the actual scheduler process. **Never rely on in-process state for cross-process coordination.**

### Redis Tooling (apps/common/redis_utils.py)
- `CacheProxy` — auto-serializing cache.
- `RedisProxy().lock()` — distributed lock.
- `RateLimiter` — sliding window rate limiter.
- `delay_double_delete()` — cache update pattern.
- All components degrade gracefully when Redis is unavailable.

### Encryption (apps/common/crypto_utils.py)
- SM4 (ECB/CBC), AES (CBC/GCM), MD5, SHA256, HMAC-SHA256.
- Each function has `_b64` variant for base64 I/O.

### Frontend Patterns
- API layer: `adminx-ui/src/api/*.ts` — typed functions calling `request` instance.
- State: Pinia stores (user store persisted via `pinia-plugin-persistedstate`).
- Router guards: `router/index.ts` checks `userStore.isLoggedIn` and redirects to `/login`.
- TypeScript: one `types/index.ts` with all API interfaces.
- All API calls use `/api/v1/` prefix (proxied by Vite in dev).

### Testing Patterns
- Use `APITestCase` from `rest_framework.test`.
- Set up shared data in `setUpTestData` classmethods, auth in `setUp`.
- Always call `response.render()` and parse JSON: `json.loads(response.content)`.
- Assert on `data["code"]` (not response.status_code) due to `StandardJsonRenderer`.

### Adding a New Module
1. Create app: `python manage.py startapp myapp apps/myapp`
2. Register in `config/settings/base.py` → `LOCAL_APPS`
3. Register routes in `config/urls.py`
4. Model → Serializer → ViewSet → URL → Test (follow demo_blog patterns)

### Built-in Roles (等保2.0 三权分立)
- **超级管理员** (`super_admin`) — all permissions, all menus
- **安全管理员** (`security_admin`) — config center (security-related), monitor, cluster, audit logs (read-only)
- **审计管理员** (`audit_admin`) — audit logs, login logs (read-only)
- **普通用户** (`user`) — basic access only

Roles and their permission/menu bindings are initialized in `init_data.py`. Run `python manage.py init_data` to apply.

### Multi-Process Architecture Awareness

This project uses **Gunicorn multi-worker** for HTTP + **separate scheduler process** (`run_scheduler`). Each is a distinct OS process with independent memory space.

**Key implications:**

1. **Python singletons are per-process.** Module-level singletons (e.g., `SchedulerManager`, APScheduler's `BackgroundScheduler`) exist independently in every process. Modifying one does NOT affect others.
2. **Signals are per-process.** Django signals fire in the process that handles the request. Signal handlers that modify in-memory state (e.g., `scheduler_manager.reload_job()`) have no effect on other processes.
3. **Cross-process coordination requires shared storage.** Use the database for cross-process communication (not cache/Redis):
   - **Status detection**: `SchedulerHeartbeat.last_heartbeat` written by scheduler process, checked by web workers via `is_alive()`.
   - **Notifications**: `SchedulerHeartbeat.reload_pending` set by web workers (CRUD signals + reload button), polled by scheduler process.
   - **Distributed locks**: use `RedisProxy().lock()` from `apps/common/redis_utils.py` (Redis only, optional).
4. **LocMemCache is per-process.** Django's `LocMemCache` is not shared between processes. Only `RedisCache` provides cross-process cache sharing. **Use DB instead of cache for any cross-process data that must work without Redis.**
5. **CacheOps** is automatically disabled when Redis is unavailable (`CACHEOPS_REDIS = None`).
6. **Never rely on in-process state for cross-process coordination.** If you need to communicate between processes, use the database (or Redis for performance-sensitive scenarios).

**Pattern to follow** (see SchedulerManager and SchedulerHeartbeat for reference):
```
# Web process: write notification to DB
MyModel.objects.update_or_create(id=FIXED_ID, defaults={"flag": True})

# Worker process: poll DB and consume
if MyModel.objects.filter(id=FIXED_ID, flag=True).exists():
    do_work()
    MyModel.objects.filter(id=FIXED_ID).update(flag=False)
```
### Menu Model Note
`Menu.path` field conflicts with treebeard MP_Node's internal `path` field. The tree structure is rebuilt on the frontend from flat menu data (by path prefix matching in `menuTree.ts`), so treebeard's tree operations (`add_root`, `add_child`) are not used. Menu creation in `init_data.py` uses direct ORM with manually set `depth`/`numchild` values.

### Spyne / SOAP Compatibility (Python 3.13+)

spyne 2.14.0 bundles six 1.14.0, whose `_SixMetaPathImporter` only implements the deprecated `find_module`/`load_module` (PEP 302). Python 3.13+ ignores `find_module` in favor of `find_spec` (PEP 451), causing `from spyne.util.six.moves.collections_abc import MutableSet` to fail.

**Fix**: `djangoadminx/common/spyne_compat.py` pre-loads spyne's bundled `six.py`, patches `_SixMetaPathImporter` with a `find_spec` method, and pre-registers known `six.moves.*` modules in `sys.modules`.

**Critical**: Every Django entry point MUST import this module **before** any spyne code loads:

```python
# manage.py / wsgi.py / asgi.py — top of the file
import djangoadminx.common.spyne_compat  # noqa: F401
```

**Troubleshooting**: If you see `ImportError: cannot import name 'Application' from 'spyne'`, the compat module's stub package cleanup failed (empty `spyne`/`spyne.util` in `sys.modules`). Check `spyne_compat.apply()` step 3b.

### note
- 禁止修改虚拟环境源码
- 