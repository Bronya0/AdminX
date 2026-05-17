from rest_framework import serializers, viewsets
from rest_framework.decorators import action
from rest_framework.permissions import IsAdminUser, IsAuthenticated
from rest_framework.response import Response

from djangoadminx.audit.mixins import AuditLogMixin
from .models import Menu
from .serializers import MenuSerializer, MenuTreeSerializer

ALLOWED_POSITIONS = {"first-child", "last-child", "left", "right"}


class MenuViewSet(AuditLogMixin, viewsets.ModelViewSet):
    """菜单 CRUD"""
    queryset = Menu.objects.all()
    serializer_class = MenuSerializer
    permission_classes = [IsAdminUser]
    search_fields = ["name", "code"]
    ordering_fields = ["sort_order", "name"]
    pagination_class = None

    def get_serializer_class(self):
        if self.action == 'list':
            from .serializers import MenuFlatSerializer
            return MenuFlatSerializer
        return MenuSerializer

    def perform_create(self, serializer):
        parent_id = self.request.data.get("parent")
        depth = 1
        if parent_id:
            try:
                parent = Menu.objects.get(id=parent_id)
                depth = parent.depth + 1
                instance = serializer.save(depth=depth, sort_order=parent.get_children_count() + 1)
            except Menu.DoesNotExist:
                raise serializers.ValidationError({"parent": "上级菜单不存在"})
        else:
            instance = serializer.save(depth=depth, sort_order=Menu.get_root_nodes().count() + 1)
        self._log_audit_create(instance)

    @action(detail=False, methods=["get"], permission_classes=[IsAdminUser])
    def tree(self, request):
        """菜单树 — 用于菜单管理页面"""
        if hasattr(Menu, 'get_root_nodes'):
            menus = Menu.get_root_nodes()
        else:
            menus = Menu.objects.filter(parent__isnull=True)
        ser = MenuTreeSerializer(menus, many=True)
        return Response({"code": 200, "msg": "success", "data": ser.data})

    @action(detail=False, methods=["get"], permission_classes=[IsAuthenticated])
    def user_tree(self, request):
        """用户菜单树 — 用于前端动态路由加载"""
        user = request.user
        if user.is_superuser:
            if hasattr(Menu, 'get_root_nodes'):
                menus = Menu.get_root_nodes().filter(is_active=True, is_visible=True)
            else:
                menus = Menu.objects.filter(parent__isnull=True, is_active=True, is_visible=True)
        else:
            role_ids = user.roles.values_list("id", flat=True)
            menus = Menu.objects.filter(
                parent__isnull=True, is_active=True, is_visible=True, roles__id__in=role_ids
            ).distinct()
        ser = MenuTreeSerializer(menus, many=True)
        return Response({"code": 200, "msg": "success", "data": ser.data})

    @action(detail=False, methods=["post"], permission_classes=[IsAdminUser])
    def move(self, request):
        """移动菜单节点（treebeard 操作因 path 字段冲突不可用，使用手动更新 parent/sort_order）"""
        menu_id = request.data.get("id")
        target_id = request.data.get("target_id")
        position = request.data.get("position", "first-child")

        if not menu_id or not target_id:
            return Response({"code": 400, "msg": "请指定 id 和 target_id"})
        if menu_id == target_id:
            return Response({"code": 400, "msg": "不能将节点移动到自己"})
        if position not in ALLOWED_POSITIONS:
            return Response({"code": 400, "msg": f"无效的 position: {position}，允许: {', '.join(ALLOWED_POSITIONS)}"})

        try:
            menu = Menu.objects.get(id=menu_id)
            target = Menu.objects.get(id=target_id)
        except Menu.DoesNotExist:
            return Response({"code": 400, "msg": "菜单节点不存在"})

        try:
            getattr(menu, position)(target)
        except Exception as e:
            return Response({"code": 400, "msg": f"移动失败: {e}"})
        return Response({"code": 200, "msg": "success"})
