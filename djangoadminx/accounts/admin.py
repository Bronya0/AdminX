from django.contrib import admin

from .models import LoginLock, Role, User, UserLoginLog


@admin.register(User)
class UserAdmin(admin.ModelAdmin):
    list_display = ["username", "email", "phone", "is_active", "is_superuser", "date_joined"]
    list_filter = ["is_active", "is_superuser", "roles"]
    search_fields = ["username", "email", "phone"]
    filter_horizontal = ["roles", "groups", "user_permissions"]


@admin.register(Role)
class RoleAdmin(admin.ModelAdmin):
    list_display = ["name", "code", "is_active", "created_at"]
    list_filter = ["is_active"]
    search_fields = ["name", "code"]
    filter_horizontal = ["permissions", "menus"]


@admin.register(UserLoginLog)
class UserLoginLogAdmin(admin.ModelAdmin):
    list_display = ["username", "ip", "success", "message", "created_at"]
    list_filter = ["success", "created_at"]
    search_fields = ["username", "ip"]
    readonly_fields = ["username", "ip", "user_agent", "success", "message", "created_at"]


@admin.register(LoginLock)
class LoginLockAdmin(admin.ModelAdmin):
    list_display = ["username", "failed_count", "locked_at", "unlocked_at"]
    search_fields = ["username"]