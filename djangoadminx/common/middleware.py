import logging
import time

from django.conf import settings

from djangoadminx.common.ip_utils import get_client_ip

logger = logging.getLogger("djangoadminx.request")


class RequestContextMiddleware:
    """将当前请求存入 thread-local — 供审计日志等模块获取操作人"""

    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        from djangoadminx.audit.signals import set_current_request
        set_current_request(request)
        return self.get_response(request)


class RequestLogMiddleware:
    """记录每个请求的路径、耗时、用户"""

    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        start = time.time()
        response = self.get_response(request)
        cost = (time.time() - start) * 1000
        user = getattr(request.user, "username", "anonymous") if hasattr(request, "user") else "anonymous"
        logger.info(
            f"[{cost:.0f}ms] {request.method} {request.path} "
            f"user={user} ip={get_client_ip(request)}"
        )
        return response


class IPBlockMiddleware:
    """IP 黑白名单检查 (结合 ConfigCenter 动态配置)"""

    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        from django.http import JsonResponse

        try:
            from django.apps import apps as django_apps
            if not django_apps.ready:
                return self.get_response(request)
            from djangoadminx.config_center.models import Config

            client_ip = get_client_ip(request)

            whitelist = Config.get_value("IP_WHITELIST", default="")
            if whitelist:
                allowed = [ip.strip() for ip in whitelist.split(",") if ip.strip()]
                if allowed and client_ip not in allowed:
                    return JsonResponse({"code": 403, "msg": "IP not allowed"})

            blacklist = Config.get_value("IP_BLACKLIST", default="")
            if blacklist:
                blocked = [ip.strip() for ip in blacklist.split(",") if ip.strip()]
                if client_ip in blocked:
                    return JsonResponse({"code": 403, "msg": "IP blocked"})
        except Exception as e:
            logger.warning(f"IPBlockMiddleware 检查异常: {e}")

        return self.get_response(request)


class UserActivityMiddleware:
    """更新已认证用户的最后活动时间 (用于在线状态判定)"""

    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        response = self.get_response(request)

        user = getattr(request, 'user', None)
        if user is None or not user.is_authenticated:
            return response

        try:
            from django.utils import timezone
            now = timezone.now()
            last = getattr(user, 'last_activity', None)
            if last is None or (now - last).total_seconds() >= 60:
                from djangoadminx.accounts.models import User
                User.objects.filter(pk=user.pk).update(last_activity=now)
        except Exception as e:
            logger.warning(f"更新 last_activity 失败: {e}")

        return response
