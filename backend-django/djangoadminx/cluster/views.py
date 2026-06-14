from rest_framework import viewsets
from rest_framework.decorators import action
from rest_framework.permissions import IsAdminUser, AllowAny
from rest_framework.response import Response

from django.utils import timezone

from djangoadminx.audit.mixins import AuditLogMixin
from .models import ClusterNode, ServiceComponent
from .serializers import (
    ClusterNodeSerializer, ServiceComponentSerializer,
    ServiceComponentUpgradeSerializer, HeartbeatSerializer,
)


class ClusterNodeViewSet(AuditLogMixin, viewsets.ModelViewSet):
    """集群节点 CRUD"""
    queryset = ClusterNode.objects.all()
    serializer_class = ClusterNodeSerializer
    permission_classes = [IsAdminUser]
    search_fields = ["name", "host"]
    ordering_fields = ["name", "status", "created_at"]
    filterset_fields = ["status"]

    @action(detail=False, methods=["get"], permission_classes=[IsAdminUser])
    def overview(self, request):
        """集群概览"""
        qs = self.get_queryset()
        total = qs.count()
        online = qs.filter(status="online").count()
        offline = qs.filter(status="offline").count()
        return Response({
            "code": 200,
            "msg": "success",
            "data": {
                "total": total,
                "online": online,
                "offline": offline,
                "nodes": ClusterNodeSerializer(qs, many=True).data,
            },
        })


class ServiceComponentViewSet(AuditLogMixin, viewsets.ModelViewSet):
    """业务组件注册管理"""
    queryset = ServiceComponent.objects.all()
    serializer_class = ServiceComponentSerializer
    permission_classes = [IsAdminUser]
    search_fields = ["name", "app_label"]
    ordering_fields = ["name", "registered_at", "last_heartbeat"]

    @action(detail=True, methods=["post"], permission_classes=[IsAdminUser])
    def set_upgrade(self, request, pk=None):
        """设置升级任务 — 下次心跳时业务服务会收到升级指令"""
        component = self.get_object()
        ser = ServiceComponentUpgradeSerializer(data=request.data)
        ser.is_valid(raise_exception=True)
        component.upgrade_version = ser.validated_data["upgrade_version"]
        component.upgrade_url = ser.validated_data["upgrade_url"]
        component.upgrade_checksum = ser.validated_data.get("upgrade_checksum", "")
        component.save(update_fields=["upgrade_version", "upgrade_url", "upgrade_checksum", "updated_at"])
        return Response({"code": 200, "msg": "升级任务已设置，等待业务服务心跳拉取"})

    @action(detail=True, methods=["post"], permission_classes=[IsAdminUser])
    def cancel_upgrade(self, request, pk=None):
        """取消升级任务"""
        component = self.get_object()
        component.upgrade_version = ""
        component.upgrade_url = ""
        component.upgrade_checksum = ""
        component.save(update_fields=["upgrade_version", "upgrade_url", "upgrade_checksum", "updated_at"])
        return Response({"code": 200, "msg": "升级任务已取消"})

    @action(detail=True, methods=["post"], permission_classes=[IsAdminUser])
    def set_uninstall(self, request, pk=None):
        """设置卸载任务 — 下次心跳时业务服务会收到卸载指令"""
        component = self.get_object()
        # 卸载与升级互斥，设置卸载时清除升级任务
        component.uninstall_pending = True
        component.upgrade_version = ""
        component.upgrade_url = ""
        component.upgrade_checksum = ""
        component.save(update_fields=[
            "uninstall_pending", "upgrade_version", "upgrade_url", "upgrade_checksum", "updated_at"
        ])
        return Response({"code": 200, "msg": "卸载任务已设置，等待业务服务心跳拉取"})

    @action(detail=True, methods=["post"], permission_classes=[IsAdminUser])
    def cancel_uninstall(self, request, pk=None):
        """取消卸载任务"""
        component = self.get_object()
        component.uninstall_pending = False
        component.save(update_fields=["uninstall_pending", "updated_at"])
        return Response({"code": 200, "msg": "卸载任务已取消"})

    @action(detail=False, methods=["post"], permission_classes=[AllowAny],
            url_path="register")
    def register(self, request):
        """业务服务注册（幂等，按 app_label 更新）"""
        ser = HeartbeatSerializer(data=request.data)
        ser.is_valid(raise_exception=True)
        d = ser.validated_data
        component, created = ServiceComponent.objects.update_or_create(
            app_label=d["app_label"],
            defaults={
                "name": request.data.get("name", d["app_label"]),
                "version": d["version"],
                "host": d["host"],
                "description": request.data.get("description", ""),
                "extra_info": d["extra_info"],
                "last_heartbeat": timezone.now(),
            },
        )
        return Response({
            "code": 200,
            "msg": "注册成功" if created else "更新成功",
            "data": ServiceComponentSerializer(component).data,
        })

    @action(detail=False, methods=["post"], permission_classes=[AllowAny],
            url_path="heartbeat")
    def heartbeat(self, request):
        """业务服务心跳上报，响应中携带升级指令（如有）"""
        ser = HeartbeatSerializer(data=request.data)
        ser.is_valid(raise_exception=True)
        d = ser.validated_data
        try:
            component = ServiceComponent.objects.get(app_label=d["app_label"])
        except ServiceComponent.DoesNotExist:
            return Response({"code": 404, "msg": "组件未注册，请先调用 /register/"})

        update_fields = ["last_heartbeat", "updated_at"]
        component.last_heartbeat = timezone.now()
        if d["version"]:
            component.version = d["version"]
            update_fields.append("version")
        if d["host"]:
            component.host = d["host"]
            update_fields.append("host")
        if d["extra_info"]:
            component.extra_info = d["extra_info"]
            update_fields.append("extra_info")
        component.save(update_fields=update_fields)

        # 如果有待升级任务，在响应中下发
        upgrade_cmd = None
        if component.has_pending_upgrade:
            upgrade_cmd = {
                "version": component.upgrade_version,
                "url": component.upgrade_url,
                "checksum": component.upgrade_checksum,
            }

        # 如果有待卸载任务，在响应中下发
        uninstall_cmd = True if component.has_pending_uninstall else None

        return Response({
            "code": 200,
            "msg": "ok",
            "data": {"upgrade": upgrade_cmd, "uninstall": uninstall_cmd},
        })

    @action(detail=False, methods=["post"], permission_classes=[AllowAny],
            url_path="unregister")
    def unregister(self, request):
        """业务服务注销"""
        app_label = request.data.get("app_label", "")
        if not app_label:
            return Response({"code": 400, "msg": "app_label 不能为空"})
        deleted, _ = ServiceComponent.objects.filter(app_label=app_label).delete()
        if deleted:
            return Response({"code": 200, "msg": "注销成功"})
        return Response({"code": 404, "msg": "组件不存在"})

    @action(detail=True, methods=["post"], permission_classes=[IsAdminUser])
    def confirm_upgrade(self, request, pk=None):
        """升级完成后清除升级任务"""
        component = self.get_object()
        component.upgrade_version = ""
        component.upgrade_url = ""
        component.upgrade_checksum = ""
        component.save(update_fields=["upgrade_version", "upgrade_url", "upgrade_checksum", "updated_at"])
        return Response({"code": 200, "msg": "升级任务已清除"})

    @action(detail=True, methods=["post"], permission_classes=[IsAdminUser])
    def confirm_uninstall(self, request, pk=None):
        """卸载完成后清除卸载任务（通常由业务服务自行调用或管理员手动清除）"""
        component = self.get_object()
        component.uninstall_pending = False
        component.save(update_fields=["uninstall_pending", "updated_at"])
        return Response({"code": 200, "msg": "卸载任务已清除"})

