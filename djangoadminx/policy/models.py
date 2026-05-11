import uuid

from django.conf import settings
from django.contrib.auth import get_user_model
from django.contrib.auth.hashers import check_password, make_password
from django.db import models

UserModel = get_user_model()


class PasswordPolicy(models.Model):
    """密码策略配置 — 单例模式"""

    min_length = models.IntegerField("最小长度", default=8)
    require_upper = models.BooleanField("需要大写字母", default=True)
    require_lower = models.BooleanField("需要小写字母", default=True)
    require_digit = models.BooleanField("需要数字", default=True)
    require_special = models.BooleanField("需要特殊字符", default=True)
    expire_days = models.IntegerField("密码过期天数", default=90, help_text="0=不过期")
    history_count = models.IntegerField("历史密码保留数", default=5, help_text="禁止复用最近N次")
    is_active = models.BooleanField("启用", default=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "密码策略"
        verbose_name_plural = "密码策略"

    def save(self, *args, **kwargs):
        self.pk = 1  # 单例
        super().save(*args, **kwargs)

    @classmethod
    def get_instance(cls):
        if not getattr(settings, "PASSWORD_POLICY_ENABLED", False):
            return None
        obj, _ = cls.objects.get_or_create(pk=1)
        return obj if obj.is_active else None

    def validate(self, password, user=None):
        """校验密码是否符合策略，返回 (is_valid, errors)"""
        errors = []
        if len(password) < self.min_length:
            errors.append(f"密码长度不能少于{self.min_length}位")
        if self.require_upper and not any(c.isupper() for c in password):
            errors.append("密码需要包含大写字母")
        if self.require_lower and not any(c.islower() for c in password):
            errors.append("密码需要包含小写字母")
        if self.require_digit and not any(c.isdigit() for c in password):
            errors.append("密码需要包含数字")
        if self.require_special and not any(c in "!@#$%^&*()_+-=[]{}|;':,./<>?`~" for c in password):
            errors.append("密码需要包含特殊字符")
        if user and self.history_count > 0:
            history = PasswordHistory.objects.filter(user=user).order_by("-created_at")[:self.history_count]
            for h in history:
                if check_password(password, h.password_hash):
                    errors.append(f"不能与最近{self.history_count}次密码相同")
                    break
        return len(errors) == 0, errors

    def is_expired(self, user):
        """检查用户密码是否过期"""
        if self.expire_days <= 0:
            return False
        if not user.password or not user.password.startswith(("pbkdf2_", "bcrypt", "argon2")):
            return False
        # 取密码修改时间（Django 不记录，用 last_login 或 date_joined 近似）
        from django.utils import timezone
        from datetime import timedelta
        ref_date = user.last_login or user.date_joined
        if ref_date is None:
            ref_date = timezone.now()
        return timezone.now() - ref_date > timedelta(days=self.expire_days)


class PasswordHistory(models.Model):
    """密码历史"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    user = models.ForeignKey(UserModel, on_delete=models.CASCADE, related_name="password_history")
    password_hash = models.CharField("密码哈希", max_length=256)
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        verbose_name = "密码历史"
        verbose_name_plural = "密码历史"
        ordering = ["-created_at"]

    @classmethod
    def record(cls, user, raw_password):
        """记录新密码"""
        cls.objects.create(user=user, password_hash=make_password(raw_password))