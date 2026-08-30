"""业务服务配置 — 按需修改"""

import os

PLATFORM_URL = os.getenv("PLATFORM_URL", "http://localhost:8000")
"""AdminX 平台地址（与 Go 后端默认端口 8000 保持一致）"""

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

INTROSPECT_CACHE_TTL = int(os.getenv("INTROSPECT_CACHE_TTL", "15"))
"""introspect 结果本地缓存秒数。缓存会延迟登出/封禁的生效时间，
默认 15 秒（此前 60 秒）；设为 0 关闭缓存。"""

CORS_ORIGINS = os.getenv("CORS_ORIGINS", "*")
"""CORS 允许的来源，逗号分隔或 *"""

DEBUG = os.getenv("DEBUG", "false").lower() in ("true", "1", "yes")
"""调试模式（默认关闭；开启才会暴露 /docs Swagger）"""

LOG_LEVEL = os.getenv("LOG_LEVEL", "DEBUG" if DEBUG else "INFO")
"""日志级别"""

