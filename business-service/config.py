"""业务服务配置 — 按需修改"""

import os

PLATFORM_URL = os.getenv("PLATFORM_URL", "http://localhost:8000")
"""AdminX 平台地址（与 Go 后端默认端口 8000 保持一致）"""

PLATFORM_ADMIN_USER = os.getenv("PLATFORM_ADMIN_USER", "admin")
"""用于注册菜单的平台管理员账号"""

PLATFORM_ADMIN_PASS = os.getenv("PLATFORM_ADMIN_PASS", "admin123")
"""对应密码"""

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

CORS_ORIGINS = os.getenv("CORS_ORIGINS", "*")
"""CORS 允许的来源，逗号分隔或 *"""

DEBUG = os.getenv("DEBUG", "true").lower() in ("true", "1", "yes")
"""调试模式"""

LOG_LEVEL = os.getenv("LOG_LEVEL", "DEBUG" if DEBUG else "INFO")
"""日志级别"""

