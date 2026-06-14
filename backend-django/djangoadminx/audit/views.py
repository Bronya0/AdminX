from rest_framework import viewsets

from .models import AuditLog
from .serializers import AuditLogSerializer


class AuditLogViewSet(viewsets.ReadOnlyModelViewSet):
    """审计日志 — 只读"""
    queryset = AuditLog.objects.all()
    serializer_class = AuditLogSerializer
    ordering_fields = ["created_at"]
    ordering = ["-created_at"]
    search_fields = ["operator", "object_repr", "model_name", "diff_summary"]
    filterset_fields = ["action"]

    def filter_queryset(self, queryset):
        queryset = super().filter_queryset(queryset)
        date_from = self.request.query_params.get("created_at__gte")
        if date_from:
            queryset = queryset.filter(created_at__gte=date_from)
        date_to = self.request.query_params.get("created_at__lte")
        if date_to:
            queryset = queryset.filter(created_at__lte=date_to)
        model_name = self.request.query_params.get("model_name")
        if model_name:
            queryset = queryset.filter(model_name__icontains=model_name)
        return queryset