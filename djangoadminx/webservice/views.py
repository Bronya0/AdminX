import json
import logging
import time

from rest_framework import viewsets
from rest_framework.decorators import action
from rest_framework.permissions import IsAdminUser
from rest_framework.response import Response

from .models import JobLog, ScheduleJob, WebService, WebServiceLog
from .serializers import (
    JobLogSerializer,
    ScheduleJobSerializer,
    WebServiceLogSerializer,
    WebServiceSerializer,
)

logger = logging.getLogger("djangoadminx.webservice")


class WebServiceViewSet(viewsets.ModelViewSet):
    """WebService 配置 CRUD"""
    queryset = WebService.objects.all()
    serializer_class = WebServiceSerializer
    permission_classes = [IsAdminUser]
    search_fields = ["name", "wsdl_url", "method"]
    ordering_fields = ["name", "created_at"]

    @action(detail=True, methods=["post"])
    def invoke(self, request, pk=None):
        """调用 WebService"""
        ws = self.get_object()
        params = request.data.get("params", {})

        start = time.time()
        try:
            from zeep import Client
            client = Client(ws.wsdl_url)
            method = getattr(client.service, ws.method)
            result = method(**params)
            cost = int((time.time() - start) * 1000)
            WebServiceLog.objects.create(
                webservice=ws,
                method=ws.method,
                request_body=json.dumps(params, ensure_ascii=False),
                response_body=str(result),
                status="success",
                cost_ms=cost,
            )
            return Response({"code": 200, "msg": "success", "data": {"result": str(result), "cost_ms": cost}})
        except Exception as e:
            cost = int((time.time() - start) * 1000)
            WebServiceLog.objects.create(
                webservice=ws,
                method=ws.method,
                request_body=json.dumps(params, ensure_ascii=False),
                status="failed",
                error_msg=str(e),
                cost_ms=cost,
            )
            return Response({"code": 500, "msg": str(e)}, status=200)


class ScheduleJobViewSet(viewsets.ModelViewSet):
    """定时任务 CRUD"""
    queryset = ScheduleJob.objects.all()
    serializer_class = ScheduleJobSerializer
    permission_classes = [IsAdminUser]
    search_fields = ["name", "handler"]
    ordering_fields = ["name", "created_at"]

    @action(detail=True, methods=["post"])
    def run_once(self, request, pk=None):
        """立即执行一次"""
        job = self.get_object()
        try:
            result = job.execute()
            return Response({"code": 200, "msg": "success", "data": {"result": str(result)}})
        except Exception as e:
            return Response({"code": 500, "msg": str(e)}, status=200)

    @action(detail=False, methods=["get"])
    def status(self, request):
        """调度器状态"""
        try:
            from djangoadminx.common.scheduler import scheduler_manager
            jobs = scheduler_manager.scheduler.get_jobs()
            return Response({
                "code": 200,
                "msg": "success",
                "data": {
                    "running": scheduler_manager.scheduler.running,
                    "job_count": len(jobs),
                    "jobs": [{"id": j.id, "name": j.name, "next_run": str(j.next_run_time)} for j in jobs],
                },
            })
        except Exception as e:
            return Response({"code": 500, "msg": str(e)}, status=200)

    @action(detail=False, methods=["post"])
    def reload(self, request):
        """全量重载任务"""
        try:
            from djangoadminx.common.scheduler import scheduler_manager
            scheduler_manager.reload_all()
            return Response({"code": 200, "msg": "success"})
        except Exception as e:
            return Response({"code": 500, "msg": str(e)}, status=200)


class JobLogViewSet(viewsets.ReadOnlyModelViewSet):
    """任务日志"""
    queryset = JobLog.objects.select_related("job").all()
    serializer_class = JobLogSerializer
    ordering = ["-started_at"]


class WebServiceLogViewSet(viewsets.ReadOnlyModelViewSet):
    """WebService 调用日志"""
    queryset = WebServiceLog.objects.select_related("webservice").all()
    serializer_class = WebServiceLogSerializer
    ordering = ["-created_at"]