# business-service — 三方业务对接示例

一个基于 FastAPI 的示例业务服务，演示第三方应用如何对接 AdminX 平台（JWT introspection 鉴权 + 菜单自动注册）。

## 架构说明

```
┌──────────────────────────────────────────────┐
│  AdminX 平台 (:8000)                    │
│  - JWT 颁发 & introspection                  │
│  - 菜单管理                                   │
│  - RBAC 权限                                  │
└────────────┬─────────────────────▲───────────┘
             │ JWT introspection    │ 菜单注册
             ▼                     │
┌──────────────────────────────────┴───────────┐
│  business-service (:8001)                     │
│  - FastAPI 业务服务                            │
│  - posts CRUD（示例业务）                      │
│  - 菜单自动注册（启动时或手动触发）             │
│  - JWT 鉴权中间件                              │
└──────────────────────────────────────────────┘
```

## 启动

```bash
cd business-service
pip install -r requirements.txt
python main.py
```

服务默认运行在 `http://localhost:8001`。

## 配置

编辑 `config.py`（或用同名大写环境变量覆盖）：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PLATFORM_URL` | `http://localhost:8000` | AdminX 平台地址 |
| `PLATFORM_BASE_PATH` | `/adminx` | 平台 API 前缀（Go 后端 server.base_path） |
| `PLATFORM_API_BASE` | 自动拼接 | 平台 API 根地址（可整串覆盖） |
| `PLATFORM_ADMIN_USER` | `admin` | 用于注册菜单的平台账号 |
| `PLATFORM_ADMIN_PASS` | （空） | 对应密码，**必须显式提供**，无默认弱口令 |
| `COMPONENT_SECRET` | （空） | 与平台 `security.component_secret` 一致的共享密钥（X-Component-Token） |
| `SERVICE_PORT` | `8001` | 业务服务端口 |
| `SERVICE_URL` | `http://localhost:8001` | 业务服务地址（菜单跳转目标） |
| `INTROSPECT_CACHE_TTL` | `15` | introspect 结果缓存秒数（0 关闭；缓存会延迟登出生效） |
| `DEBUG` | `false` | 调试模式（开启才暴露 /docs） |

## API 端点

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| `POST` | `/api/v1/register/` | 手动注册菜单到平台 | — |
| `POST` | `/api/v1/unregister/` | 手动注销菜单 | — |
| `GET` | `/api/v1/posts/` | 文章列表 | JWT |
| `POST` | `/api/v1/posts/` | 新建文章 | JWT |
| `PUT` | `/api/v1/posts/{id}/` | 更新文章 | JWT |
| `DELETE` | `/api/v1/posts/{id}/` | 删除文章 | JWT |
| `GET` | `/docs` | Swagger 文档 | — |

> `/api/v1/posts/*` 路径受 JWT introspection 中间件保护，需在请求头携带 `Authorization: Bearer <token>`。

## 菜单注册

服务启动时自动调用 `register_menu()` 向平台注册菜单（幂等），或手动触发：

```bash
curl -X POST http://localhost:8001/api/v1/register/
```

注册的菜单会出现在 AdminX 侧边栏中，点击可跳转到业务服务页面。

## JWT 鉴权流程

1. 用户在 AdminX 平台登录获取 JWT
2. 平台通过 **postMessage** 下发 token：业务页面加载后发送
   `{ type: 'adminx:request-token' }` 给 parent/opener，平台校验来源后回传
   `{ type: 'adminx:token', token }`。token 不经 URL 传递（hash/query 会进
   浏览器历史与日志）；兼容读取旧 `business_token` localStorage 缓存
3. 请求业务 API 时在 `Authorization` header 中携带 token
4. 业务服务中间件拦截 `/api/v1/posts/*` 请求，调用平台 `introspect` 接口校验
5. 校验通过后在 `request.state.user` 注入用户信息供后续使用

## 在 AdminX 中访问

在平台菜单中点击"业务管理"即可打开业务服务页面（token 由平台通过
postMessage 自动下发，无需手工拼接）。直接在浏览器打开时：

```
http://localhost:8001/?primary=1890ff&dark=0
```

页面会从 URL 读取平台主题色和暗黑模式状态，自动匹配平台外观
（此时无平台下发的 token，需先在平台登录后经页面跳转进入）。
