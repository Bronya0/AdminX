from .base import *  # noqa: F403

DEBUG = False

# 生产环境必须配置 SECRET_KEY
if SECRET_KEY == "django-insecure-change-me-in-production":  # noqa: F405
    raise RuntimeError(
        "生产环境必须设置 SECRET_KEY 环境变量！"
        "请在 .env 或系统环境中配置一个安全的随机密钥。"
    )

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