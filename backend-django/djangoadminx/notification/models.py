import uuid

from django.conf import settings
from django.db import models


class WebhookConfig(models.Model):
    """Webhook 外发配置"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    name = models.CharField("名称", max_length=128)
    url = models.URLField("Webhook URL", max_length=512)
    secret = models.CharField("签名密钥", max_length=256, blank=True, default="")
    events = models.CharField("触发事件", max_length=256, blank=True, default="info,success,warning,error",
                              help_text="逗号分隔的通知类型，空=全部")
    is_active = models.BooleanField("启用", default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "Webhook 配置"
        verbose_name_plural = "Webhook 配置"
        ordering = ["-created_at"]

    def __str__(self):
        return self.name


class WebhookLog(models.Model):
    """Webhook 发送日志"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    webhook = models.ForeignKey(WebhookConfig, on_delete=models.CASCADE, verbose_name="Webhook 配置")
    notification = models.ForeignKey("Notification", on_delete=models.SET_NULL, null=True, blank=True, verbose_name="关联通知")
    status = models.CharField("状态", max_length=20, choices=[("success", "成功"), ("failed", "失败")], default="success")
    response_status = models.IntegerField("HTTP 状态码", null=True, blank=True)
    response_body = models.TextField("响应内容", blank=True, default="")
    error_message = models.TextField("错误信息", blank=True, default="")
    created_at = models.DateTimeField("发送时间", auto_now_add=True)

    class Meta:
        verbose_name = "Webhook 发送日志"
        verbose_name_plural = "Webhook 发送日志"
        ordering = ["-created_at"]

    def __str__(self):
        return f"{self.webhook.name} - {self.status}"


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
