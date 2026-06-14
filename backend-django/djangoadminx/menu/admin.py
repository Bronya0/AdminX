from django.contrib import admin

from .models import Menu


@admin.register(Menu)
class MenuAdmin(admin.ModelAdmin):
    list_display = ["name", "code", "menu_type", "is_active", "is_visible", "sort_order"]
    list_filter = ["menu_type", "is_active"]
    search_fields = ["name", "code"]
    list_editable = ["sort_order", "is_active", "is_visible"]