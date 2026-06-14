import fnmatch
import json

from rest_framework.permissions import BasePermission


def _match_path(request, rule: str) -> bool:
    """检查请求是否匹配白名单规则（如 'GET:/api/v1/accounts/users/*'）

    使用 request.path_info 而非 request.path，
    以兼容 FORCE_SCRIPT_NAME 场景（如子路径部署）。
    """
    rl = rule.strip()
    rl_method = ""
    rl_path = rl
    if ":" in rl:
        parts = rl.split(":", 1)
        rl_method = parts[0].upper()
        rl_path = parts[1].strip()
    if rl_method and rl_method != request.method.upper():
        return False
    return fnmatch.fnmatch(request.path_info, rl_path)


class RBACPermission(BasePermission):
    """基于菜单 + 接口路径白名单的 RBAC 权限校验

    - 超级管理员 → 放行所有
    - 普通用户 → 根据角色绑定的菜单（Menu.allowed_paths 和
      BusinessCommand.allowed_paths）匹配当前请求路径
    """

    def has_permission(self, request, view):
        if not request.user or not request.user.is_authenticated:
            return False

        if request.user.is_superuser:
            return True

        role_ids = list(request.user.roles.values_list("id", flat=True))
        if not role_ids:
            return False

        from djangoadminx.menu.models import Menu

        user_menu_paths = set(
            Menu.objects.filter(is_active=True, roles__id__in=role_ids)
            .values_list("path", flat=True)
        )

        # 1. 检查平台菜单的 allowed_paths
        menus = Menu.objects.filter(
            is_active=True, path__in=user_menu_paths
        ).exclude(allowed_paths="[]").exclude(allowed_paths="")
        for menu in menus:
            try:
                paths = json.loads(menu.allowed_paths)
            except (json.JSONDecodeError, TypeError):
                continue
            for rule in paths:
                if _match_path(request, rule):
                    return True

        # 2. 检查三方业务命令的白名单
        from djangoadminx.accounts.models import BusinessCommand

        commands = BusinessCommand.objects.filter(
            is_active=True, menu_path__in=user_menu_paths
        )
        for cmd in commands:
            try:
                paths = json.loads(cmd.allowed_paths) if cmd.allowed_paths else []
            except json.JSONDecodeError:
                continue
            for rule in paths:
                if _match_path(request, rule):
                    return True

        return False

    def has_object_permission(self, request, view, obj):
        if request.user.is_superuser:
            return True
        return super().has_object_permission(request, view, obj)
