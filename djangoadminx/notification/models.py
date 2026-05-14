import uuid

from django.conf import settings
from django.db import models


class Notification(models.Model):
    """系统通知"""

    class TypeChoices(models.TextChoices):
        INFO = "info", "信息"
        SUCCESS = "success", "成功"
        WARNING = "warning", "警告"
        ERROR = "error", "错误"

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    user = models.ForeignKey(
        settings.AUTH_USER_MODEL,
        on_delete=models.CASCADE,
        verbose_name="接收用户",
        null=True, blank=True,
        help_text="空=全员通知",
    )
    title = models.CharField("标题", max_length=255)
    content = models.TextField("内容", blank=True, default="")
    notification_type = models.CharField("类型", max_length=20, choices=TypeChoices.choices, default=TypeChoices.INFO)
    is_read = models.BooleanField("是否已读", default=False)
    created_at = models.DateTimeField("创建时间", auto_now_add=True)

    class Meta:
        verbose_name = "通知"
        verbose_name_plural = "通知"
        ordering = ["-created_at"]

    def __str__(self):
        return f"[{self.get_notification_type_display()}] {self.title}"
