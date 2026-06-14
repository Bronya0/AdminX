import json

from rest_framework import serializers

from .models import Menu


def _get_children_queryset(obj, active_only=False):
    """用 code 前缀匹配替代 treebeard 的 get_children()。
    
    Menu.path 覆盖了 treebeard 内部物化路径，导致 get_children() 不可用。
    项目使用 code 编码约定（如 system:user 是 system 的子节点）来维护层级。
    """
    qs = Menu.objects.filter(
        code__startswith=obj.code + ":",
        depth=obj.depth + 1,
    ).order_by("sort_order")
    if active_only:
        qs = qs.filter(is_active=True, is_visible=True)
    return qs


class MenuFlatSerializer(serializers.ModelSerializer):
    """扁平菜单序列化器（列表用，不含 children）"""
    parent = serializers.SerializerMethodField()

    class Meta:
        model = Menu
        fields = [
            "id", "code", "name", "icon", "path", "component",
            "permission_code", "menu_type", "is_active", "is_visible",
            "sort_order", "depth", "numchild", "parent",
            "allowed_paths",
            "created_at", "updated_at",
        ]
        read_only_fields = ["id", "depth", "numchild", "created_at", "updated_at"]

    def get_parent(self, obj):
        if obj.depth <= 1 or ":" not in obj.code:
            return None
        try:
            parent = Menu.objects.get(code=obj.code.rsplit(":", 1)[0], depth=obj.depth - 1)
            return str(parent.pk)
        except Menu.DoesNotExist:
            return None


class MenuSerializer(serializers.ModelSerializer):
    children = serializers.SerializerMethodField()

    class Meta:
        model = Menu
        fields = [
            "id", "code", "name", "icon", "path", "component",
            "permission_code", "menu_type", "is_active", "is_visible",
            "sort_order", "depth", "numchild", "children",
            "allowed_paths",
            "created_at", "updated_at",
        ]
        read_only_fields = ["id", "depth", "numchild", "created_at", "updated_at"]

    def get_children(self, obj):
        children = _get_children_queryset(obj)
        if children:
            return MenuSerializer(children, many=True).data
        return []


    def validate_allowed_paths(self, value):
        if not value:
            return '[]'
        try:
            items = json.loads(value) if isinstance(value, str) else value
            if not isinstance(items, list):
                return value
            cleaned = [s.strip() for s in items if isinstance(s, str) and s.strip()]
            return json.dumps(cleaned)
        except (json.JSONDecodeError, TypeError):
            return value

class MenuTreeSerializer(serializers.ModelSerializer):
    """树形菜单 — 用于前端路由加载"""
    children = serializers.SerializerMethodField()

    class Meta:
        model = Menu
        fields = ["id", "code", "name", "icon", "path", "component",
                   "menu_type", "permission_code", "sort_order", "children"]

    def get_children(self, obj):
        children = _get_children_queryset(obj, active_only=True)
        if children:
            return MenuTreeSerializer(children, many=True).data
        return []