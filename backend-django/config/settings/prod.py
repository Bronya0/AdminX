from .base import *  # noqa: F403

DEBUG = False

# 生产环境必须配置 SECRET_KEY
if SECRET_KEY == "django-insecure-change-me-in-production":  # noqa: F405
    raise RuntimeError(
        "生产环境必须设置 SECRET_KEY 环境变量！"
        "请在 .env 或系统环境中配置一个安全的随机密钥。"
    )

# 生产环境必须配置 ALLOWED_HOSTS
if not ALLOWED_HOSTS:  # noqa: F405
    raise RuntimeError("生产环境必须设置 ALLOWED_HOSTS 环境变量！")

# PostgreSQL for production
DATABASES = {
    "default": env.db_url("DATABASE_URL"),  # noqa: F405
}

# 安全配置
SESSION_COOKIE_SECURE = True
SESSION_COOKIE_HTTPONLY = True
SESSION_COOKIE_SAMESITE = "Lax"
CSRF_COOKIE_SECURE = True
SECURE_CONTENT_TYPE_NOSNIFF = True
X_FRAME_OPTIONS = "DENY"
SECURE_SSL_REDIRECT = True
SECURE_HSTS_SECONDS = 31536000
SECURE_HSTS_INCLUDE_SUBDOMAINS = True
SECURE_HSTS_PRELOAD = True
SECURE_PROXY_SSL_HEADER = ("HTTP_X_FORWARDED_PROTO", "https")
SECURE_REFERRER_POLICY = "same-origin"

# 生产环境在反向代理后面，信任 X-Forwarded-For
TRUST_PROXY_HEADERS = True

LOGGING["handlers"]["file"]["level"] = "INFO"  # noqa: F405
LOGGING["handlers"]["error_file"]["level"] = "ERROR"  # noqa: F405
LOGGING["handlers"]["console"]["level"] = "WARNING"  # noqa: F405

# 生产环境开启登录锁定
LOGIN_LOCK_ENABLED = True

# 生产环境强制 Redis —— 集群下限流/缓存/Channel Layer 必须共享，
# 降级到 LocMemCache 会导致每节点独立计数、配置缓存不一致等严重问题。
# base.py 的探测逻辑保证：Redis 可用时 CACHES 即 django_redis，否则为 LocMemCache。
if CACHES["default"]["BACKEND"] != "django_redis.cache.RedisCache":  # noqa: F405
    raise RuntimeError(
        "生产环境必须配置可用的 REDIS_URL（当前 Redis 不可达，缓存降级到了 LocMemCache）。"
        "集群部署下限流、配置缓存、WebSocket 通道层都依赖共享 Redis。"
    )