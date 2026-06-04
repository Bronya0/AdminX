from django.db import models
from rest_framework import viewsets
from rest_framework.decorators import action
from rest_framework.permissions import IsAdminUser, IsAuthenticated

from djangoadminx.accounts.permissions import RBACPermission
from rest_framework.response import Response

from .models import Notification, WebhookConfig, WebhookLog
from .serializers import NotificationSerializer, WebhookConfigSerializer, WebhookLogSerializer


class NotificationViewSet(viewsets.ReadOnlyModelViewSet):
    """通知 — 只读 + 标记已读"""
    serializer_class = NotificationSerializer
    permission_classes = [IsAuthenticated, RBACPermission]
    ordering = ["-created_at"]
    search_fields = ["title", "content"]
    filterset_fields = ["notification_type", "is_read"]

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


class WebhookConfigViewSet(viewsets.ModelViewSet):
    """Webhook 配置 CRUD"""
    queryset = WebhookConfig.objects.all()
    serializer_class = WebhookConfigSerializer
    permission_classes = [IsAdminUser]
    search_fields = ["name", "url"]
    ordering_fields = ["name", "created_at"]

    @action(detail=True, methods=["post"])
    def test(self, request, pk=None):
        """测试发送"""
        config = self.get_object()
        from .webhook import send_webhook
        from .models import Notification
        notification = Notification.objects.create(
            title=f"Webhook 测试 - {config.name}",
            content="这是一条测试消息",
            notification_type="info",
        )
        send_webhook(config, notification)
        return Response({"code": 200, "msg": "测试消息已发送"})


class WebhookLogViewSet(viewsets.ReadOnlyModelViewSet):
    """Webhook 发送日志"""
    queryset = WebhookLog.objects.select_related("webhook").all()
    serializer_class = WebhookLogSerializer
    permission_classes = [IsAdminUser]
    ordering = ["-created_at"]
    filterset_fields = ["webhook", "status"]
