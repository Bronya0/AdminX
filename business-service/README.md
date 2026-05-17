# business-service — 三方业务对接示例

一个基于 FastAPI 的示例业务服务，演示第三方应用如何对接 DjangoAdminX 平台（JWT introspection 鉴权 + 菜单自动注册）。

## 架构说明

```
┌──────────────────────────────────────────────┐
│  DjangoAdminX 平台 (:8000)                    │
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

编辑 `config.py`：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PLATFORM_URL` | `http://localhost:8000` | DjangoAdminX 平台地址 |
| `PLATFORM_ADMIN_USER` | `admin` | 用于注册菜单的平台账号 |
| `PLATFORM_ADMIN_PASS` | `admin123` | 对应密码 |
| `SERVICE_PORT` | `8001` | 业务服务端口 |
| `SERVICE_URL` | `http://localhost:8001` | 业务服务地址（菜单跳转目标） |

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

注册的菜单会出现在 DjangoAdminX 侧边栏中，点击可跳转到业务服务页面。跳转时自动携带 `token` 参数，业务服务前端解析后用于后续 API 请求。

## JWT 鉴权流程

1. 用户在 DjangoAdminX 平台登录获取 JWT
2. 业务服务前端从 URL 参数或 localStorage 获取 token
3. 请求业务 API 时在 `Authorization` header 中携带 token
4. 业务服务中间件拦截 `/api/v1/posts/*` 请求，调用平台 `introspect` 接口校验
5. 校验通过后在 `request.state.user` 注入用户信息供后续使用

## 在 DjangoAdminX 中访问

在平台菜单中点击"业务管理"即可打开业务服务页面。如需直接在浏览器打开：

```
http://localhost:8001/?token=<jwt_token>&primary=1890ff&dark=0
```

页面会从 URL 读取平台主题色和暗黑模式状态，自动匹配平台外观。
