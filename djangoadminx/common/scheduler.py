"""
APScheduler 管理 — 通过独立进程运行 (python manage.py run_scheduler)

设计原则:
  - 独立进程运行，不受 Gunicorn 多进程影响
  - Job 定义存储在数据库，CRUD 后自动重载调度
  - 有 Redis 时用 RedisJobStore，无 Redis 时用 SQLite 兜底
  - 首次通过懒初始化，避免 Django 启动时连接 Redis
  - 通过数据库心跳检测调度器进程存活状态（不依赖 Redis）
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

logger = logging.getLogger("djangoadminx.scheduler")

# 心跳超时阈值（秒）：超过此时间未收到心跳视为调度器已停止
SCHEDULER_HEARTBEAT_TTL = 30
# 心跳记录固定 UUID（用于 update_or_create 单行记录）
HEARTBEAT_ID = uuid.UUID("00000000-0000-0000-0000-000000000001")


def _get_jobstore(redis_url=None):
    """根据 Redis 可用性选择 JobStore"""
    if redis_url:
        try:
            from apscheduler.jobstores.redis import RedisJobStore
            return RedisJobStore(
                jobs_key="scheduler:jobs",
                run_times_key="scheduler:run_times",
                host=redis_url,
            ), "Redis"
        except Exception as e:
            logger.warning(f"RedisJobStore 不可用，降级为 SQLite: {e}")

    from apscheduler.jobstores.memory import MemoryJobStore
    return MemoryJobStore(), "Memory"  # 重启后 job 会从数据库重新加载


class SchedulerManager:
    """调度器管理器 — 封装 APScheduler 核心操作"""

    def __init__(self):
        self._scheduler = None
        self._redis_url = None

    def _ensure_scheduler(self):
        if self._scheduler is None:
            from django.conf import settings

            redis_url = getattr(settings, "REDIS_URL", None)
            # 检测 Redis 是否可用
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
        """启动调度器"""
        self._ensure_scheduler()
        if self._scheduler.running:
            return
        self._load_jobs_from_db()
        self._scheduler.start()
        logger.info("APScheduler started")

    def shutdown(self):
        """关闭调度器"""
        if self._scheduler and self._scheduler.running:
            self._scheduler.shutdown(wait=False)
            logger.info("APScheduler shutdown")
        self.clear_heartbeat()

    # ── 调度器进程心跳（数据库） ─────────────────────

    @staticmethod
    def _get_heartbeat_model():
        """延迟导入避免循环依赖"""
        from djangoadminx.webservice.models import SchedulerHeartbeat
        return SchedulerHeartbeat

    @staticmethod
    def write_heartbeat():
        """调度器进程写入心跳（供 web 进程检测存活）"""
        try:
            SchedulerHeartbeat = SchedulerManager._get_heartbeat_model()
            SchedulerHeartbeat.objects.update_or_create(
                id=HEARTBEAT_ID,
                defaults={"last_heartbeat": timezone.now()},
            )
        except Exception:
            pass

    @staticmethod
    def clear_heartbeat():
        """调度器进程清除心跳记录"""
        try:
            SchedulerHeartbeat = SchedulerManager._get_heartbeat_model()
            SchedulerHeartbeat.objects.filter(id=HEARTBEAT_ID).delete()
        except Exception:
            pass

    @staticmethod
    def is_alive():
        """检查调度器进程是否存活（通过数据库心跳）"""
        try:
            SchedulerHeartbeat = SchedulerManager._get_heartbeat_model()
            hb = SchedulerHeartbeat.objects.values("last_heartbeat").filter(id=HEARTBEAT_ID).first()
            if hb and hb["last_heartbeat"]:
                elapsed = (timezone.now() - hb["last_heartbeat"]).total_seconds()
                return elapsed < SCHEDULER_HEARTBEAT_TTL
            return False
        except Exception:
            return False

    # ── 跨进程通知（数据库） ─────────────────────

    @staticmethod
    def notify_reload():
        """通知调度器进程需要全量重载（web 进程 → 调度器进程）"""
        try:
            SchedulerHeartbeat = SchedulerManager._get_heartbeat_model()
            SchedulerHeartbeat.objects.update_or_create(
                id=HEARTBEAT_ID,
                defaults={"reload_pending": True},
            )
        except Exception:
            pass

    def process_notifications(self):
        """处理数据库通知（调度器进程主循环调用）"""
        try:
            if self._scheduler is None or not self._scheduler.running:
                return
            SchedulerHeartbeat = self._get_heartbeat_model()
            hb = SchedulerHeartbeat.objects.filter(id=HEARTBEAT_ID, reload_pending=True).first()
            if hb:
                logger.info("收到全量重载通知，执行重载")
                self.reload_all()
                SchedulerHeartbeat.objects.filter(id=HEARTBEAT_ID).update(reload_pending=False)
        except Exception:
            pass

    def _load_jobs_from_db(self):
        """从数据库加载定时任务"""
        from djangoadminx.webservice.models import ScheduleJob

        count = 0
        for job in ScheduleJob.objects.filter(is_active=True):
            if self._add_job_to_scheduler(job):
                count += 1
        logger.info(f"从数据库加载 {count} 个定时任务")

    def _add_job_to_scheduler(self, job):
        """添加单个 Job 到调度器"""
        try:
            trigger = self._build_trigger(job)
            if trigger is None:
                logger.warning(f"Job {job.name} 触发配置无效")
                return False

            self.scheduler.add_job(
                func=self._execute_job_wrapper,
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

    def _execute_job_wrapper(self, job_id):
        """执行 Job — 被 APScheduler 调用"""
        from djangoadminx.webservice.models import JobLog, ScheduleJob

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

    def reload_job(self, job_id):
        """重载单个 Job"""
        try:
            self.scheduler.remove_job(str(job_id))
        except JobLookupError:
            pass

        from djangoadminx.webservice.models import ScheduleJob

        try:
            job = ScheduleJob.objects.get(id=job_id, is_active=True)
            self._add_job_to_scheduler(job)
        except ScheduleJob.DoesNotExist:
            pass

    def reload_all(self):
        """全量重载"""
        self.scheduler.remove_all_jobs()
        self._load_jobs_from_db()


# 全局单例 — 调用时懒初始化
scheduler_manager = SchedulerManager()