import os
import uuid

from django.conf import settings
from django.db import models


class FileRecord(models.Model):
    """文件记录 — 统一管理上传文件"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    original_name = models.CharField("原始文件名", max_length=512)
    size = models.BigIntegerField("文件大小(bytes)")
    mime_type = models.CharField("MIME 类型", max_length=128, blank=True, default="")
    storage_backend = models.CharField("存储后端", max_length=32, default="local")
    storage_path = models.CharField("存储路径", max_length=1024)
    url = models.URLField("访问 URL", max_length=1024, blank=True, default="")
    uploaded_by = models.CharField("上传人", max_length=128, blank=True, default="")
    created_at = models.DateTimeField("上传时间", auto_now_add=True)

    class Meta:
        verbose_name = "文件记录"
        verbose_name_plural = "文件记录"
        ordering = ["-created_at"]

    def __str__(self):
        return self.original_name

    def delete(self, *args, **kwargs):
        """删除时同时删除物理文件"""
        backend = get_storage_backend(self.storage_backend)
        backend.delete(self.storage_path)
        super().delete(*args, **kwargs)


def get_storage_backend(name="local"):
    """工厂方法 — 获取存储后端实例"""
    from .storage_backends.local import LocalStorageBackend
    from .storage_backends.minio_backend import MinioStorageBackend

    backends = {
        "local": LocalStorageBackend,
        "minio": MinioStorageBackend,
    }
    cls = backends.get(name, LocalStorageBackend)
    return cls()