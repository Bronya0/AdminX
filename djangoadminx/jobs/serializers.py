from rest_framework import serializers

from .models import JobLog, ScheduleJob


class ScheduleJobSerializer(serializers.ModelSerializer):
    last_run = serializers.SerializerMethodField()

    class Meta:
        model = ScheduleJob
        fields = ["id", "name", "command_type", "handler", "command",
                   "trigger_type", "trigger_config", "args", "kwargs",
                   "is_active", "last_run", "created_at", "updated_at"]
        read_only_fields = ["id", "created_at", "updated_at"]

    def validate(self, attrs):
        if attrs.get("command_type") == "python" and not attrs.get("handler"):
            raise serializers.ValidationError({"handler": "Python 类型任务必须填写处理函数"})
        if attrs.get("command_type") == "shell" and not attrs.get("command"):
            raise serializers.ValidationError({"command": "Shell 类型任务必须填写命令"})
        return attrs

    @staticmethod
    def get_last_run(obj):
        last = JobLog.objects.filter(job=obj).order_by("-started_at").first()
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
