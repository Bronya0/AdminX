from django.contrib import admin

from .models import JobLog, ScheduleJob, SchedulerHeartbeat, WebService, WebServiceLog


@admin.register(WebService)
class WebServiceAdmin(admin.ModelAdmin):
    list_display = ["name", "type", "method", "wsdl_url", "is_active"]
    list_filter = ["type", "is_active"]
    search_fields = ["name", "method", "wsdl_url"]


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


@admin.register(WebServiceLog)
class WebServiceLogAdmin(admin.ModelAdmin):
    list_display = ["webservice", "method", "status", "cost_ms", "created_at"]
    list_filter = ["status"]
    readonly_fields = ["webservice", "method", "request_body", "response_body", "status", "error_msg", "cost_ms"]