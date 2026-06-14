from django.contrib import admin

from .models import Config


@admin.register(Config)
class ConfigAdmin(admin.ModelAdmin):
    list_display = ["key", "value_type", "group", "is_active", "updated_at"]
    list_filter = ["value_type", "group", "is_active"]
    search_fields = ["key", "desc"]
    list_editable = ["is_active"]
    readonly_fields = ["created_at", "updated_at"]