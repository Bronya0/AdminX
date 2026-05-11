import json

from cryptography.fernet import Fernet
from django.conf import settings
from django.core.cache import cache
from django.db import models
from django.db.models.signals import post_save, post_delete
from django.dispatch import receiver


def get_fernet():
    """获取 Fernet 实例 — 密钥来自 settings"""
    key = getattr(settings, "FERNET_KEY", None)
    if not key:
        # 自动生成并存储（仅首次）
        key = Fernet.generate_key()
        settings.FERNET_KEY = key
    return Fernet(key)


class EncryptedConfigField(models.CharField):
    """加密配置字段 — 透明加解密"""

    def __init__(self, *args, **kwargs):
        kwargs["max_length"] = 1024
        kwargs["blank"] = True
        kwargs["default"] = ""
        super().__init__(*args, **kwargs)

    def from_db_value(self, value, expression, connection):
        if value:
            try:
                return get_fernet().decrypt(value.encode()).decode()
            except Exception:
                return value
        return value

    def get_prep_value(self, value):
        if value:
            return get_fernet().encrypt(value.encode()).decode()
        return value


class Config(models.Model):
    """统一配置模型 — 系统参数 + 业务选项列表一体化"""

    class TypeChoices(models.TextChoices):
        STRING = "string", "字符串"
        INT = "int", "整数"
        BOOL = "bool", "布尔"
        JSON = "json", "JSON"
        ENCRYPTED = "encrypted", "加密"
        OPTIONS = "options", "选项列表"

    key = models.CharField("配置键", max_length=255, unique=True, db_index=True)
    value = models.TextField("配置值", blank=True, default="")
    value_type = models.CharField(
        "值类型", max_length=20, choices=TypeChoices.choices, default=TypeChoices.STRING
    )
    encrypted_value = EncryptedConfigField("加密值")
    desc = models.CharField("描述", max_length=500, blank=True, default="")
    group = models.CharField("分组", max_length=128, blank=True, default="default")
    is_active = models.BooleanField("启用", default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "配置"
        verbose_name_plural = "配置"
        ordering = ["group", "key"]

    def __str__(self):
        return f"{self.key}={self.parse_value()}"

    def parse_value(self):
        """解析 value 为对应类型"""
        if self.value_type == self.TypeChoices.ENCRYPTED:
            return self.encrypted_value
        if self.value_type == self.TypeChoices.INT:
            return int(self.value)
        if self.value_type == self.TypeChoices.BOOL:
            return self.value.lower() in ("true", "1", "yes")
        if self.value_type in (self.TypeChoices.JSON, self.TypeChoices.OPTIONS):
            return json.loads(self.value) if self.value else ([] if self.value_type == self.TypeChoices.OPTIONS else None)
        return self.value

    def get_options(self):
        """获取选项列表 — 仅对 options 类型有效"""
        if self.value_type != self.TypeChoices.OPTIONS:
            return None
        return self.parse_value()

    @classmethod
    def get_value(cls, key, default=None):
        """类方法 — 带缓存读取配置（用于系统参数）"""
        cache_key = f"config:{key}"
        cached = cache.get(cache_key)
        if cached is not None:
            return cached

        try:
            obj = cls.objects.get(key=key, is_active=True)
            val = obj.parse_value()
            cache.set(cache_key, val, timeout=3600)
            return val
        except cls.DoesNotExist:
            return default

    @classmethod
    def get_by_group(cls, group):
        """
        按分组批量获取配置（用于选项列表查询）

        返回 {key1: parsed_value, key2: parsed_value, ...}
        options 类型的 value 自动解析为选项列表
        """
        cache_key = f"config_group:{group}"
        cached = cache.get(cache_key)
        if cached is not None:
            return cached

        configs = cls.objects.filter(group=group, is_active=True).values("key", "value", "value_type", "encrypted_value")
        result = {}
        for c in configs:
            obj = cls(key=c["key"], value=c["value"], value_type=c["value_type"], encrypted_value=c.get("encrypted_value", ""))
            result[c["key"]] = obj.parse_value()

        cache.set(cache_key, result, timeout=3600)
        return result


@receiver([post_save, post_delete], sender=Config)
def clear_config_cache(sender, instance, **kwargs):
    cache.delete(f"config:{instance.key}")
    cache.delete(f"config_group:{instance.group}")