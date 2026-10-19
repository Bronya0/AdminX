# business-service — 独立业务接入示例

FastAPI + 原生 JavaScript，独立运行、独立部署。AdminX 负责登录、导航和主题，业务页面只管理文章内容。支持 iframe 和平台打开的新标签页，不需要把业务页面合并进 AdminX 前端。

## 本地启动

先启动 AdminX 后端（默认 8000）和前端（默认 5173），在平台登录。

```bash
cd business-service
python -m venv .venv
source .venv/Scripts/activate  # Windows Git Bash；Linux 使用 .venv/bin/activate
pip install -r requirements.txt

export PLATFORM_FRONTEND_URL='http://localhost:5173/adminx/'
export PLATFORM_URL='http://localhost:8000'
export SERVICE_URL='http://localhost:8001'
export COMPONENT_SECRET='与平台 security.component_secret 相同的密钥'
# 菜单注册使用超管 access token，只在服务端配置；过期后重新提供。
export PLATFORM_ADMIN_TOKEN='平台登录后取得的超管 access token'
python main.py
```

平台开启验证码时，自动账号密码登录无法完成验证码交互，使用上面的 `PLATFORM_ADMIN_TOKEN`。如平台允许非交互登录，也可配置 `PLATFORM_ADMIN_USER` / `PLATFORM_ADMIN_PASS`。

启动后注册“业务系统（内嵌）”和“业务系统（新标签）”两个菜单；普通用户需要先由平台管理员分配对应菜单。启动注册失败不会阻止业务 HTTP 服务启动，排除配置问题后可手动重试：

```bash
curl -X POST http://localhost:8001/api/v1/register/ \
  -H "X-Component-Token: $COMPONENT_SECRET"
```

手动注销菜单使用 `POST /api/v1/unregister/`，同样需要组件密钥。不要把管理员 token、账号密码或组件密钥放进前端配置。

## 已实现的接入行为

- 页面握手后获取当前用户、可导航菜单和主题。access token 只保存在业务页面内存中，旧的 `business_token` 持久化缓存会被删除。
- 业务 API 返回 401 时向平台续期，并发请求合并，原请求最多重试一次。refresh token 始终留在平台。
- 平台退出时业务页面立即清空内容、草稿和 token，并取消请求。业务内真实交互每 5 秒最多上报一次活动，供平台空闲检测使用。
- 同步主色、明暗模式、字号、圆角；平台预览主题时当前业务页面也同步变化。
- 普通编辑表单在业务内容区显示；删除、放弃修改使用平台确认框。
- 通过菜单 code 跳转，平台只允许当前用户菜单里的目标。不能要求平台跳到任意 URL。
- 分页、筛选和编辑中的文章 ID 写入业务 hash；iframe 同步到平台 URL 的 `businessRoute`。刷新或分享完整平台地址可恢复页面。草稿内容不写 URL。
- 切换平台菜单、关闭或刷新页面时提示未保存内容。iframe 内可用区固定，业务自行滚动。
- 平台有就绪握手、45 秒加载超时和重试入口；直接打开业务地址会提示从平台进入。

示例采用替换历史记录同步业务 hash，不承诺逐条业务操作对应浏览器后退。切换业务不缓存 iframe，文章数据是进程内存演示数据，服务重启会丢失。

## 配置

| 环境变量 | 默认值 | 用途 |
|---|---|---|
| `PLATFORM_URL` | `http://localhost:8000` | 业务后端访问平台的地址 |
| `PLATFORM_BASE_PATH` | `/adminx` | 平台后端前缀 |
| `PLATFORM_API_BASE` | 自动拼接 | 可整串覆盖平台 API 根地址 |
| `PLATFORM_FRONTEND_URL` | `http://localhost:5173/adminx/` | 浏览器访问平台的入口；生产必须修改 |
| `PLATFORM_FRONTEND_ORIGINS` | 前端 URL 与平台 URL 的 origin | 允许通信、嵌入本页面的平台来源，逗号分隔，禁止 `*` |
| `PLATFORM_ADMIN_TOKEN` | 空 | 注册菜单用的超管 access token |
| `PLATFORM_ADMIN_USER` / `PLATFORM_ADMIN_PASS` | `admin` / 空 | 未配置 token 时的注册登录方式 |
| `COMPONENT_SECRET` | 空 | 组件及手动注册端点共享密钥 |
| `SERVICE_PORT` | `8001` | 业务监听端口 |
| `SERVICE_URL` | `http://localhost:8001` | 浏览器可访问的业务 URL，可含网关子路径 |
| `INTROSPECT_CACHE_TTL` | `0` | 默认不缓存认证；开启缓存会延迟退出/封禁生效，不超过 access token 到期时间 |
| `DEBUG` | `false` | 开启 Swagger 文档 |
| `CORS_ORIGINS` | `*` | 业务 API 的 CORS；页面与 API 同源时不需要跨域 |

`localhost`、`127.0.0.1`、不同端口属于不同 origin。浏览器实际入口必须在 `PLATFORM_FRONTEND_ORIGINS` 中；内部容器服务名不应当作浏览器入口。

生产可使用独立业务域名，也可通过网关把 `/apps/demo/` 代理到本服务根目录。保留浏览器端路径的末尾 `/`，页面资源与 API 使用相对路径。平台与业务统一 HTTPS，网关不要覆盖业务页面的 `frame-ancestors` 为禁止嵌入。

## 在自己的业务项目中复用

`static/adminx-bridge.js` 是无框架依赖的小 SDK。服务端把可信来源写到页面里的 `adminx-config` JSON 节点，再加载 SDK，不能从 URL 参数接受可信平台来源。

```js
const platform = window.AdminXBridge
platform.on('theme', applyTheme)
platform.on('logout', clearBusinessState)
const context = await platform.start()
await platform.ready()                         // 页面已完成初始化
const data = await platform.request('api/v1/posts/')
const accepted = await platform.confirm('确定删除？')
await platform.navigate('已授权的平台菜单code')
await platform.dirty(true)
await platform.route('#post=123')
```

协议版本为 1：请求 `adminx:request` 带 `id/method/payload`，响应 `adminx:response` 带相同 `id` 与 `result` 或 `error`。平台事件 `adminx:event` 包括 `theme/logout/disconnect`。双方都验证消息 `origin` 和窗口 `source`，敏感消息只向准确 origin 发送。平台保留旧 `adminx:request-token` 协议以兼容原接入页。

## 后端鉴权与数据边界

文章 API 由平台 JWT introspection 校验身份，按创建者隔离数据。平台不可达返回 502，token 无效返回 401，避免网络故障触发无意义续期。

**菜单白名单控制平台入口，不会自动拦截直达业务后端的请求。** 此示例没有把平台的接口 RBAC 自动扩展到业务服务；正式业务仍须在后端实现自己的授权规则。前端收到的菜单或用户资料不能充当后端授权依据。

## 验证

```bash
# 在 business-service 目录运行
python -m unittest discover -s tests -p 'test_*.py'
node --test tests/bridge.test.cjs
python -m py_compile *.py routers/*.py

# 平台前端
cd ../adminx-ui
npm run type-check
npm run build-only
```

测试覆盖真实 ASGI 路由的认证错误/网络错误区分、文章归属隔离、页面可信来源配置，以及 SDK 来源校验、并发续期、一次重试和退出竞态。

`tests/browser_smoke.cjs` 可选地验证真实平台页面与业务页面之间的通信，认证和业务 API 使用测试数据，不需要真实管理员账号。启动 Vite 在 `localhost:5179`，业务服务在 `localhost:18001`，并设置业务的 `PLATFORM_FRONTEND_URL=http://localhost:5179/adminx/`。准备 `playwright-core` 与 Chromium 后运行：

```bash
PLAYWRIGHT_MODULE='/绝对路径/playwright-core' \
PLAYWRIGHT_CHROMIUM_EXECUTABLE='/绝对路径/chrome' \
node tests/browser_smoke.cjs
```

浏览器测试覆盖首次握手、主题变化、并发续期、未保存确认、详情地址刷新恢复、全局确认框、新窗口退出和直接访问提示。
