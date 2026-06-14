from django.db import models
from django.utils import timezone


class SystemMetricSnapshot(models.Model):
    """系统资源历史快照"""

    collected_at = models.DateTimeField("采集时间", default=timezone.now, db_index=True)
    cpu_percent = models.FloatField("CPU 使用率", default=0)
    memory_percent = models.FloatField("内存使用率", default=0)
    disk_percent = models.FloatField("磁盘使用率", default=0)
    disk_read_bytes = models.BigIntegerField("磁盘累计读取字节", default=0)
    disk_write_bytes = models.BigIntegerField("磁盘累计写入字节", default=0)

    class Meta:
        verbose_name = "系统资源快照"
        verbose_name_plural = "系统资源快照"
        ordering = ["-collected_at"]

    def __str__(self):
        return self.collected_at.strftime("%Y-%m-%d %H:%M:%S")
