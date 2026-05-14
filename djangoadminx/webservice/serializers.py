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
    last_run = serializers.SerializerMethodField()

    class Meta:
        model = ScheduleJob
        fields = "__all__"
        read_only_fields = ["id", "created_at", "updated_at"]

    def validate(self, attrs):
        if attrs.get("command_type") == "python" and not attrs.get("handler"):
            raise serializers.ValidationError({"handler": "Python 类型任务必须填写处理函数"})
        if attrs.get("command_type") == "shell" and not attrs.get("command"):
            raise serializers.ValidationError({"command": "Shell 类型任务必须填写命令"})
        return attrs

    @staticmethod
    def get_last_run(obj):
        last = JobLog.objects.filter(job=obj).first()
        if last:
            return {
                "status": last.status,
                "result": last.result[:200] if last.result else "",
                "started_at": last.started_at,
                "finished_at": last.finished_at,
            }
        return None


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