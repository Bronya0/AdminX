from django.contrib import admin

from .models import JobLog, ScheduleJob, SchedulerHeartbeat


@admin.register(ScheduleJob)
class ScheduleJobAdmin(admin.ModelAdmin):
    list_display = ["name", "handler", "trigger_type", "is_active", "updated_at"]
    list_filter = ["trigger_type", "is_active"]
    search_fields = ["name", "handler"]


@admin.register(JobLog)
class JobLogAdmin(admin.ModelAdmin):
    list_display = ["job", "status", "started_at", "finished_at"]
    list_filter = ["status"]
    readonly_fields = ["job", "status", "result", "started_at", "finished_at"]


@admin.register(SchedulerHeartbeat)
class SchedulerHeartbeatAdmin(admin.ModelAdmin):
    list_display = ["id", "last_heartbeat", "reload_pending"]
    readonly_fields = ["id", "last_heartbeat", "reload_pending"]
