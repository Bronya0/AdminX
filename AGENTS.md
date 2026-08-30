# AGENTS.md

本仓库为 monorepo，是已迁移完成的 Go 技术栈版本：

- `backend-go/` — 唯一后端（Go 1.24 + Gin + GORM + PostgreSQL + Redis）。**旧 `backend-django/` Django 版已删除**，本文档不再包含 Django 内容。
- `adminx-ui/` — 前端（Vue 3 + TypeScript + Ant Design Vue 4 + Vite + Pinia）。
- `business-service/` — FastAPI 示例业务服务，演示第三方系统对接平台（JWT introspection + 菜单/组件注册）。

各目录有独立的 AGENTS.md（backend-go/AGENTS.md 含安全基线清单，改动后端前必读）。

## 常用命令

```bash
# Backend (backend-go/)
cd backend-go
go build ./...                 # 编译检查
go vet ./...                   # 静态检查（必须通过）
go test ./...                  # 测试
go run ./cmd/server            # 启动 HTTP 服务（默认 :8000）
go run ./cmd/server --scheduler  # 启动 + 内置调度器（生产建议独立进程轮询 reload_pending）
go run ./cmd/init-data         # 初始化种子数据（不传 --password 会生成随机超管密码并打印一次）
bash scripts/build.sh          # 生产构建到 bin/

# Frontend (adminx-ui/)
cd adminx-ui
npm run dev                    # Vite dev server（:5173，/adminx/api 代理到 :8000）
npm run build                  # 生产构建
npm run type-check             # Vue/TS 类型检查（需先 npm install）

# Business service (business-service/)
pip install -r requirements.txt && python main.py   # :8001
```

## 关键约定

### 路径与权限对齐（三方一致，改任何一处都要同步检查）
- 后端路由挂载在 `server.base_path`（默认 `/adminx`）下：`/adminx/api/v1/...`
- 前端 vite `base: '/adminx/'`，dev proxy 保留前缀直传（**不做 rewrite**）
- RBAC 种子 allowed_paths 按 API 路径书写（`/api/v1/...`），RBAC 中间件匹配前会剥掉 base_path

### 统一响应
- 所有响应 HTTP 恒 200，业务状态在 `code` 字段：`{code, msg, data}`（`pkg/response`）
- service 层返回 `*apperr.AppError`，handler 用 `response.Error` 统一转换

### 认证与安全（详见 backend-go/AGENTS.md 安全基线）
- JWT HS256：access 30m / refresh 7d；refresh 原子消费（SETNX）轮换防重放
- `last_logout` 二次失效：JWTAuth、Introspect、**Refresh 三处都必须校验**
- RBAC：superuser 全放行；普通用户按 user→role→menu→allowed_paths glob 匹配
- 高危端点（jobs 写/run_once、set_upgrade、menu create/register、policy 写）在 router 的 admin 组：JWT+RBAC+RequireSuperuser
- **super_admin 角色的授予/撤销仅限超级管理员**（`UserService.CheckRoleAssignment`，Create/Update 前必须调用）；`AssignRoles` 接受角色 ID 或名称混传
- 组件注册/心跳端点用 `security.component_secret`（X-Component-Token 头）保护
- 密码：bcrypt cost 12；策略（长度/复杂度/历史）在创建、重置、改密时统一执行；改密更新 last_logout 吊销旧 token
- 前端 token：access 存 localStorage，refresh 只存内存 + sessionStorage（不进 localStorage）

### 前端模式
- API 层 `adminx-ui/src/api/*.ts`（带类型）；状态 Pinia（`stores/user.ts` 持久化 token/theme）
- token 刷新：`utils/request.ts` 单飞 + 队列 + `_retried` 防死循环；login/refresh/logout 端点豁免重试
- 外链/iframe 的 token 通过 postMessage（`adminx:request-token` / `adminx:token`）下发，不进 URL；iframe URL 必须匹配用户菜单中 `menu_type === 'iframe'` 的条目
- 禁止 v-html/innerHTML 渲染用户可控内容

### 业务服务对接（business-service 为参考实现）
1. 启动时以平台管理员账号调 `/menu/register/` 注册菜单（按 code upsert）
2. 组件生命周期：`/cluster/components/register|heartbeat|unregister`（需 X-Component-Token）
3. 业务 API 用平台 `POST .../accounts/introspect/` 校验用户 JWT
4. 心跳响应可携带 `upgrade` 指令（url+SHA-256 checksum 必填），业务服务自校验后自行升级

## 测试
- Go：`cd backend-go && go test ./...`（jwt/middleware/service/model/captcha/crypto/errors）
- 前端：`cd adminx-ui && npm run type-check`（需先 npm install）
- business-service：`python3 -m py_compile *.py routers/*.py`

## note
- 禁止修改虚拟环境源码
- 根 README.md 为项目门面；本文件为 AI/开发者工作指南，两者冲突时以代码为准
