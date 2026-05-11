from django.contrib import admin

from .models import AuditLog


@admin.register(AuditLog)
class AuditLogAdmin(admin.ModelAdmin):
    list_display = ["action", "model_name", "object_repr", "operator", "diff_summary", "created_at"]
    list_filter = ["action", "model_name", "created_at"]
    search_fields = ["operator", "object_repr", "diff_summary"]
    readonly_fields = [f.name for f in AuditLog._meta.fields]  # 全部只读
    date_hierarchy = "created_at"

    def has_add_permission(self, request):
        return False

    def has_delete_permission(self, request, obj=None):
        return False