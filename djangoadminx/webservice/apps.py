import logging

from django.apps import AppConfig
from django.db.models.signals import post_delete, post_save

logger = logging.getLogger("djangoadminx.webservice")


class WebserviceConfig(AppConfig):
    name = "djangoadminx.webservice"
    verbose_name = "WebService 管理"

    def ready(self):
        """注册信号 — 定时任务变更时通知调度器进程重载"""
        from .models import ScheduleJob

        def notify_scheduler(sender, instance, **kwargs):
            try:
                from djangoadminx.common.scheduler import scheduler_manager
                scheduler_manager.notify_reload()
            except Exception as e:
                logger.warning(f"通知调度器失败: {e}")

        post_save.connect(notify_scheduler, sender=ScheduleJob, weak=False)
        post_delete.connect(notify_scheduler, sender=ScheduleJob, weak=False)
        logger.debug("Scheduler notification signals registered")