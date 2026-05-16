import uuid
import json
import logging

from django.db import models
from django.utils import timezone

logger = logging.getLogger("djangoadminx.webservice")


class ScheduleJob(models.Model):
    """定时任务 — 由 APScheduler 执行"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    name = models.CharField("任务名称", max_length=128)
    command_type = models.CharField("命令类型", max_length=20,
                                    choices=[("python", "Python 函数"), ("shell", "Shell 命令")],
                                    default="python")
    handler = models.CharField("处理函数", max_length=255, blank=True, default="",
                               help_text="如 djangoadminx.webservice.tasks.ntp_sync")
    command = models.TextField("Shell 命令", blank=True, default="",
                               help_text="command_type=shell 时，要执行的命令或脚本")
    trigger_type = models.CharField("触发类型", max_length=20,
                                    choices=[("cron", "Cron"), ("interval", "间隔"), ("date", "指定时间")],
                                    default="interval")
    trigger_config = models.TextField("触发配置", blank=True, default="",
                                      help_text="JSON 格式，如 {\"minutes\": 5}")
    args = models.TextField("参数", blank=True, default="", help_text="JSON 数组")
    kwargs = models.TextField("关键字参数", blank=True, default="{}", help_text="JSON 对象")
    is_active = models.BooleanField("启用", default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "定时任务"
        verbose_name_plural = "定时任务"
        ordering = ["name"]

    def __str__(self):
        return self.name

    def execute(self):
        """执行任务 — Python 函数或 Shell 命令"""
        import importlib
        import subprocess

        if self.command_type == "shell":
            if not self.command:
                raise ValueError("Shell 命令为空")
            logger.warning(f"执行 shell 命令: {self.command[:200]}")
            result = subprocess.run(
                self.command,
                shell=True,
                capture_output=True,
                text=True,
                timeout=3600,
            )
            output = ""
            if result.stdout:
                output += f"[stdout]\n{result.stdout}"
            if result.stderr:
                output += f"[stderr]\n{result.stderr}"
            if result.returncode != 0:
                raise RuntimeError(f"命令退出码 {result.returncode}\n{output}")
            return output.strip() or f"完成 (exit=0)"

        # Python 函数
        if "." not in self.handler:
            raise ValueError(f"无效处理函数路径，应为 module.func: {self.handler}")
        module_path, func_name = self.handler.rsplit(".", 1)
        module = importlib.import_module(module_path)
        func = getattr(module, func_name)
        args = json.loads(self.args) if self.args else []
        kwargs = json.loads(self.kwargs) if self.kwargs else {}
        return func(*args, **kwargs)


class JobLog(models.Model):
    """任务执行日志"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    job = models.ForeignKey(ScheduleJob, on_delete=models.CASCADE, verbose_name="任务")
    status = models.CharField("状态", max_length=20,
                              choices=[("running", "运行中"), ("success", "成功"), ("failed", "失败")],
                              default="running")
    result = models.TextField("结果", blank=True, default="")
    started_at = models.DateTimeField("开始时间", null=True, blank=True)
    finished_at = models.DateTimeField("结束时间", null=True, blank=True)

    class Meta:
        verbose_name = "任务日志"
        verbose_name_plural = "任务日志"
        ordering = ["-started_at"]


class SchedulerHeartbeat(models.Model):
    """调度器进程心跳 — 跨进程检测调度器存活与通知重载"""

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    last_heartbeat = models.DateTimeField("最后心跳", null=True, blank=True)
    reload_pending = models.BooleanField("待重载", default=False)

    class Meta:
        verbose_name = "调度器心跳"
        verbose_name_plural = "调度器心跳"
