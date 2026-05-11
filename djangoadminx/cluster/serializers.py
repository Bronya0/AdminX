from rest_framework import serializers

from .models import ClusterNode


class ClusterNodeSerializer(serializers.ModelSerializer):
    class Meta:
        model = ClusterNode
        fields = "__all__"
        read_only_fields = ["id", "last_heartbeat", "created_at", "updated_at"]