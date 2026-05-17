import json
import logging

from rest_framework import viewsets
from rest_framework.decorators import action
from rest_framework.permissions import IsAdminUser, IsAuthenticated
from rest_framework.response import Response

from djangoadminx.audit.mixins import AuditLogMixin
from .models import JobLog, ScheduleJob
from .serializers import JobLogSerializer, ScheduleJobSerializer

logger = logging.getLogger("djangoadminx.jobs")


class ScheduleJobViewSet(AuditLogMixin, viewsets.ModelViewSet):
    """定时任务 CRUD"""
    queryset = ScheduleJob.objects.all()
    serializer_class = ScheduleJobSerializer
    permission_classes = [IsAdminUser]
    search_fields = ["name", "handler", "command"]
    ordering_fields = ["name", "created_at"]
    filterset_fields = ["command_type", "trigger_type", "is_active"]

    @action(detail=True, methods=["post"])
    def run_once(self, request, pk=None):
        """立即执行一次（同时写入执行日志）"""
        from django.utils import timezone
        from .models import JobLog

        job = self.get_object()
        started_at = timezone.now()
        log = JobLog.objects.create(job=job, status="running", started_at=started_at)
        try:
            result = job.execute()
            log.status = "success"
            log.result = str(result)[:500] if result is not None else "ok"
            log.finished_at = timezone.now()
            log.save()
            return Response({"code": 200, "msg": "success", "data": {"result": log.result}})
        except Exception as e:
            log.status = "failed"
            log.result = str(e)[:500]
            log.finished_at = timezone.now()
            log.save()
            return Response({"code": 500, "msg": str(e)}, status=200)

    @action(detail=False, methods=["get"])
    def status(self, request):
        """调度器状态（通过心跳检测独立调度器进程是否存活）"""
        try:
            from djangoadminx.common.scheduler import scheduler_manager
            from .models import ScheduleJob

            running = scheduler_manager.is_alive()
            job_count = ScheduleJob.objects.filter(is_active=True).count()

            job_list = []
            try:
                sched = scheduler_manager._scheduler
                if sched and sched.running:
                    for j in sched.get_jobs():
                        next_run = str(j.next_run_time) if j.next_run_time else None
                        job_list.append({"id": j.id, "name": j.name, "next_run": next_run})
            except Exception:
                pass

            return Response({
                "code": 200,
                "msg": "success",
                "data": {
                    "running": running,
                    "job_count": job_count,
                    "jobs": job_list,
                },
            })
        except Exception as e:
            logger.warning(f"获取调度器状态异常: {e}")
            return Response({"code": 200, "msg": "success", "data": {"running": False, "job_count": 0, "jobs": []}})

    @action(detail=False, methods=["post"])
    def reload(self, request):
        """通知调度器进程全量重载"""
        try:
            from djangoadminx.common.scheduler import scheduler_manager
            scheduler_manager.notify_reload()
            return Response({"code": 200, "msg": "已通知调度器重载"})
        except Exception as e:
            return Response({"code": 500, "msg": str(e)}, status=200)


class JobLogViewSet(viewsets.ReadOnlyModelViewSet):
    """任务日志"""
    queryset = JobLog.objects.select_related("job").all()
    serializer_class = JobLogSerializer
    ordering = ["-started_at"]
    filterset_fields = ["job"]
