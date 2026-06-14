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


class ServiceComponent(models.Model):
    """业务组件注册表 — 业务系统自主注册，30s 心跳，60s 无心跳视为离线"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    app_label = models.CharField("应用标识", max_length=128, unique=True,
                                 help_text="全局唯一标识，如 business-service")
    name = models.CharField("组件名称", max_length=255)
    version = models.CharField("当前版本", max_length=64, blank=True, default="")
    host = models.CharField("服务地址", max_length=255, blank=True, default="",
                            help_text="业务服务的访问地址，如 http://10.0.0.2:8001")
    description = models.TextField("描述", blank=True, default="")
    last_heartbeat = models.DateTimeField("最后心跳", null=True, blank=True)
    # 升级任务
    upgrade_version = models.CharField("待升级版本", max_length=64, blank=True, default="")
    upgrade_url = models.TextField("升级包下载地址", blank=True, default="")
    upgrade_checksum = models.CharField("升级包 SHA-256", max_length=64, blank=True, default="")
    # 卸载任务
    uninstall_pending = models.BooleanField("待卸载", default=False)
    # 元数据
    extra_info = models.JSONField("附加信息", default=dict, blank=True)
    registered_at = models.DateTimeField("注册时间", auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    HEARTBEAT_TTL = 60  # 超过 60s 无心跳视为离线

    class Meta:
        verbose_name = "业务组件"
        verbose_name_plural = "业务组件"
        ordering = ["name"]

    @property
    def status(self):
        if not self.last_heartbeat:
            return "offline"
        from django.utils import timezone
        elapsed = (timezone.now() - self.last_heartbeat).total_seconds()
        return "online" if elapsed < self.HEARTBEAT_TTL else "offline"

    @property
    def has_pending_upgrade(self):
        return bool(self.upgrade_version)

    @property
    def has_pending_uninstall(self):
        return bool(self.uninstall_pending)

    @property
    def pending_command(self):
        """当前待执行指令：upgrade / uninstall / None"""
        if self.uninstall_pending:
            return "uninstall"
        if self.upgrade_version:
            return "upgrade"
        return None

    def __str__(self):
        return f"{self.name} ({self.app_label})"