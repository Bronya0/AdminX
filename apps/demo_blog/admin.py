"""
Django admin 注册

最佳实践:
  1. 注册所有模型，方便运维管理
  2. list_display / list_filter / search_fields 提供常用筛选
"""
from django.contrib import admin

from .models import Category, Post


@admin.register(Category)
class CategoryAdmin(admin.ModelAdmin):
    list_display = ["name", "sort_order", "is_active", "post_count", "created_at"]
    list_filter = ["is_active"]
    search_fields = ["name"]

    @admin.display(description="文章数")
    def post_count(self, obj):
        return obj.posts.count()


@admin.register(Post)
class PostAdmin(admin.ModelAdmin):
    list_display = ["title", "status", "category", "author", "view_count", "created_at", "published_at"]
    list_filter = ["status", "category"]
    search_fields = ["title", "content"]
    readonly_fields = ["view_count", "created_at", "updated_at"]
    date_hierarchy = "created_at"