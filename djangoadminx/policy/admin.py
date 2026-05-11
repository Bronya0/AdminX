from django.contrib import admin

from .models import PasswordHistory, PasswordPolicy


@admin.register(PasswordPolicy)
class PasswordPolicyAdmin(admin.ModelAdmin):
    list_display = [f.name for f in PasswordPolicy._meta.fields if f.name != "id"]

    def has_add_permission(self, request):
        return not PasswordPolicy.objects.exists()


@admin.register(PasswordHistory)
class PasswordHistoryAdmin(admin.ModelAdmin):
    list_display = ["user", "created_at"]
    search_fields = ["user__username"]
    readonly_fields = ["user", "password_hash", "created_at"]