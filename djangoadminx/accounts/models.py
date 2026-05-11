import uuid

from django.contrib.auth.models import AbstractUser, UserManager as BaseUserManager
from django.db import models
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