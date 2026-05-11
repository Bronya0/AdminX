from django.contrib import admin

from .models import ClusterNode


@admin.register(ClusterNode)
class ClusterNodeAdmin(admin.ModelAdmin):
    list_display = ["name", "host", "port", "role", "status", "last_heartbeat", "is_active"]
    list_filter = ["role", "status", "is_active"]
    search_fields = ["name", "host"]