import os
import sys
from pathlib import Path

import environ

env = environ.Env(
    # 默认值
    DEBUG=(bool, False),
    ALLOWED_HOSTS=(list, []),
    CSRF_TRUSTED_ORIGINS=(list, []),
)

BASE_DIR = Path(__file__).resolve().parent.parent.parent

# 加载 .env
env_file = BASE_DIR / ".env"
if env_file.exists():
    env.read_env(str(env_file))

# ---------- 核心 ----------
SECRET_KEY = env("SECRET_KEY", default="django-insecure-change-me-in-production")
DEBUG = env("DEBUG", default=False)
ALLOWED_HOSTS = env("ALLOWED_HOSTS")
CSRF_TRUSTED_ORIGINS = env("CSRF_TRUSTED_ORIGINS")
ROOT_URLCONF = "config.urls"
WSGI_APPLICATION = "config.wsgi.application"
ASGI_APPLICATION = "config.asgi.application"

# ---------- 默认主键 ----------
DEFAULT_AUTO_FIELD = "django.db.models.BigAutoField"

# ---------- Apps ----------
DJANGO_APPS = [
    "django.contrib.admin",
    "django.contrib.auth",
    "django.contrib.contenttypes",
    "django.contrib.sessions",
    "django.contrib.messages",
    "django.contrib.staticfiles",
]

THIRD_PARTY_APPS = [
    "rest_framework",
    "rest_framework_simplejwt",
    "rest_framework_simplejwt.token_blacklist",
    "corsheaders",
    "django_filters",
    "drf_spectacular",
    "channels",
    "treebeard",
    "safedelete",
    "cacheops",
]

LOCAL_APPS = [
    "djangoadminx.accounts",
    "djangoadminx.menu",
    "djangoadminx.config_center",
    "djangoadminx.monitor",
    "djangoadminx.cluster",
    "djangoadminx.common",
    "djangoadminx.webservice",
    "djangoadminx.audit",
    "djangoadminx.file_center",
    "djangoadminx.policy",
    "djangoadminx.captcha",
    "djangoadminx.data_center",
    "djangoadminx.notification",
]

INSTALLED_APPS = DJANGO_APPS + THIRD_PARTY_APPS + LOCAL_APPS

# ---------- 中间件 ----------
MIDDLEWARE = [
    "corsheaders.middleware.CorsMiddleware",
    "django.middleware.security.SecurityMiddleware",
    "django.contrib.sessions.middleware.SessionMiddleware",
    "django.middleware.common.CommonMiddleware",
    "django.middleware.csrf.CsrfViewMiddleware",
    "django.contrib.auth.middleware.AuthenticationMiddleware",
    "django.contrib.messages.middleware.MessageMiddleware",
    "django.middleware.clickjacking.XFrameOptionsMiddleware",
    # 自定义中间件
    "djangoadminx.common.middleware.RequestContextMiddleware",
    "djangoadminx.common.middleware.UserActivityMiddleware",
    "djangoadminx.common.middleware.RequestLogMiddleware",
    "djangoadminx.common.middleware.IPBlockMiddleware",
]

# ---------- Fernet (配置加密) ----------
FERNET_KEY = env("FERNET_KEY", default=None)

# ---------- 数据库 ----------
DATABASES = {
    "default": env.db_url(
        "DATABASE_URL",
        default=f"sqlite:///{BASE_DIR / 'db.sqlite3'}",
    ),
}

# ---------- 缓存 & Redis ----------
REDIS_URL = env("REDIS_URL", default="redis://127.0.0.1:6379/0")

CACHES = {
    "default": {
        "BACKEND": "django.core.cache.backends.locmem.LocMemCache",
        "LOCATION": "djangoadminx-default",
    }
}

# Redis 可选 — 仅当 REDIS_URL 可连接时使用 django-redis
try:
    import redis
    r = redis.from_url(REDIS_URL)
    r.ping()
    CACHES["default"] = {
        "BACKEND": "django_redis.cache.RedisCache",
        "LOCATION": REDIS_URL,
        "OPTIONS": {"CLIENT_CLASS": "django_redis.client.DefaultClient"},
    }
except Exception:
    pass

# cacheops — 可选, 需要 Redis
CACHEOPS_REDIS = REDIS_URL
CACHEOPS_DEGRADE_ON_FAILURE = True
CACHEOPS_DEFAULT_TIMEOUT = 60 * 60
if not CACHES["default"]["BACKEND"].startswith("django_redis"):
    CACHEOPS_REDIS = None  # 无 Redis 时禁用 cacheops

# ---------- DRF ----------
REST_FRAMEWORK = {
    "DEFAULT_AUTHENTICATION_CLASSES": (
        "rest_framework_simplejwt.authentication.JWTAuthentication",
        "rest_framework.authentication.SessionAuthentication",
    ),
    "DEFAULT_PERMISSION_CLASSES": (
        "djangoadminx.accounts.permissions.RBACPermission",
    ),
    "DEFAULT_RENDERER_CLASSES": (
        "djangoadminx.common.renderers.StandardJsonRenderer",
        "rest_framework.renderers.BrowsableAPIRenderer",
    ),
    "DEFAULT_PAGINATION_CLASS": "djangoadminx.common.pagination.StandardPagination",
    "PAGE_SIZE": 10,
    "DEFAULT_FILTER_BACKENDS": (
        "django_filters.rest_framework.DjangoFilterBackend",
        "rest_framework.filters.SearchFilter",
        "rest_framework.filters.OrderingFilter",
    ),
    "DEFAULT_THROTTLE_CLASSES": [
        "rest_framework.throttling.AnonRateThrottle",
        "rest_framework.throttling.UserRateThrottle",
    ],
    "DEFAULT_THROTTLE_RATES": {
        "anon": env("THROTTLE_ANON", default="30/minute"),
        "user": env("THROTTLE_USER", default="200/minute"),
    },
    "DEFAULT_SCHEMA_CLASS": "drf_spectacular.openapi.AutoSchema",
    "EXCEPTION_HANDLER": "djangoadminx.common.exceptions.custom_exception_handler",
}

# ---------- JWT ----------
from datetime import timedelta

SIMPLE_JWT = {
    "ACCESS_TOKEN_LIFETIME": timedelta(minutes=env.int("JWT_ACCESS_EXPIRE", default=30)),
    "REFRESH_TOKEN_LIFETIME": timedelta(days=env.int("JWT_REFRESH_EXPIRE", default=7)),
    "ROTATE_REFRESH_TOKENS": True,
    "BLACKLIST_AFTER_ROTATION": True,
    "AUTH_HEADER_TYPES": ("Bearer",),
}

# ---------- Spectacular (API 文档) ----------
SPECTACULAR_SETTINGS = {
    "TITLE": "DjangoAdminX API",
    "DESCRIPTION": "企业级 Django Admin 框架 API",
    "VERSION": "1.0.0",
    "SERVE_INCLUDE_SCHEMA": False,
}

# ---------- CORS ----------
CORS_ALLOW_ALL_ORIGINS = DEBUG
CORS_ALLOWED_ORIGINS = env("CORS_ALLOWED_ORIGINS", default=[])
CORS_ALLOW_CREDENTIALS = True

# ---------- 静态 & 媒体 ----------
STATIC_URL = "static/"
STATIC_ROOT = BASE_DIR / "staticfiles"
MEDIA_URL = "media/"
MEDIA_ROOT = BASE_DIR / "media"

# 文件存储
FILE_STORAGE_BACKEND = env("FILE_STORAGE_BACKEND", default="local")  # local | minio
MINIO_ENDPOINT = env("MINIO_ENDPOINT", default="")
MINIO_ACCESS_KEY = env("MINIO_ACCESS_KEY", default="")
MINIO_SECRET_KEY = env("MINIO_SECRET_KEY", default="")
MINIO_BUCKET = env("MINIO_BUCKET", default="djangoadminx")
MINIO_SECURE = env.bool("MINIO_SECURE", default=True)

# ---------- 国际化 ----------
LANGUAGE_CODE = "zh-hans"
TIME_ZONE = "Asia/Shanghai"
USE_I18N = True
USE_TZ = True

# ---------- 模板 ----------
TEMPLATES = [
    {
        "BACKEND": "django.template.backends.django.DjangoTemplates",
        "DIRS": [],
        "APP_DIRS": True,
        "OPTIONS": {
            "context_processors": [
                "django.template.context_processors.debug",
                "django.template.context_processors.request",
                "django.contrib.auth.context_processors.auth",
                "django.contrib.messages.context_processors.messages",
            ],
        },
    },
]

# ---------- 认证 ----------
AUTH_USER_MODEL = "accounts.User"
AUTH_PASSWORD_VALIDATORS = [
    {"NAME": "django.contrib.auth.password_validation.MinimumLengthValidator"},
    {"NAME": "django.contrib.auth.password_validation.CommonPasswordValidator"},
    {"NAME": "django.contrib.auth.password_validation.NumericPasswordValidator"},
]

# 密码策略（可选开关）
PASSWORD_POLICY_ENABLED = env.bool("PASSWORD_POLICY_ENABLED", default=False)

# 验证码（可选开关）
CAPTCHA_ENABLED = env.bool("CAPTCHA_ENABLED", default=False)

# CAS 单点登录（可选）
# CAS_SERVER_URL = "https://cas.example.com/cas/"
CAS_SERVER_URL = env("CAS_SERVER_URL", default=None)
CAS_IGNORE_REFERER = True
CAS_LOGIN_MSG = None

# ---------- Channels ----------
if CACHES["default"]["BACKEND"].startswith("django_redis"):
    CHANNEL_LAYERS = {
        "default": {
            "BACKEND": "channels_redis.core.RedisChannelLayer",
            "CONFIG": {"hosts": [REDIS_URL]},
        },
    }
else:
    CHANNEL_LAYERS = {
        "default": {
            "BACKEND": "channels.layers.InMemoryChannelLayer",
        },
    }

# ---------- 日志 ----------
LOGGING = {
    "version": 1,
    "disable_existing_loggers": False,
    "formatters": {
        "standard": {
            "format": "{levelname} {asctime} {module} {message}",
            "style": "{",
        },
        "json": {
            "()": "pythonjsonlogger.jsonlogger.JsonFormatter",
            "format": "%(levelname)s %(asctime)s %(name)s %(module)s %(message)s",
        },
    },
    "handlers": {
        "console": {
            "level": "DEBUG",
            "class": "logging.StreamHandler",
            "formatter": "standard",
        },
        "file": {
            "level": "INFO",
            "class": "logging.handlers.TimedRotatingFileHandler",
            "filename": BASE_DIR / "logs" / "django.log",
            "when": "midnight",
            "backupCount": 30,
            "formatter": "standard",
            "delay": True,
        },
        "error_file": {
            "level": "ERROR",
            "class": "logging.handlers.TimedRotatingFileHandler",
            "filename": BASE_DIR / "logs" / "error.log",
            "when": "midnight",
            "backupCount": 90,
            "formatter": "standard",
            "delay": True,
        },
        "websocket": {
            "level": "INFO",
            "class": "logging.handlers.RotatingFileHandler",
            "filename": BASE_DIR / "logs" / "realtime.log",
            "maxBytes": 1024 * 1024 * 10,  # 10MB
            "backupCount": 5,
            "formatter": "standard",
            "delay": True,
        },
    },
    "loggers": {
        "django": {
            "handlers": ["console", "file", "error_file"],
            "level": "INFO",
            "propagate": False,
        },
        "django.request": {
            "handlers": ["error_file"],
            "level": "ERROR",
            "propagate": False,
        },
        "realtime": {
            "handlers": ["websocket"],
            "level": "INFO",
            "propagate": False,
        },
    },
}