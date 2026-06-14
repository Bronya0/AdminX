"""
APScheduler 管理 — 通过独立进程运行 (python manage.py run_scheduler)
"""
import json
import logging
import uuid
from apscheduler.executors.pool import ThreadPoolExecutor
from apscheduler.jobstores.base import JobLookupError
from apscheduler.schedulers.background import BackgroundScheduler
from apscheduler.triggers.cron import CronTrigger
from apscheduler.triggers.interval import IntervalTrigger

from django.utils import timezone

from .scheduler_leadership import SchedulerLeader

logger = logging.getLogger("djangoadminx.scheduler")

SCHEDULER_HEARTBEAT_TTL = 30
HEARTBEAT_ID = uuid.UUID("00000000-0000-0000-0000-000000000001")


def _get_jobstore(redis_url=None):
    if redis_url:
        try:
            from apscheduler.jobstores.redis import RedisJobStore
            import redis as redis_module
            parsed = redis_module.from_url(redis_url)
            pool = parsed.connection_pool
            return RedisJobStore(
                jobs_key="scheduler:jobs",
                run_times_key="scheduler:run_times",
                host=pool.connection_kwargs.get("host", "localhost"),
                port=pool.connection_kwargs.get("port", 6379),
                password=pool.connection_kwargs.get("password", None),
                db=pool.connection_kwargs.get("db", 0),
            ), "Redis"
        except Exception as e:
            logger.warning(f"RedisJobStore 不可用，降级为 Memory: {e}")

    from apscheduler.jobstores.memory import MemoryJobStore
    return MemoryJobStore(), "Memory"


class SchedulerManager:
    """调度器管理器"""

    def __init__(self):
        self._scheduler = None
        self._redis_url = None
        self.leader = SchedulerLeader()

    def _ensure_scheduler(self):
        if self._scheduler is None:
            from django.conf import settings

            redis_url = getattr(settings, "REDIS_URL", None)
            if redis_url:
                try:
                    import redis as redis_module
                    r = redis_module.from_url(redis_url)
                    r.ping()
                    r.connection_pool.disconnect()
                except Exception:
                    redis_url = None

            jobstore, store_name = _get_jobstore(redis_url)
            executors = {"default": ThreadPoolExecutor(10)}

            self._scheduler = BackgroundScheduler(
                jobstores={"default": jobstore},
                executors=executors,
                timezone="Asia/Shanghai",
            )
            logger.info(f"Scheduler 使用 {store_name}JobStore")

    @property
    def scheduler(self):
        self._ensure_scheduler()
        return self._scheduler

    @property
    def running(self):
        return self._scheduler is not None and self._scheduler.running

    def start(self):
        """初始化 scheduler 对象（不加载 job、不启动）。

        job 的加载与 APScheduler.start() 由 try_become_leader() 触发，
        保证只有 leader 才真正执行任务。本方法幂等，可安全多次调用。
        """
        self._ensure_scheduler()

    # ── Leader 门控 ──

    def try_become_leader(self):
        """尝试抢占 leader。成功则加载 job + 启动调度器 + 写心跳。"""
        if not self.leader.try_acquire():
            return False  # standby
        # 抢到锁（或单机模式），真正启动
        self._ensure_scheduler()
        if not self._scheduler.running:
            self._load_jobs_from_db()
            self._scheduler.start()
            logger.info("APScheduler started (leader)")
        self.write_heartbeat()
        return True

    def step_down(self):
        """leader 锁丢失时调用：停掉调度器，但进程继续，等待重新抢占。"""
        if self._scheduler and self._scheduler.running:
            self._scheduler.shutdown(wait=False)
            logger.warning("APScheduler 已停止（leader 锁丢失，转 standby）")
            # shutdown 后 _scheduler 不可复用，置 None 让 _ensure_scheduler 重建
            self._scheduler = None
        self.leader._is_leader = False

    def shutdown(self):
        if self._scheduler and self._scheduler.running:
            self._scheduler.shutdown(wait=False)
            logger.info("APScheduler shutdown")
        self.leader.release()
        self.clear_heartbeat()

    # ── 调度器进程心跳 ──

    @staticmethod
    def _get_heartbeat_model():
        from djangoadminx.jobs.models import SchedulerHeartbeat
        return SchedulerHeartbeat

    @staticmethod
    def write_heartbeat():
        try:
            SchedulerHeartbeat = SchedulerManager._get_heartbeat_model()
            SchedulerHeartbeat.objects.update_or_create(
                id=HEARTBEAT_ID,
                defaults={"last_heartbeat": timezone.now()},
            )
        except Exception as e:
            logger.warning(f"写入心跳失败: {e}")

    @staticmethod
    def clear_heartbeat():
        try:
            SchedulerHeartbeat = SchedulerManager._get_heartbeat_model()
            SchedulerHeartbeat.objects.filter(id=HEARTBEAT_ID).delete()
        except Exception as e:
            logger.warning(f"清除心跳失败: {e}")

    @staticmethod
    def is_alive():
        try:
            SchedulerHeartbeat = SchedulerManager._get_heartbeat_model()
            hb = SchedulerHeartbeat.objects.values("last_heartbeat").filter(id=HEARTBEAT_ID).first()
            if hb and hb["last_heartbeat"]:
                elapsed = (timezone.now() - hb["last_heartbeat"]).total_seconds()
                return elapsed < SCHEDULER_HEARTBEAT_TTL
            return False
        except Exception as e:
            logger.warning(f"检查心跳失败: {e}")
            return False

    # ── 跨进程通知 ──

    @staticmethod
    def notify_reload():
        try:
            SchedulerHeartbeat = SchedulerManager._get_heartbeat_model()
            SchedulerHeartbeat.objects.update_or_create(
                id=HEARTBEAT_ID,
                defaults={"reload_pending": True},
            )
        except Exception as e:
            logger.warning(f"通知重载失败: {e}")

    def process_notifications(self):
        try:
            if self._scheduler is None or not self._scheduler.running:
                return
            SchedulerHeartbeat = self._get_heartbeat_model()
            updated = SchedulerHeartbeat.objects.filter(
                id=HEARTBEAT_ID, reload_pending=True
            ).update(reload_pending=False)
            if updated:
                logger.info("收到全量重载通知，执行重载")
                self.reload_all()
        except Exception as e:
            logger.warning(f"处理通知失败: {e}")

    def _load_jobs_from_db(self):
        from djangoadminx.jobs.models import ScheduleJob

        count = 0
        for job in ScheduleJob.objects.filter(is_active=True):
            if self._add_job_to_scheduler(job):
                count += 1
        logger.info(f"从数据库加载 {count} 个定时任务")

    def _add_job_to_scheduler(self, job):
        try:
            trigger = self._build_trigger(job)
            if trigger is None:
                logger.warning(f"Job {job.name} 触发配置无效")
                return False

            self.scheduler.add_job(
                func=_execute_job_wrapper,
                trigger=trigger,
                id=str(job.id),
                name=job.name,
                args=[str(job.id)],
                replace_existing=True,
                misfire_grace_time=60,
            )
            return True
        except Exception as e:
            logger.error(f"加载 Job {job.name} 失败: {e}")
            return False

    def _build_trigger(self, job):
        config = json.loads(job.trigger_config) if job.trigger_config else {}
        if job.trigger_type == "cron":
            return CronTrigger(**config)
        elif job.trigger_type == "interval":
            return IntervalTrigger(**config)
        elif job.trigger_type == "date":
            from apscheduler.triggers.date import DateTrigger
            return DateTrigger(**config)
        return None

    def reload_all(self):
        self.scheduler.remove_all_jobs()
        self._load_jobs_from_db()


def _execute_job_wrapper(job_id):
    from djangoadminx.jobs.models import JobLog, ScheduleJob

    try:
        job = ScheduleJob.objects.get(id=job_id, is_active=True)
        started_at = timezone.now()
        log = JobLog.objects.create(job=job, status="running", started_at=started_at)

        try:
            result = job.execute()
            log.status = "success"
            log.result = str(result)[:500] if result is not None else "ok"
        except Exception as e:
            log.status = "failed"
            log.result = str(e)[:500]
            logger.exception(f"Job {job.name} 执行失败")
        log.finished_at = timezone.now()
        log.save()
    except ScheduleJob.DoesNotExist:
        logger.warning(f"Job {job_id} 不存在或已禁用")


scheduler_manager = SchedulerManager()
