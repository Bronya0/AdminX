"""测试数据工厂 — 创建用户/角色/配置等"""

from django.contrib.auth.models import Permission

from djangoadminx.accounts.models import User, Role


class UserFactory:
    @staticmethod
    def create_admin(username="admin", password="admin123"):
        user = User.objects.create(
            username=username, is_superuser=True, is_staff=True, is_active=True,
        )
        user.set_password(password)
        user.save(update_fields=["password"])
        return user

    @staticmethod
    def create_user(username="user", password="test123"):
        user = User.objects.create(username=username, is_active=True)
        user.set_password(password)
        user.save(update_fields=["password"])
        return user

    @staticmethod
    def create_with_permission(perm_codename, username="perm_user"):
        role = Role.objects.create(name=f"role_{perm_codename}")
        try:
            perm = Permission.objects.get(codename=perm_codename)
            role.permissions.add(perm)
        except Permission.DoesNotExist:
            pass
        user = User.objects.create(username=username, is_active=True)
        user.set_password("test123")
        user.save(update_fields=["password"])
        user.roles.add(role)
        return user


class ConfigFactory:
    @staticmethod
    def create(key, value="", value_type="string", group="test", **kw):
        from djangoadminx.config_center.models import Config
        return Config.objects.create(
            key=key, value=value, value_type=value_type, group=group, **kw
        )
