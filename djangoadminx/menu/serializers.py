from rest_framework import serializers

from .models import Menu


class MenuFlatSerializer(serializers.ModelSerializer):
    """扁平菜单序列化器 — 不含 children，用于列表"""
    parent = serializers.SerializerMethodField()

    class Meta:
        model = Menu
        fields = [
            "id", "code", "name", "icon", "path", "component",
            "permission_code", "menu_type", "is_active", "is_visible",
            "sort_order", "depth", "numchild", "parent",
            "created_at", "updated_at",
        ]
        read_only_fields = ["id", "depth", "numchild", "created_at", "updated_at"]

    def get_parent(self, obj):
        try:
            parent = obj.get_parent()
            return str(parent.pk) if parent else None
        except Exception:
            # treebeard path 被路由 path 覆盖时 get_parent 会失败
            return None


class MenuSerializer(serializers.ModelSerializer):
    children = serializers.SerializerMethodField()

    class Meta:
        model = Menu
        fields = [
            "id", "code", "name", "icon", "path", "component",
            "permission_code", "menu_type", "is_active", "is_visible",
            "sort_order", "depth", "path", "numchild", "children",
            "created_at", "updated_at",
        ]
        read_only_fields = ["id", "depth", "numchild", "created_at", "updated_at"]

    def get_children(self, obj):
        children = obj.get_children()
        if children:
            return MenuSerializer(children, many=True).data
        return []


class MenuTreeSerializer(serializers.ModelSerializer):
    """树形菜单 — 用于前端路由加载"""
    children = serializers.SerializerMethodField()

    class Meta:
        model = Menu
        fields = ["id", "code", "name", "icon", "path", "component",
                   "menu_type", "permission_code", "sort_order", "children"]

    def get_children(self, obj):
        children = obj.get_children().filter(is_active=True, is_visible=True)
        if children:
            return MenuTreeSerializer(children, many=True).data
        return []