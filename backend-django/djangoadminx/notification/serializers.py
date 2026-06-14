from rest_framework import serializers
from .models import Notification, WebhookConfig, WebhookLog


class NotificationSerializer(serializers.ModelSerializer):
    class Meta:
        model = Notification
        fields = ["id", "title", "content", "notification_type", "is_read", "created_at"]
        read_only_fields = ["id", "created_at"]


class WebhookConfigSerializer(serializers.ModelSerializer):
    class Meta:
        model = WebhookConfig
        fields = "__all__"
        read_only_fields = ["id", "created_at", "updated_at"]


class WebhookLogSerializer(serializers.ModelSerializer):
    webhook_name = serializers.CharField(source="webhook.name", read_only=True)

    class Meta:
        model = WebhookLog
        fields = ["id", "webhook", "webhook_name", "notification", "status", "response_status", "response_body", "error_message", "created_at"]
        read_only_fields = ["id", "created_at"]
