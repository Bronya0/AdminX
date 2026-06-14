from rest_framework.decorators import api_view, permission_classes
from rest_framework.permissions import IsAdminUser
from rest_framework.response import Response

from .utils import SystemMonitor


@api_view(["GET"])
@permission_classes([IsAdminUser])
def system_resource(request):
    """系统资源实时信息"""
    data = SystemMonitor.all()
    return Response({"code": 200, "msg": "success", "data": data})


@api_view(["GET"])
@permission_classes([IsAdminUser])
def system_resource_history(request):
    """系统资源历史趋势"""
    range_key = request.query_params.get("range", "1h")
    interval_key = request.query_params.get("interval", "auto")

    try:
        data = SystemMonitor.history(range_key=range_key, interval_key=interval_key)
    except ValueError as exc:
        return Response({"code": 400, "msg": str(exc), "data": None})

    return Response({"code": 200, "msg": "success", "data": data})


@api_view(["GET"])
@permission_classes([IsAdminUser])
def netstat_info(request):
    """网络连接统计"""
    data = SystemMonitor.netstat()
    return Response({"code": 200, "msg": "success", "data": data})
