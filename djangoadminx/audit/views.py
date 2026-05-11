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