from rest_framework import viewsets, mixins
from rest_framework.decorators import action
from rest_framework.exceptions import ValidationError
from rest_framework.permissions import IsAdminUser, IsAuthenticated
from rest_framework.response import Response

from djangoadminx.audit.mixins import AuditLogMixin
from .models import Config
from .serializers import ConfigSerializer


class ConfigViewSet(AuditLogMixin,
                    mixins.CreateModelMixin,
                    mixins.UpdateModelMixin,
                    mixins.DestroyModelMixin,
                    viewsets.ReadOnlyModelViewSet):
    """配置中心 CRUD + 分组批量查询"""
    queryset = Config.objects.order_by("-created_at")
    serializer_class = ConfigSerializer
    permission_classes = [IsAdminUser]
    search_fields = ["key", "desc", "group"]
    ordering_fields = ["group", "key", "created_at"]
    filterset_fields = ["group", "value_type", "is_active", "is_encrypted"]

    def perform_update(self, serializer):
        if serializer.instance.is_encrypted:
            if "value" not in self.request.data:
                serializer.validated_data.pop("value", None)
            if self.request.data.get("is_encrypted") is False:
                raise ValidationError("加密存储的配置不允许改回非加密")
        super().perform_update(serializer)

    def filter_queryset(self, queryset):
        queryset = super().filter_queryset(queryset)
        desc = self.request.query_params.get("desc")
        if desc:
            queryset = queryset.filter(desc__icontains=desc)
        return queryset

    @action(detail=False, methods=["post"], permission_classes=[IsAuthenticated])
    def ntp_sync(self, request):
        """NTP 时间同步 — 用于配置中心的主题设置页"""
        from djangoadminx.common.ntp import sync_time
        from djangoadminx.config_center.models import Config
        server = Config.get_value("NTP_SERVER", default="")
        enabled = Config.get_value("NTP_SYNC_ENABLED", default=False)
        if not enabled or not server:
            return Response({"code": 200, "msg": "success", "data": {"enabled": False, "server": server or "(未配置)"}})
        result = sync_time(server)
        return Response({"code": 200, "msg": "success", "data": result})

    @action(detail=False, methods=["get"], permission_classes=[IsAuthenticated])
    def by_group(self, request):
        """
        按分组批量查询配置

        用于前端一次性拉取某个业务场景的所有配置/选项列表。
        例如: /api/v1/config/by_group/?group=post_status
        → 返回 {code, msg, data: {status_draft: {...}, status_published: {...}}}
        其中 OPTIONS 类型的 value 自动解析为 [{label, value}, ...] 格式。
        """
        group = request.query_params.get("group", "")
        if not group:
            return Response({"code": 400, "msg": "请指定 group 参数"})

        data = Config.get_by_group(group)
        return Response({"code": 200, "msg": "success", "data": data})

    @action(detail=False, methods=["get"], permission_classes=[IsAuthenticated])
    def get_value(self, request):
        """按 key 获取单个配置值（业务容器友好）

        业务容器通过用户 JWT 调用此接口，获取配置中心的值。
        例如: GET /api/v1/config/get_value/?key=SOME_KEY
        → 返回 {code: 200, data: {key: "SOME_KEY", value: "xxx"}}
        """
        key = request.query_params.get("key", "")
        if not key:
            return Response({"code": 400, "msg": "请指定 key 参数"})

        value = Config.get_value(key, default=None)
        if value is None:
            return Response({"code": 404, "msg": f"配置 {key} 不存在"})

        return Response({
            "code": 200,
            "msg": "success",
            "data": {"key": key, "value": value},
        })

    @action(detail=False, methods=["get"], permission_classes=[IsAdminUser])
    def groups(self, request):
        """获取所有分组列表"""
        groups = Config.objects.values_list("group", flat=True).distinct().order_by("group")
        return Response({"code": 200, "msg": "success", "data": list(groups)})