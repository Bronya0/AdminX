import os
import uuid

from django.conf import settings
from django.core.files.storage import FileSystemStorage
from django.utils.deconstruct import deconstructible


@deconstructible
class LocalStorageBackend:
    """本地文件存储"""

    def __init__(self):
        self.base_dir = settings.MEDIA_ROOT / "uploads"
        os.makedirs(self.base_dir, exist_ok=True)

    def save(self, file_obj, filename):
        """保存文件，返回 storage_path"""
        # 生成唯一文件名避免冲突
        ext = os.path.splitext(filename)[1]
        unique_name = f"{uuid.uuid4().hex}{ext}"
        # 按日期分目录
        from datetime import date
        sub_dir = date.today().strftime("%Y/%m/%d")
        save_dir = self.base_dir / sub_dir
        os.makedirs(save_dir, exist_ok=True)

        storage_path = os.path.join(sub_dir, unique_name)
        full_path = self.base_dir / storage_path

        with open(full_path, "wb") as f:
            for chunk in file_obj.chunks():
                f.write(chunk)

        return storage_path.replace("\\", "/")

    def url(self, storage_path):
        """返回访问 URL"""
        return f"{settings.MEDIA_URL}uploads/{storage_path}"

    def delete(self, storage_path):
        """删除物理文件"""
        full_path = self.base_dir / storage_path
        try:
            os.remove(full_path)
        except FileNotFoundError:
            pass