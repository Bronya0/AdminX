"""业务服务配置 — 按需修改"""

import os
from urllib.parse import urlsplit

PLATFORM_URL = os.getenv("PLATFORM_URL", "http://localhost:8000")
"""AdminX 平台地址（与 Go 后端默认端口 8000 保持一致）"""

PLATFORM_FRONTEND_URL = os.getenv("PLATFORM_FRONTEND_URL", "http://localhost:5173/adminx/")
"""浏览器访问的平台前端地址；生产环境必须改成实际入口。"""


def _origin(value: str) -> str:
    parsed = urlsplit(value.strip())
    if parsed.scheme not in ("http", "https") or not parsed.hostname or parsed.username or parsed.password or any(ch.isspace() for ch in value):
        raise ValueError("平台地址必须是合法的 HTTP(S) 地址")
    port = parsed.port
    host = f"[{parsed.hostname}]" if ":" in parsed.hostname else parsed.hostname
    return f"{parsed.scheme}://{host}" + (f":{port}" if port and (parsed.scheme, port) not in (("http", 80), ("https", 443)) else "")


PLATFORM_FRONTEND_ORIGINS = list(dict.fromkeys(
    _origin(value) for value in os.getenv(
        "PLATFORM_FRONTEND_ORIGINS", f"{PLATFORM_FRONTEND_URL},{PLATFORM_URL}"
    ).split(",") if value.strip()
))
_origin(PLATFORM_FRONTEND_URL)
if not PLATFORM_FRONTEND_ORIGINS:
    raise ValueError("PLATFORM_FRONTEND_ORIGINS 不能为空")

PLATFORM_BASE_PATH = os.getenv("PLATFORM_BASE_PATH", "/adminx")
"""平台 API 前缀（Go 后端 server.base_path，默认 /adminx）"""

PLATFORM_API_BASE = os.getenv(
    "PLATFORM_API_BASE", f"{PLATFORM_URL.rstrip('/')}{PLATFORM_BASE_PATH}/api/v1"
)
"""平台 API 根地址（自动拼接 base_path，也可整串覆盖）"""

PLATFORM_ADMIN_USER = os.getenv("PLATFORM_ADMIN_USER", "admin")
"""用于注册菜单的平台管理员账号"""

PLATFORM_ADMIN_PASS = os.getenv("PLATFORM_ADMIN_PASS", "")
"""对应密码（必须通过环境变量显式提供，不再内置 admin123 默认值）"""

PLATFORM_ADMIN_TOKEN = os.getenv("PLATFORM_ADMIN_TOKEN", "")
"""可选：注册菜单时使用的超管 access token，适用于平台登录要求验证码的场景。仅配置在服务端。"""

COMPONENT_SECRET = os.getenv("COMPONENT_SECRET", "")
"""与平台 security.component_secret 一致的共享密钥（X-Component-Token 头）。
平台配置了该密钥后必须设置，否则组件注册/心跳会被拒绝。"""

SERVICE_PORT = int(os.getenv("SERVICE_PORT", "8001"))
"""业务服务自身端口"""

SERVICE_URL = os.getenv("SERVICE_URL", f"http://localhost:{SERVICE_PORT}")
"""业务服务自身地址（用作菜单 path 跳转目标）"""

APP_LABEL = os.getenv("APP_LABEL", "business-service")
"""组件唯一标识，平台注册时使用"""

APP_NAME = os.getenv("APP_NAME", "业务服务")
"""组件显示名称"""

APP_VERSION = os.getenv("APP_VERSION", "1.0.0")
"""当前版本号，心跳时上报"""

INTROSPECT_CACHE_TTL = int(os.getenv("INTROSPECT_CACHE_TTL", "0"))
"""introspect 结果本地缓存秒数。缓存会延迟登出/封禁的生效时间，
默认关闭，确保演示中的登出和封禁即时生效。"""

CORS_ORIGINS = os.getenv("CORS_ORIGINS", "*")
"""CORS 允许的来源，逗号分隔或 *"""

DEBUG = os.getenv("DEBUG", "false").lower() in ("true", "1", "yes")
"""调试模式（默认关闭；开启才会暴露 /docs Swagger）"""

LOG_LEVEL = os.getenv("LOG_LEVEL", "DEBUG" if DEBUG else "INFO")
"""日志级别"""

