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
scripts\start-win.bat                   # Dev startup (Windows)
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
└── requirements/          # base.txt / dev.txt / prod.txt
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
- Job definitions stored in DB (`ScheduleJob` model), signals trigger reload on CRUD.
- Supports `cron` / `interval` / `date` triggers.

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

### Menu Model Note
`Menu.path` field conflicts with treebeard MP_Node's internal `path` field. The tree structure is rebuilt on the frontend from flat menu data (by path prefix matching in `menuTree.ts`), so treebeard's tree operations (`add_root`, `add_child`) are not used. Menu creation in `init_data.py` uses direct ORM with manually set `depth`/`numchild` values.
