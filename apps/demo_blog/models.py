"""
业务模型定义 — 规范示例

最佳实践:
  1. UUID 主键（不暴露自增 ID，安全且方便分布式）
  2. auto_now_add / auto_now 记录时间戳
  3. class Meta 中配置 verbose_name、ordering、indexes
  4. __str__ 返回可读描述
  5. 字段用 verbose_name 加中文名
"""
import uuid

from django.conf import settings
from django.db import models
from djangoadminx.config_center.models import Config


class Category(models.Model):
    """文章分类"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    name = models.CharField("分类名称", max_length=64, unique=True)
    sort_order = models.IntegerField("排序", default=0)
    is_active = models.BooleanField("启用", default=True)
    created_at = models.DateTimeField("创建时间", auto_now_add=True)

    class Meta:
        verbose_name = "分类"
        verbose_name_plural = "分类"
        ordering = ["sort_order", "name"]
        indexes = [
            models.Index(fields=["name"]),
            models.Index(fields=["is_active"]),
        ]

    def __str__(self):
        return self.name


class Post(models.Model):
    """文章"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    title = models.CharField("标题", max_length=255)
    content = models.TextField("内容")
    summary = models.TextField("摘要", blank=True, default="")

    category = models.ForeignKey(
        Category,
        on_delete=models.SET_NULL,
        null=True,
        blank=True,
        verbose_name="分类",
        related_name="posts",
    )

    # 发布状态
    STATUS_DRAFT = "draft"
    STATUS_PUBLISHED = "published"
    STATUS_ARCHIVED = "archived"
    STATUS_CHOICES = [
        (STATUS_DRAFT, "草稿"),
        (STATUS_PUBLISHED, "已发布"),
        (STATUS_ARCHIVED, "已归档"),
    ]
    status = models.CharField("状态", max_length=20, choices=STATUS_CHOICES, default=STATUS_DRAFT)

    # 发布人（关联框架的 User 模型）
    author = models.ForeignKey(
        settings.AUTH_USER_MODEL,
        on_delete=models.SET_NULL,
        null=True,
        blank=True,
        verbose_name="作者",
        related_name="posts",
    )

    # 标签 — 简单逗号分隔
    tags = models.CharField("标签", max_length=255, blank=True, default="",
                            help_text="多个标签用逗号分隔")

    # 浏览量
    view_count = models.PositiveIntegerField("浏览量", default=0)

    # 时间戳
    created_at = models.DateTimeField("创建时间", auto_now_add=True)
    updated_at = models.DateTimeField("更新时间", auto_now=True)
    published_at = models.DateTimeField("发布时间", null=True, blank=True)

    class Meta:
        verbose_name = "文章"
        verbose_name_plural = "文章"
        ordering = ["-created_at"]
        indexes = [
            models.Index(fields=["status", "created_at"]),
            models.Index(fields=["author"]),
            models.Index(fields=["category"]),
        ]

    def __str__(self):
        return self.title

    @classmethod
    def get_status_options(cls):
        """
        从配置中心获取动态状态选项（带缓存），无配置时回退硬编码选项

        返回: [{"label": "草稿", "value": "draft"}, ...]
        """
        try:
            options = Config.get_value("post_status_options")
            if isinstance(options, list) and options:
                return options
        except Exception:
            pass
        return [{"label": label, "value": value} for value, label in cls.STATUS_CHOICES]