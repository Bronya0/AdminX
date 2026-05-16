import uuid
from datetime import timedelta

from django.contrib.auth.models import AbstractUser, UserManager as BaseUserManager
from django.db import models
from django.utils import timezone
from safedelete.managers import SafeDeleteManager
from safedelete.models import SafeDeleteModel, SOFT_DELETE


class UserManager(SafeDeleteManager, BaseUserManager):
    """合并 safedelete + auth UserManager"""

    def get_by_natural_key(self, username):
        return self.get(**{self.model.USERNAME_FIELD: username})

    def create_superuser(self, username, password=None, **extra_fields):
        return self.all_objects.create_superuser(username, password, **extra_fields)


class User(SafeDeleteModel, AbstractUser):
    """用户模型 — 支持软删除"""

    _safedelete_policy = SOFT_DELETE

    objects = UserManager()
    all_objects = SafeDeleteManager()

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    phone = models.CharField("手机号", max_length=20, blank=True, default="")
    avatar = models.URLField("头像", blank=True, default="")
    desc = models.TextField("描述", blank=True, default="")
    last_activity = models.DateTimeField("最后活动时间", null=True, blank=True)
    roles = models.ManyToManyField(
        "Role",
        verbose_name="角色",
        blank=True,
        related_name="users",
    )

    class Meta:
        verbose_name = "用户"
        verbose_name_plural = "用户"
        ordering = ["-date_joined"]

    def __str__(self):
        return self.username or self.email

    @property
    def is_online(self):
        """在线判定：最后活动时间在 5 分钟以内"""
        if not self.last_activity:
            return False
        return self.last_activity >= timezone.now() - timedelta(minutes=5)


class Role(models.Model):
    """角色模型 — 基于 Django Group 扩展"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    name = models.CharField("角色名称", max_length=128, unique=True)
    code = models.CharField("角色编码", max_length=128, unique=True)
    desc = models.TextField("描述", blank=True, default="")
    is_active = models.BooleanField("启用", default=True)
    permissions = models.ManyToManyField(
        "auth.Permission",
        verbose_name="权限",
        blank=True,
        related_name="roles",
    )
    business_permissions = models.ManyToManyField(
        "BusinessPermission",
        verbose_name="业务权限",
        blank=True,
        related_name="roles",
    )
    menus = models.ManyToManyField(
        "menu.Menu",
        verbose_name="关联菜单",
        blank=True,
        related_name="roles",
    )
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "角色"
        verbose_name_plural = "角色"
        ordering = ["name"]

    def __str__(self):
        return self.name


class UserLoginLog(models.Model):
    """登录日志"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    user = models.ForeignKey(User, on_delete=models.SET_NULL, null=True, verbose_name="用户")
    username = models.CharField("用户名", max_length=128)
    ip = models.GenericIPAddressField("IP", blank=True, null=True)
    user_agent = models.TextField("UA", blank=True, default="")
    success = models.BooleanField("是否成功", default=True)
    message = models.CharField("消息", max_length=255, blank=True, default="")
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        verbose_name = "登录日志"
        verbose_name_plural = "登录日志"
        ordering = ["-created_at"]


class LoginLock(models.Model):
    """登录锁定记录"""

    username = models.CharField("用户名", max_length=128, unique=True)
    failed_count = models.IntegerField("失败次数", default=0)
    locked_at = models.DateTimeField("锁定时间", null=True, blank=True)
    unlocked_at = models.DateTimeField("解锁时间", null=True, blank=True)

    class Meta:
        verbose_name = "登录锁定"
        verbose_name_plural = "登录锁定"

    def __str__(self):
        return f"{self.username} locked"


class BusinessPermission(models.Model):
    """业务权限 — 业务容器注册的权限，不依赖 Django ContentType"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    app_label = models.CharField("应用标识", max_length=128, db_index=True,
                                 help_text="如 blog、crm，用于前端分组")
    codename = models.CharField("权限编码", max_length=255, unique=True,
                                help_text="如 blog:post:list，业务容器通过 introspect 获取")
    name = models.CharField("权限名称", max_length=255)
    desc = models.TextField("描述", blank=True, default="")
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        verbose_name = "业务权限"
        verbose_name_plural = "业务权限"
        ordering = ["app_label", "codename"]

    def __str__(self):
        return f"[{self.app_label}] {self.name}"


class BusinessCommand(models.Model):
    """业务命令 — 业务容器注册的菜单 + REST 路径白名单"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    app_label = models.CharField("应用标识", max_length=128, db_index=True,
                                 help_text="如 blog、crm")
    name = models.CharField("名称", max_length=255, help_text="菜单显示名称")
    icon = models.CharField("图标", max_length=128, blank=True, default="",
                            help_text="Ant Design 图标名，如 FileTextOutlined")
    menu_path = models.CharField("前端路由", max_length=255,
                                 help_text="如 /plugins/blog，侧边栏点击后跳转至此")
    allowed_paths = models.TextField("路径白名单", blank=True, default="[]",
                                     help_text='JSON 数组，如 ["GET:/api/v1/blog/*","POST:/api/v1/blog/*"]')
    description = models.TextField("描述", blank=True, default="")
    is_active = models.BooleanField("启用", default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "业务命令"
        verbose_name_plural = "业务命令"
        ordering = ["app_label", "name"]
        unique_together = [["app_label", "menu_path"]]

    def __str__(self):
        return f"[{self.app_label}] {self.name}"