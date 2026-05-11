from .base import *  # noqa: F403

DEBUG = False

# PostgreSQL for production
DATABASES = {
    "default": env.db_url("DATABASE_URL"),  # noqa: F405
}

# 安全配置
SESSION_COOKIE_SECURE = True
CSRF_COOKIE_SECURE = True
SECURE_BROWSER_XSS_FILTER = True
SECURE_CONTENT_TYPE_NOSNIFF = True
X_FRAME_OPTIONS = "DENY"

LOGGING["handlers"]["file"]["level"] = "INFO"  # noqa: F405
LOGGING["handlers"]["error_file"]["level"] = "ERROR"  # noqa: F405

# 生产环境开启登录锁定
LOGIN_LOCK_ENABLED = True