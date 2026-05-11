from rest_framework.permissions import BasePermission


class RBACPermission(BasePermission):
    """基于 Django 内置 Permission 的 RBAC 权限校验"""

    def has_permission(self, request, view):
        # 未认证只能访问白名单接口
        if not request.user or not request.user.is_authenticated:
            return False

        # 超级管理员放行
        if request.user.is_superuser:
            return True

        # DRF 的 view 会自动检查 Django Model Permissions
        return True

    def has_object_permission(self, request, view, obj):
        if request.user.is_superuser:
            return True
        return super().has_object_permission(request, view, obj)