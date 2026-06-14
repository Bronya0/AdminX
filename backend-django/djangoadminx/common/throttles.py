"""自定义限速类"""
from rest_framework.throttling import SimpleRateThrottle


class IntrospectThrottle(SimpleRateThrottle):
    """TokenIntrospectView 专用限速

    使用 Django 默认 cache（生产环境为 Redis，开发为 LocMemCache）。
    多进程部署时只有 Redis cache 才能保证跨进程计数准确。
    如果没有 Redis，限速仍会生效但仅限单进程内。

    速率通过 settings.REST_FRAMEWORK["DEFAULT_THROTTLE_RATES"]["introspect"] 配置，
    默认 300/min（5 次/秒），足够业务容器高频调用。
    """

    scope = "introspect"

    def get_cache_key(self, request, view):
        # 匿名调用（业务容器没有用户上下文），按 IP 限速
        return self.cache_format % {
            "scope": self.scope,
            "ident": self.get_ident(request),
        }
