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

# ========== 默认角色 ==========
DEFAULT_ROLES = [
    {"name": "超级管理员", "code": "super_admin", "desc": "系统最高权限"},
    {"name": "普通用户", "code": "user", "desc": "普通用户"},
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
            {
                "code": "system:user",
                "name": "用户管理",
                "icon": "User",
                "path": "/system/user",
                "component": "system/user/index",
                "permission_code": "accounts:user:list",
                "menu_type": "menu",
                "sort_order": 1,
            },
            {
                "code": "system:role",
                "name": "角色管理",
                "icon": "Shield",
                "path": "/system/role",
                "component": "system/role/index",
                "permission_code": "accounts:role:list",
                "menu_type": "menu",
                "sort_order": 2,
            },
            {
                "code": "system:menu",
                "name": "菜单管理",
                "icon": "Menu",
                "path": "/system/menu",
                "component": "system/menu/index",
                "permission_code": "menu:list",
                "menu_type": "menu",
                "sort_order": 3,
            },
        ],
    },
    {
        "code": "config",
        "name": "配置中心",
        "icon": "Setting",
        "path": "/config",
        "menu_type": "menu",
        "sort_order": 2,
        "children": [
            {
                "code": "config:list",
                "name": "配置管理",
                "icon": "Operation",
                "path": "/config/list",
                "component": "config/list/index",
                "permission_code": "config_center:config:list",
                "menu_type": "menu",
                "sort_order": 1,
            },
        ],
    },
    {
        "code": "monitor",
        "name": "系统监控",
        "icon": "Monitor",
        "path": "/monitor",
        "menu_type": "menu",
        "sort_order": 3,
        "children": [
            {
                "code": "monitor:resource",
                "name": "系统资源",
                "icon": "Cpu",
                "path": "/monitor/resource",
                "component": "monitor/resource/index",
                "permission_code": "monitor:resource:list",
                "menu_type": "menu",
                "sort_order": 1,
            },
            {
                "code": "monitor:cluster",
                "name": "集群管理",
                "icon": "Cloud",
                "path": "/monitor/cluster",
                "component": "monitor/cluster/index",
                "permission_code": "cluster:node:list",
                "menu_type": "menu",
                "sort_order": 2,
            },
        ],
    },
    {
        "code": "webservice",
        "name": "接口管理",
        "icon": "Link",
        "path": "/webservice",
        "menu_type": "menu",
        "sort_order": 4,
        "children": [
            {
                "code": "webservice:config",
                "name": "WebService 配置",
                "icon": "Api",
                "path": "/webservice/config",
                "component": "webservice/config/index",
                "permission_code": "webservice:config:list",
                "menu_type": "menu",
                "sort_order": 1,
            },
            {
                "code": "webservice:job",
                "name": "定时任务",
                "icon": "Time",
                "path": "/webservice/job",
                "component": "webservice/job/index",
                "permission_code": "webservice:job:list",
                "menu_type": "menu",
                "sort_order": 2,
            },
        ],
    },
]

# ========== 默认配置 ==========
DEFAULT_CONFIGS = [
    {"key": "IP_WHITELIST", "value": "", "value_type": "string", "desc": "IP 白名单（逗号分隔）", "group": "security"},
    {"key": "IP_BLACKLIST", "value": "", "value_type": "string", "desc": "IP 黑名单（逗号分隔）", "group": "security"},
    {"key": "LOGIN_MAX_ATTEMPTS", "value": "5", "value_type": "int", "desc": "登录最大失败次数", "group": "security"},
    {"key": "LOGIN_LOCK_DURATION", "value": "15", "value_type": "int", "desc": "登录锁定时长（分钟）", "group": "security"},
    {"key": "SITE_NAME", "value": "DjangoAdminX", "value_type": "string", "desc": "站点名称", "group": "site"},
    {"key": "SITE_LOGO", "value": "", "value_type": "string", "desc": "站点 Logo URL", "group": "site"},
]


def create_menus(menu_data, parent=None):
    """递归创建菜单 — 使用 treebeard MP_Node API"""
    from djangoadminx.menu.models import Menu

    created = []
    for item in menu_data:
        children = item.pop("children", [])
        code = item.pop("code")
        defaults = {**item, "is_active": True}

        try:
            menu = Menu.objects.get(code=code)
            # 更新字段
            for key, val in defaults.items():
                setattr(menu, key, val)
            menu.save()
        except Menu.DoesNotExist:
            if parent:
                menu = parent.add_child(code=code, **defaults)
            else:
                menu = Menu.add_root(code=code, **defaults)

        created.append(menu)
        if children:
            created.extend(create_menus(children, parent=menu))
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

        # 超级管理员赋予所有权限
        admin_role = role_map.get("super_admin")
        if admin_role:
            all_perms = Permission.objects.all()
            admin_role.permissions.add(*all_perms)
            self.stdout.write(f"  超级管理员已赋予 {all_perms.count()} 个权限")

        # 2. 创建默认菜单
        menus = create_menus(DEFAULT_MENUS)
        self.stdout.write(f"  菜单创建/更新: {len(menus)} 条")

        # 超级管理员角色绑定所有菜单
        if admin_role:
            from djangoadminx.menu.models import Menu
            admin_role.menus.add(*Menu.objects.all())

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