import uuid

from django.db import models


class ClusterNode(models.Model):
    """集群节点"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    name = models.CharField("节点名称", max_length=128, unique=True)
    host = models.CharField("主机地址", max_length=255)
    port = models.IntegerField("端口", default=8000)
    role = models.CharField("角色", max_length=32,
                            choices=[("master", "主节点"), ("slave", "从节点")],
                            default="slave")
    status = models.CharField("状态", max_length=20,
                              choices=[("online", "在线"), ("offline", "离线"), ("maintenance", "维护")],
                              default="offline")
    version = models.CharField("版本", max_length=32, blank=True, default="")
    last_heartbeat = models.DateTimeField("最后心跳", null=True, blank=True)
    is_active = models.BooleanField("启用", default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "集群节点"
        verbose_name_plural = "集群节点"
        ordering = ["name"]

    def __str__(self):
        return f"{self.name} ({self.host}:{self.port})"