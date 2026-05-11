from rest_framework import viewsets
from rest_framework.decorators import action
from rest_framework.permissions import IsAdminUser, IsAuthenticated
from rest_framework.response import Response

from .models import Menu
from .serializers import MenuSerializer, MenuTreeSerializer


class MenuViewSet(viewsets.ModelViewSet):
    """菜单 CRUD"""
    queryset = Menu.objects.all()
    serializer_class = MenuSerializer
    permission_classes = [IsAdminUser]
    search_fields = ["name", "code"]
    ordering_fields = ["sort_order", "name"]

    def perform_create(self, serializer):
        parent_id = self.request.data.get("parent")
        if parent_id:
            parent = Menu.objects.get(id=parent_id)
            serializer.save(sort_order=parent.get_children_count() + 1)
        else:
            serializer.save(sort_order=Menu.get_root_nodes().count() + 1)

    @action(detail=False, methods=["get"], permission_classes=[IsAuthenticated])
    def tree(self, request):
        """菜单树 — 用于动态路由加载"""
        user = request.user
        if user.is_superuser:
            menus = Menu.get_root_nodes().filter(is_active=True, is_visible=True)
        else:
            role_ids = user.roles.values_list("id", flat=True)
            menus = Menu.get_root_nodes().filter(
                is_active=True, is_visible=True, roles__id__in=role_ids
            ).distinct()
        ser = MenuTreeSerializer(menus, many=True)
        return Response({"code": 200, "msg": "success", "data": ser.data})

    @action(detail=False, methods=["post"], permission_classes=[IsAdminUser])
    def move(self, request):
        """移动菜单节点"""
        menu_id = request.data.get("id")
        target_id = request.data.get("target_id")
        position = request.data.get("position", "first-child")  # first-child, left, right

        menu = Menu.objects.get(id=menu_id)
        target = Menu.objects.get(id=target_id)
        getattr(menu, position)(target)
        return Response({"code": 200, "msg": "success"})