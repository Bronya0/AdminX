import os
import json
import logging

from django.conf import settings
from django.http import JsonResponse, StreamingHttpResponse
from django.views.decorators.http import require_GET
from rest_framework.decorators import api_view, permission_classes
from rest_framework.permissions import AllowAny


@api_view(["GET"])
@permission_classes([AllowAny])
def site_info(request):
    """站点信息（公开）— 供前端登录页/顶栏/侧栏使用"""
    from djangoadminx.config_center.models import Config
    return JsonResponse({
        "code": 200,
        "msg": "success",
        "data": {
            "site_name": Config.get_value("SITE_NAME", default="DjangoAdminX"),
            "site_desc": Config.get_value("SITE_DESC", default="企业级 Django Admin 框架"),
            "site_logo": Config.get_value("SITE_LOGO", default=""),
            "site_theme_color": Config.get_value("SITE_THEME_COLOR", default="#1890ff"),
            "idle_timeout": Config.get_value("SESSION_IDLE_TIMEOUT", default=30),
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
    """SSE 实时日志推送"""
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