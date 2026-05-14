from django.db import models
from rest_framework import viewsets
from rest_framework.decorators import action
from rest_framework.permissions import IsAuthenticated
from rest_framework.response import Response

from .models import Notification
from .serializers import NotificationSerializer


class NotificationViewSet(viewsets.ReadOnlyModelViewSet):
    """通知 — 只读 + 标记已读"""
    serializer_class = NotificationSerializer
    permission_classes = [IsAuthenticated]
    ordering = ["-created_at"]

    def get_queryset(self):
        user = self.request.user
        if user.is_superuser:
            return Notification.objects.all()
        return Notification.objects.filter(
            models.Q(user__isnull=True) | models.Q(user=user)
        )

    @action(detail=True, methods=["post"])
    def mark_read(self, request, pk=None):
        """标记单条通知为已读"""
        notification = self.get_object()
        notification.is_read = True
        notification.save(update_fields=["is_read"])
        return Response({"code": 200, "msg": "success"})

    @action(detail=False, methods=["post"])
    def mark_all_read(self, request):
        """标记所有通知为已读"""
        self.get_queryset().filter(is_read=False).update(is_read=True)
        return Response({"code": 200, "msg": "success"})

    @action(detail=False, methods=["get"])
    def unread_count(self, request):
        """未读通知数量"""
        count = self.get_queryset().filter(is_read=False).count()
        return Response({"code": 200, "msg": "success", "data": {"count": count}})
