from django.http import JsonResponse


class HealthCheckMixin:
    """健康检查视图混入"""

    def health(self, request):
        import psutil

        return JsonResponse({
            "status": "ok",
            "cpu_percent": psutil.cpu_percent(interval=1),
            "memory_percent": psutil.virtual_memory().percent,
            "disk_percent": psutil.disk_usage("/").percent,
        })