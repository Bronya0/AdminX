from .base import *  # noqa: F403

DEBUG = True

# SQLite for local dev
DATABASES = {
    "default": {
        "ENGINE": "django.db.backends.sqlite3",
        "NAME": BASE_DIR / "db.sqlite3",  # noqa: F405
    }
}

CORS_ALLOW_ALL_ORIGINS = True

# 开发环境关闭登录锁定检测
LOGIN_LOCK_ENABLED = False