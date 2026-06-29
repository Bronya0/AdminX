import os
import json
import logging

from django.conf import settings
from django.http import JsonResponse, StreamingHttpResponse
from django.views.decorators.http import require_GET
from rest_framework.decorators import api_view, permission_classes
from rest_framework.permissions import AllowAny, IsAdminUser


_SITE_FIELD_MAP = {
    "site_name": ("SITE_NAME", "string"),
    "site_desc": ("SITE_DESC", "string"),
    "site_logo": ("SITE_LOGO", "string"),
    "favicon": ("SITE_FAVICON", "string"),
    "idle_timeout": ("SESSION_IDLE_TIMEOUT", "int"),
    "login_bg_image": ("LOGIN_BG_IMAGE", "string"),
    "app_version": ("APP_VERSION", "string"),
}


@api_view(["GET", "POST"])
@permission_classes([AllowAny])
def site_info(request):
    """站点信息 — GET 公开读取，POST 管理员更新"""
    from djangoadminx.config_center.models import Config

    if request.method == "POST":
        if not request.user or not request.user.is_authenticated or not request.user.is_staff:
            return JsonResponse({"code": 403, "msg": "权限不足"})
        for field, (key, value_type) in _SITE_FIELD_MAP.items():
            if field in request.data:
                val = request.data[field]
                Config.objects.update_or_create(
                    key=key,
                    defaults={"value": str(val), "value_type": value_type, "is_active": True, "group": "site"},
                )
        return JsonResponse({"code": 200, "msg": "保存成功", "data": {}})

    return JsonResponse({
        "code": 200,
        "msg": "success",
        "data": {
            "site_name": Config.get_value("SITE_NAME", default="DjangoAdminX"),
            "site_desc": Config.get_value("SITE_DESC", default="企业级 Django Admin 框架"),
            "site_logo": Config.get_value("SITE_LOGO", default=""),
            "site_theme_color": Config.get_value("SITE_THEME_COLOR", default="#1890ff"),
            "idle_timeout": Config.get_value("SESSION_IDLE_TIMEOUT", default=30),
            "login_bg_image": Config.get_value("LOGIN_BG_IMAGE", default=""),
            "app_version": Config.get_value("APP_VERSION", default="1.0.0"),
            "favicon": Config.get_value("SITE_FAVICON", default=""),
        },
    })


@api_view(["GET"])
@permission_classes([AllowAny])
def health_check(request):
    """健康检查"""
    import psutil

    data = {
        "status": "ok",
        "cpu_percent": psutil.cpu_percent(interval=1),
        "memory": psutil.virtual_memory()._asdict(),
        "disk": psutil.disk_usage("/")._asdict(),
        "network": psutil.net_io_counters()._asdict(),
    }
    return JsonResponse({"code": 200, "msg": "success", "data": data})


@require_GET
def log_tail(request):
    """SSE 实时日志推送（仅管理员可访问）"""
    # 手动 JWT 鉴权（SSE 无法走 DRF 标准流程）
    from rest_framework_simplejwt.authentication import JWTAuthentication
    try:
        auth = JWTAuthentication()
        user, _ = auth.authenticate(request)
        if not user or not user.is_staff:
            return JsonResponse({"code": 403, "msg": "权限不足"}, status=200)
    except Exception:
        return JsonResponse({"code": 401, "msg": "未授权"}, status=200)

    log_file = settings.BASE_DIR / "logs" / "realtime.log"

    def event_stream():
        # 先发送文件已有内容
        if log_file.exists():
            with open(log_file, "r", encoding="utf-8") as f:
                for line in f:
                    yield f"data: {json.dumps({'line': line.rstrip()})}\n\n"

        # 持续 tail
        with open(log_file, "r", encoding="utf-8") as f:
            f.seek(0, os.SEEK_END)
            while True:
                line = f.readline()
                if line:
                    yield f"data: {json.dumps({'line': line.rstrip()})}\n\n"
                else:
                    import time
                    time.sleep(0.5)

    return StreamingHttpResponse(
        event_stream(),
        content_type="text/event-stream",
    )


# ── NTP 时间同步 ──

@api_view(["GET", "POST"])
def ntp_sync(request):
    """查询 NTP 状态 / 手动触发同步"""
    from djangoadminx.common.ntp import sync_time
    from djangoadminx.config_center.models import Config

    server = Config.get_value("NTP_SERVER", default="")
    enabled = Config.get_value("NTP_SYNC_ENABLED", default=False)

    if not enabled or not server:
        return JsonResponse({
            "code": 200, "msg": "success",
            "data": {"enabled": False, "server": server or "(未配置)"},
        })

    if request.method == "POST":
        result = sync_time(server)
        return JsonResponse({"code": 200, "msg": "success", "data": result})

    # GET: 返回状态不真正同步（手动同步用 POST）
    return JsonResponse({
        "code": 200, "msg": "success",
        "data": {"enabled": True, "server": server},
    })


@api_view(["GET"])
def dashboard_stats(request):
    """仪表盘统计数据（聚合各模块计数，避免前端直接调用各模块 list 接口）"""
    from django.contrib.auth import get_user_model
    from djangoadminx.accounts.models import Role
    from djangoadminx.menu.models import Menu
    from djangoadminx.accounts.models import UserLoginLog
    from djangoadminx.cluster.models import ClusterNode
    from djangoadminx.monitor.utils import SystemMonitor

    User = get_user_model()

    # 各模块计数（单个模块失败不影响其余）
    def safe_int(fn):
        try:
            return fn()
        except Exception:
            return 0

    user_count = safe_int(lambda: User.objects.all().count())
    role_count = safe_int(lambda: Role.objects.count())
    menu_count = safe_int(lambda: Menu.objects.filter(is_active=True).count())

    # 资源监控
    try:
        resources = SystemMonitor.all()
    except Exception:
        resources = {}
    cpu_usage = resources.get("cpu", {}).get("percent", 0)
    cpu_cores = resources.get("cpu", {}).get("count", 0)
    memory = resources.get("memory", {})
    memory_usage = memory.get("percent", 0)
    memory_total = memory.get("total", 0)
    memory_used = memory.get("used", 0)

    # 集群概况
    online_nodes = safe_int(lambda: ClusterNode.objects.filter(status="online").count())
    offline_nodes = safe_int(lambda: ClusterNode.objects.filter(status="offline").count())
    maintenance_nodes = safe_int(lambda: ClusterNode.objects.filter(status="maintenance").count())

    # 最近登录记录
    try:
        recent_logs = list(
            UserLoginLog.objects.values("username", "ip", "success", "message", "created_at")
            .order_by("-created_at")[:5]
        )
    except Exception:
        recent_logs = []

    return JsonResponse({
        "code": 200,
        "msg": "success",
        "data": {
            "user_count": user_count,
            "role_count": role_count,
            "menu_count": menu_count,
            "cpu_usage": cpu_usage,
            "cpu_cores": cpu_cores,
            "memory_usage": memory_usage,
            "memory_total": memory_total,
            "memory_used": memory_used,
            "online_nodes": online_nodes,
            "offline_nodes": offline_nodes,
            "maintenance_nodes": maintenance_nodes,
            "recent_logs": recent_logs,
        },
    })


@api_view(["GET"])
@permission_classes([IsAdminUser])
def system_components(request):
    """系统内置组件状态（DB / Redis / 调度器）"""
    import django
    import sys
    import platform

    # DB
    db_status = "ok"
    db_msg = ""
    try:
        from django.db import connection
        connection.ensure_connection()
    except Exception as e:
        db_status = "error"
        db_msg = str(e)

    # Redis / Cache
    cache_status = "ok"
    cache_backend = "locmem"
    cache_msg = ""
    try:
        from django.core.cache import cache
        cache.set("__ping__", "1", 5)
        backend_cls = type(cache).__name__
        if "Redis" in backend_cls:
            cache_backend = "redis"
        elif "Memcache" in backend_cls:
            cache_backend = "memcache"
    except Exception as e:
        cache_status = "error"
        cache_msg = str(e)

    # 调度器
    from djangoadminx.common.scheduler import SchedulerManager
    scheduler_alive = SchedulerManager.is_alive()

    return JsonResponse({
        "code": 200,
        "msg": "success",
        "data": {
            "platform": {
                "python_version": sys.version.split()[0],
                "django_version": django.__version__,
                "os": platform.system() + " " + platform.release(),
            },
            "components": [
                {
                    "key": "database",
                    "name": "数据库",
                    "type": "system",
                    "status": "ok",
                    "message": db_msg or "连接正常",
                } if db_status == "ok" else {
                    "key": "database",
                    "name": "数据库",
                    "type": "system",
                    "status": "error",
                    "message": db_msg,
                },
                {
                    "key": "cache",
                    "name": "缓存服务",
                    "type": "system",
                    "status": cache_status,
                    "backend": cache_backend,
                    "message": cache_msg or cache_backend,
                },
                {
                    "key": "scheduler",
                    "name": "调度器",
                    "type": "system",
                    "status": "ok" if scheduler_alive else "offline",
                    "message": "运行中" if scheduler_alive else "未运行",
                },
            ],
        },
    })