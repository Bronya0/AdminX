"""业务服务配置 — 按需修改"""

import os

PLATFORM_URL = os.getenv("PLATFORM_URL", "http://localhost:8000")
"""DjangoAdminX 平台地址"""

PLATFORM_ADMIN_USER = os.getenv("PLATFORM_ADMIN_USER", "admin")
"""用于注册菜单的平台管理员账号"""

PLATFORM_ADMIN_PASS = os.getenv("PLATFORM_ADMIN_PASS", "admin123")
"""对应密码"""

SERVICE_PORT = int(os.getenv("SERVICE_PORT", "8001"))
"""业务服务自身端口"""

SERVICE_URL = os.getenv("SERVICE_URL", f"http://localhost:{SERVICE_PORT}")
"""业务服务自身地址（用作菜单 path 跳转目标）"""

CORS_ORIGINS = os.getenv("CORS_ORIGINS", "*")
"""CORS 允许的来源，逗号分隔或 *"""

DEBUG = os.getenv("DEBUG", "true").lower() in ("true", "1", "yes")
"""调试模式"""

LOG_LEVEL = os.getenv("LOG_LEVEL", "DEBUG" if DEBUG else "INFO")
"""日志级别"""
