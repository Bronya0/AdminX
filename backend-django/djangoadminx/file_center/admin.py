from django.contrib import admin

from .models import FileRecord


@admin.register(FileRecord)
class FileRecordAdmin(admin.ModelAdmin):
    list_display = ["original_name", "size", "mime_type", "storage_backend", "uploaded_by", "created_at"]
    list_filter = ["storage_backend", "mime_type"]
    search_fields = ["original_name", "uploaded_by"]
    readonly_fields = [f.name for f in FileRecord._meta.fields]