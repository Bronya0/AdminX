import os
import uuid

from django.conf import settings


class MinioStorageBackend:
    """MinIO 对象存储"""

    def __init__(self):
        self.bucket = getattr(settings, "MINIO_BUCKET", "uploads")
        self.client = self._get_client()

    def _get_client(self):
        from minio import Minio
        return Minio(
            settings.MINIO_ENDPOINT,
            access_key=settings.MINIO_ACCESS_KEY,
            secret_key=settings.MINIO_SECRET_KEY,
            secure=getattr(settings, "MINIO_SECURE", True),
        )

    def save(self, file_obj, filename):
        ext = os.path.splitext(filename)[1]
        object_name = f"{uuid.uuid4().hex}{ext}"
        from datetime import date
        prefix = date.today().strftime("%Y/%m/%d")
        storage_path = f"{prefix}/{object_name}"

        # 确保 bucket 存在
        if not self.client.bucket_exists(self.bucket):
            self.client.make_bucket(self.bucket)

        self.client.put_object(
            self.bucket,
            storage_path,
            file_obj,
            length=-1,
            part_size=10 * 1024 * 1024,
            content_type=file_obj.content_type,
        )
        return storage_path

    def url(self, storage_path):
        return f"{self.client._base_url}/{self.bucket}/{storage_path}"

    def delete(self, storage_path):
        try:
            self.client.remove_object(self.bucket, storage_path)
        except Exception:
            pass