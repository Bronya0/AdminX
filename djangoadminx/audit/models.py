import uuid

from django.db import models


class AuditLog(models.Model):
    """操作审计日志 — 记录核心模型的增删改"""

    class ActionChoices(models.TextChoices):
        CREATE = "create", "创建"
        UPDATE = "update", "更新"
        DELETE = "delete", "删除"

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    action = models.CharField("操作类型", max_length=20, choices=ActionChoices.choices)
    model_name = models.CharField("模型名", max_length=128, db_index=True)
    object_id = models.CharField("对象ID", max_length=128, blank=True, default="")
    object_repr = models.CharField("对象描述", max_length=255, blank=True, default="")
    operator = models.CharField("操作人", max_length=128, blank=True, default="", db_index=True)
    operator_ip = models.GenericIPAddressField("IP", blank=True, null=True)
    old_values = models.JSONField("变更前", null=True, blank=True, default=None)
    new_values = models.JSONField("变更后", null=True, blank=True, default=None)
    diff_summary = models.CharField("变更摘要", max_length=500, blank=True, default="")
    created_at = models.DateTimeField("操作时间", auto_now_add=True, db_index=True)

    class Meta:
        verbose_name = "审计日志"
        verbose_name_plural = "审计日志"
        ordering = ["-created_at"]
        indexes = [
            models.Index(fields=["model_name", "object_id"]),
        ]

    def __str__(self):
        return f"[{self.action}] {self.model_name} #{self.object_id} by {self.operator}"