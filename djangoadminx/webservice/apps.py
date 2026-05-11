import logging

from django.apps import AppConfig
from django.db.models.signals import post_delete, post_save

logger = logging.getLogger("djangoadminx.webservice")


class WebserviceConfig(AppConfig):
    name = "djangoadminx.webservice"
    verbose_name = "WebService 管理"

    def ready(self):
        """注册信号 — 定时任务变更时通知 APScheduler"""
        from .models import ScheduleJob

        def reload_job_handler(sender, instance, **kwargs):
            try:
                from djangoadminx.common.scheduler import scheduler_manager
                job_id = str(getattr(instance, "id", instance.pk))
                scheduler_manager.reload_job(job_id)
            except Exception as e:
                logger.warning(f"Scheduler reload failed: {e}")

        post_save.connect(reload_job_handler, sender=ScheduleJob, weak=False)
        post_delete.connect(reload_job_handler, sender=ScheduleJob, weak=False)
        logger.debug("Scheduler signals registered")