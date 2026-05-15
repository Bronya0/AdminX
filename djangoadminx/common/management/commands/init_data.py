"""
初始化系统基础数据: 角色、菜单、配置、超级用户

用法:
  python manage.py init_data                       # 初始化基础数据
  python manage.py init_data --superuser           # 同时创建超级用户
  python manage.py init_data --superuser --username admin --password admin123
"""
import json
import logging

from django.conf import settings
from django.contrib.auth import get_user_model
from django.contrib.auth.models import Permission
from django.contrib.contenttypes.models import ContentType
from django.core.management.base import BaseCommand

logger = logging.getLogger("djangoadminx.init_data")

UserModel = get_user_model()

# ========== 默认角色（等保2.0 三权分立）==========
DEFAULT_ROLES = [
    {"name": "超级管理员", "code": "super_admin", "desc": "系统最高权限，拥有所有操作权限"},
    {"name": "安全管理员", "code": "security_admin", "desc": "负责安全策略配置、系统监控与安全审计"},
    {"name": "审计管理员", "code": "audit_admin", "desc": "负责审计日志查看与操作追溯"},
    {"name": "普通用户", "code": "user", "desc": "普通用户，仅有基本查看权限"},
]

# ========== 默认菜单树 ==========
DEFAULT_MENUS = [
    {
        "code": "system",
        "name": "系统管理",
        "icon": "Settings",
        "path": "/system",
        "menu_type": "menu",
        "sort_order": 1,
        "children": [
            {"code": "system:user", "name": "用户管理", "icon": "User", "path": "/system/user", "component": "system/user/index", "permission_code": "accounts:user:list", "menu_type": "menu", "sort_order": 1},
            {"code": "system:role", "name": "角色管理", "icon": "Shield", "path": "/system/role", "component": "system/role/index", "permission_code": "accounts:role:list", "menu_type": "menu", "sort_order": 2},
            {"code": "system:permission", "name": "权限管理", "icon": "Safety", "path": "/system/permissions", "component": "system/permissions/index", "menu_type": "menu", "sort_order": 3},
            {"code": "system:config", "name": "配置中心", "icon": "Setting", "path": "/system/config", "component": "config/list/index", "permission_code": "config_center:config:list", "menu_type": "menu", "sort_order": 4},
            {"code": "system:resource", "name": "系统资源", "icon": "Cpu", "path": "/system/resource", "component": "monitor/resource/index", "permission_code": "monitor:resource:list", "menu_type": "menu", "sort_order": 5},
            {"code": "system:component", "name": "组件管理", "icon": "Appstore", "path": "/system/component", "component": "monitor/component/index", "permission_code": "monitor:resource:list", "menu_type": "menu", "sort_order": 6},
            {"code": "system:cluster", "name": "集群管理", "icon": "Cloud", "path": "/system/cluster", "menu_type": "menu", "sort_order": 7, "children": [
                {"code": "system:cluster:nodes", "name": "节点管理", "icon": "Hdd", "path": "/system/cluster/nodes", "component": "cluster/nodes/index", "permission_code": "cluster:node:list", "menu_type": "menu", "sort_order": 1},
            ]},
            {"code": "system:scheduler", "name": "定时任务", "icon": "ClockCircle", "path": "/system/scheduler", "component": "scheduler/index", "permission_code": "webservice:schedulejob:list", "menu_type": "menu", "sort_order": 8},
            {"code": "system:notification", "name": "通知中心", "icon": "Bell", "path": "/system/notification", "component": "notification/index", "permission_code": "notification:notification:list", "menu_type": "menu", "sort_order": 9},
        ],
    },
    {
        "code": "audit",
        "name": "安全审计",
        "icon": "Safety",
        "path": "/audit",
        "menu_type": "menu",
        "sort_order": 2,
        "children": [
            {"code": "audit:log", "name": "操作审计", "icon": "FileSearch", "path": "/audit/log", "component": "audit/log/index", "permission_code": "audit:auditlog:list", "menu_type": "menu", "sort_order": 1},
            {"code": "audit:login-log", "name": "登录日志", "icon": "Login", "path": "/audit/login-log", "component": "audit/login-log/index", "permission_code": "accounts:userloginlog:list", "menu_type": "menu", "sort_order": 2},
        ],
    },
]

# ========== 默认配置 ==========
DEFAULT_CONFIGS = [
    {"key": "IP_WHITELIST", "value": "", "value_type": "string", "desc": "IP 白名单（逗号分隔）", "group": "security"},
    {"key": "IP_BLACKLIST", "value": "", "value_type": "string", "desc": "IP 黑名单（逗号分隔）", "group": "security"},
    {"key": "LOGIN_MAX_ATTEMPTS", "value": "5", "value_type": "int", "desc": "登录锁定阈值：连续失败达此次数后，账号将被临时锁定", "group": "security"},
    {"key": "LOGIN_LOCK_DURATION", "value": "15", "value_type": "int", "desc": "登录锁定持续时间（分钟）：达到失败次数后，锁定多长时间自动解封", "group": "security"},
    {"key": "SITE_NAME", "value": "DjangoAdminX", "value_type": "string", "desc": "站点名称", "group": "site"},
    {"key": "SITE_DESC", "value": "企业级 Django Admin 框架", "value_type": "string", "desc": "站点描述", "group": "site"},
    {"key": "SITE_LOGO", "value": "", "value_type": "string", "desc": "站点 Logo URL", "group": "site"},
    {"key": "SITE_THEME_COLOR", "value": "#1890ff", "value_type": "string", "desc": "站点主题色", "group": "site"},
    {"key": "SESSION_IDLE_TIMEOUT", "value": "30", "value_type": "int", "desc": "登录后空闲超时（分钟）：用户无操作达此时长后自动登出，0=不超时", "group": "security"},
    {"key": "APP_VERSION", "value": "1.0.0", "value_type": "string", "desc": "平台版本号", "group": "system"},
    {"key": "NTP_SYNC_ENABLED", "value": "false", "value_type": "bool", "desc": "是否启用 NTP 时间同步", "group": "system"},
    {"key": "NTP_SERVER", "value": "", "value_type": "string", "desc": "NTP 服务器地址（如 ntp.aliyun.com）", "group": "system"},
    {"key": "NTP_SYNC_INTERVAL", "value": "60", "value_type": "int", "desc": "NTP 同步间隔（分钟）", "group": "system"},
]


def _build_menu_tree(menu_data, depth=1, code_prefix=""):
    """递归构建扁平菜单列表，标记层级关系"""
    items = []
    for item in menu_data:
        children = item.pop("children", [])
        code = item["code"]
        items.append({
            "code": code,
            "name": item.get("name", ""),
            "icon": item.get("icon", ""),
            "path": item.get("path", ""),
            "component": item.get("component", ""),
            "permission_code": item.get("permission_code", ""),
            "menu_type": item.get("menu_type", "menu"),
            "sort_order": item.get("sort_order", 0),
            "depth": depth,
        })
        items.extend(_build_menu_tree(children, depth=depth + 1, code_prefix=code))
    return items


def create_menus(menu_data):
    """创建/更新菜单（绕过 treebeard API，直接 ORM）

    Menu.path 与 treebeard 内部 path 冲突，treebeard 的 add_root/add_child
    不可用。直接使用 ORM 创建并设置 depth/numchild。
    """
    from djangoadminx.menu.models import Menu

    flat = _build_menu_tree(menu_data)
    created = []
    for item in flat:
        code = item.pop("code")
        defaults = {**item, "is_active": True}

        menu, _ = Menu.objects.update_or_create(
            code=code,
            defaults=defaults,
        )
        created.append(menu)

    # 更新每个菜单的 numchild（子节点数）
    codes = {m.code for m in created}
    for menu in created:
        child_count = Menu.objects.filter(
            code__startswith=menu.code + ":"
        ).count()
        if child_count != menu.numchild:
            Menu.objects.filter(code=menu.code).update(numchild=child_count)

    return created


class Command(BaseCommand):
    help = "初始化系统基础数据"

    def add_arguments(self, parser):
        parser.add_argument("--superuser", action="store_true", help="创建超级用户")
        parser.add_argument("--username", type=str, default="admin", help="超级用户名")
        parser.add_argument("--password", type=str, default="admin123", help="超级用户密码")

    def handle(self, *args, **options):
        self.stdout.write(self.style.SUCCESS("[init_data] 开始初始化基础数据..."))

        # 1. 创建默认角色
        from djangoadminx.accounts.models import Role

        role_map = {}
        for r in DEFAULT_ROLES:
            role, created = Role.objects.get_or_create(
                code=r["code"],
                defaults={"name": r["name"], "desc": r["desc"]},
            )
            role_map[r["code"]] = role
            self.stdout.write(f"  角色 {'创建' if created else '已存在'}: {role.name}")

        # 超级管理员 — 全部权限 + 全部菜单
        admin_role = role_map.get("super_admin")
        if admin_role:
            all_perms = Permission.objects.all()
            admin_role.permissions.add(*all_perms)
            self.stdout.write(f"  超级管理员已赋予 {all_perms.count()} 个权限")

        # 安全管理员 — 安全配置/监控/审计查看权限 + 对应菜单
        security_role = role_map.get("security_admin")
        if security_role:
            security_perms = Permission.objects.filter(
                codename__in=[
                    "view_config", "change_config",
                    "view_clusternode",
                    "view_userloginlog",
                    "view_auditlog",
                    "view_loginlock",
                    "view_menu",
                ]
            )
            security_role.permissions.add(*security_perms)
            self.stdout.write(f"  安全管理员已赋予 {security_perms.count()} 个权限")

        # 审计管理员 — 审计日志/登录日志查看权限 + 对应菜单
        audit_role = role_map.get("audit_admin")
        if audit_role:
            audit_perms = Permission.objects.filter(
                codename__in=[
                    "view_auditlog",
                    "view_userloginlog",
                    "view_loginlock",
                ]
            )
            audit_role.permissions.add(*audit_perms)
            self.stdout.write(f"  审计管理员已赋予 {audit_perms.count()} 个权限")

        # 2. 创建默认菜单
        menus = create_menus(DEFAULT_MENUS)
        self.stdout.write(f"  菜单创建/更新: {len(menus)} 条")

        # 超级管理员绑定所有菜单
        if admin_role:
            from djangoadminx.menu.models import Menu
            admin_role.menus.add(*Menu.objects.all())

        # 安全管理员菜单：配置中心 + 系统监控(资源/集群) + 安全审计
        if security_role:
            from djangoadminx.menu.models import Menu
            security_menus = Menu.objects.filter(
                code__in=[
                    "system:permission", "system:config", "system:resource", "system:cluster",
                    "system:cluster:nodes",
                    "audit", "audit:log", "audit:login-log",
                ]
            )
            security_role.menus.add(*security_menus)
            self.stdout.write(f"  安全管理员已绑定 {security_menus.count()} 个菜单")

        # 审计管理员菜单：安全审计
        if audit_role:
            from djangoadminx.menu.models import Menu
            audit_menus = Menu.objects.filter(
                code__in=["audit", "audit:log", "audit:login-log"]
            )
            audit_role.menus.add(*audit_menus)
            self.stdout.write(f"  审计管理员已绑定 {audit_menus.count()} 个菜单")

        # 3. 创建默认配置
        from djangoadminx.config_center.models import Config

        for c in DEFAULT_CONFIGS:
            _, created = Config.objects.get_or_create(
                key=c["key"],
                defaults=c,
            )
            if created:
                self.stdout.write(f"  配置创建: {c['key']}")

        # 4. 可选: 创建超级用户
        if options["superuser"]:
            username = options["username"]
            password = options["password"]
            if not UserModel.all_objects.filter(username=username).exists():
                user = UserModel.all_objects.create(
                    username=username,
                    is_superuser=True,
                    is_staff=True,
                    is_active=True,
                )
                user.set_password(password)
                user.save(update_fields=["password"])
                if admin_role:
                    user.roles.add(admin_role)
                self.stdout.write(self.style.SUCCESS(f"  超级用户创建: {username} / {password}"))
            else:
                self.stdout.write(f"  超级用户已存在: {username}")

        self.stdout.write(self.style.SUCCESS("[init_data] 初始化完成!"))