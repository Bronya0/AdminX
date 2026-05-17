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

# Redis (optional - see README "启用 Redis" section)
# Just set REDIS_URL in .env, no code changes needed. Auto-detected at startup.

# Tests
python manage.py test                   # All tests (80 total)
python manage.py test djangoadminx.accounts    # Single module

# Frontend (adminx-ui/)
cd adminx-ui && npm run dev             # Vite dev server (port 5173, proxies /api to :8000)
cd adminx-ui && npm run build           # Production build
cd adminx-ui && npm run type-check      # Vue/TS type checking (skip build)
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
│   ├── webservice/        #   APScheduler jobs + job logs
│   ├── file_center/       #   File upload (Local/MinIO)
│   ├── data_center/       #   Excel import/export
│   ├── captcha/           #   Captcha
│   ├── audit/             #   Audit log (mixin + signals)
│   ├── policy/            #   Password policy
│   └── common/            #   Middleware, unified response/exception/pagination, scheduler, SSE logs, crypto_utils
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
  - See [`SchedulerManager`](djangoadminx/common/scheduler.py) and [`SchedulerHeartbeat`](djangoadminx/webservice/models.py) for implementation.
- **Anti-pattern (fixed)**: Before this approach, signals and reload called `reload_job()`/`reload_all()` directly on `SchedulerManager` singleton — this only affected the calling process (one Gunicorn worker) and was silently ignored by the actual scheduler process. **Never rely on in-process state for cross-process coordination.**
- **Shell commands**: Use `shlex.split()` + `subprocess.run(shell=False)` to prevent command injection.
- **Security**: `ScheduleJobViewSet` requires `IsAdminUser` permission.

### Encryption (djangoadminx/common/crypto_utils.py)
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
- 