import logging
import time

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
            f"user={user} ip={request.META.get('REMOTE_ADDR', '')}"
        )
        return response


class IPBlockMiddleware:
    """IP 黑白名单检查 (结合 ConfigCenter 动态配置)"""

    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        from django.http import JsonResponse

        try:
            from django.apps import apps
            if not djangoadminx.ready:
                return self.get_response(request)
            from djangoadminx.config_center.models import Config

            client_ip = request.META.get("REMOTE_ADDR", "")

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
        except Exception:
            pass

        return self.get_response(request)


class UserActivityMiddleware:
    """更新已认证用户的最后活动时间 (用于在线状态判定)

    防抖策略：距上次更新超过 60 秒才写入，避免每个请求都写库。
    """

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
            # 防抖：如果距上次更新 < 60 秒，跳过
            last = getattr(user, 'last_activity', None)
            if last is None or (now - last).total_seconds() >= 60:
                from djangoadminx.accounts.models import User
                User.objects.filter(pk=user.pk).update(last_activity=now)
        except Exception:
            pass

        return response