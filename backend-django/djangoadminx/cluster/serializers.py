from rest_framework import serializers

from .models import ClusterNode, ServiceComponent


class ClusterNodeSerializer(serializers.ModelSerializer):
    class Meta:
        model = ClusterNode
        fields = "__all__"
        read_only_fields = ["id", "last_heartbeat", "created_at", "updated_at"]


class ServiceComponentSerializer(serializers.ModelSerializer):
    status = serializers.ReadOnlyField()
    has_pending_upgrade = serializers.ReadOnlyField()
    has_pending_uninstall = serializers.ReadOnlyField()
    pending_command = serializers.ReadOnlyField()

    class Meta:
        model = ServiceComponent
        fields = [
            "id", "app_label", "name", "version", "host", "description",
            "last_heartbeat", "status", "has_pending_upgrade", "has_pending_uninstall",
            "pending_command", "upgrade_version", "upgrade_url", "upgrade_checksum",
            "uninstall_pending", "extra_info", "registered_at", "updated_at",
        ]
        read_only_fields = ["id", "last_heartbeat", "registered_at", "updated_at"]


class ServiceComponentUpgradeSerializer(serializers.Serializer):
    """管理员设置升级任务"""
    upgrade_version = serializers.CharField(max_length=64)
    upgrade_url = serializers.CharField()
    upgrade_checksum = serializers.CharField(max_length=64, required=False, default="")


class HeartbeatSerializer(serializers.Serializer):
    """业务服务上报心跳"""
    app_label = serializers.CharField(max_length=128)
    version = serializers.CharField(max_length=64, required=False, default="")
    host = serializers.CharField(required=False, default="")
    extra_info = serializers.DictField(required=False, default=dict)