from rest_framework import viewsets, mixins
from rest_framework.decorators import action
from rest_framework.permissions import IsAdminUser, IsAuthenticated
from rest_framework.response import Response

from .models import Config
from .serializers import ConfigSerializer


class ConfigViewSet(mixins.CreateModelMixin,
                    mixins.UpdateModelMixin,
                    mixins.DestroyModelMixin,
                    viewsets.ReadOnlyModelViewSet):
    """配置中心 CRUD + 分组批量查询"""
    queryset = Config.objects.all()
    serializer_class = ConfigSerializer
    permission_classes = [IsAdminUser]
    search_fields = ["key", "desc", "group"]
    ordering_fields = ["group", "key", "created_at"]

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

    @action(detail=False, methods=["get"], permission_classes=[IsAdminUser])
    def groups(self, request):
        """获取所有分组列表"""
        groups = Config.objects.values_list("group", flat=True).distinct().order_by("group")
        return Response({"code": 200, "msg": "success", "data": list(groups)})