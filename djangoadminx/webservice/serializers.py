from rest_framework import serializers

from .models import JobLog, ScheduleJob, WebService, WebServiceLog


class WebServiceSerializer(serializers.ModelSerializer):
    """WebService 序列化器 — auth_password 仅写入，不返回明文"""
    display_auth_password = serializers.SerializerMethodField()

    class Meta:
        model = WebService
        fields = "__all__"
        read_only_fields = ["id", "created_at", "updated_at"]
        extra_kwargs = {
            "auth_password": {"write_only": True},
        }

    @staticmethod
    def get_display_auth_password(obj):
        return "********" if obj.auth_password else ""


class ScheduleJobSerializer(serializers.ModelSerializer):
    class Meta:
        model = ScheduleJob
        fields = "__all__"
        read_only_fields = ["id", "created_at", "updated_at"]


class JobLogSerializer(serializers.ModelSerializer):
    job_name = serializers.CharField(source="job.name", read_only=True)

    class Meta:
        model = JobLog
        fields = "__all__"


class WebServiceLogSerializer(serializers.ModelSerializer):
    service_name = serializers.CharField(source="webservice.name", read_only=True)

    class Meta:
        model = WebServiceLog
        fields = "__all__"