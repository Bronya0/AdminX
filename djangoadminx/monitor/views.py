from rest_framework import status
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
def netstat_info(request):
    """网络连接统计"""
    data = SystemMonitor.netstat()
    return Response({"code": 200, "msg": "success", "data": data})