"""调度器 Leader 选举 — Redis 分布式锁实现 Active-Standby HA。

设计要点:
- Redis 锁是"执行门控"：只有抢到锁的实例真正跑 APScheduler，其余 standby。
- DB 心跳（SchedulerHeartbeat 表）是"存活观测"：UI/API 据此判断调度器是否在线。
  两者职责分离，互不依赖 —— DB 心跳相关测试无需 Redis 即可运行。
- 无 Redis 时降级为"单机 leader"（永远当 leader），与改造前的单进程行为一致，
  保证本地开发无 Redis 也能正常跑调度器，不报错。

锁语义:
- SET key instance_id NX EX TTL   —— 原子抢锁
- Lua 脚本（check owner + expire）—— 原子续租，防止误续别人的锁
- Lua 脚本（check owner + del）  —— 原子释放，防止误删别人的锁
"""
import logging
import uuid

logger = logging.getLogger("djangoadminx.scheduler")

LOCK_KEY = "scheduler:leader"
LOCK_TTL = 30  # 锁过期秒数，与 SCHEDULER_HEARTBEAT_TTL 保持一致

# Lua: 仅当锁 owner == ARGV[1] 时才续期，返回 1 成功 / 0 失败
_RENEW_SCRIPT = """
if redis.call('get', KEYS[1]) == ARGV[1] then
    return redis.call('expire', KEYS[1], ARGV[2])
else
    return 0
end
"""

# Lua: 仅当锁 owner == ARGV[1] 时才删除
_RELEASE_SCRIPT = """
if redis.call('get', KEYS[1]) == ARGV[1] then
    return redis.call('del', KEYS[1])
else
    return 0
end
"""


class SchedulerLeader:
    """Redis 分布式锁 leader 选举。

    线程安全说明：调度器进程内只有一个 SchedulerLeader 实例，由主循环串行调用，
    无需加锁。
    """

    def __init__(self):
        # 实例唯一标识 —— 区分哪个调度器进程持有锁
        self._instance_id = uuid.uuid4().hex
        self._redis = None          # 懒加载，None = 尚未尝试连接
        self._redis_unavailable = False  # 已确认连不上，不再重试
        self._is_leader = False

    # ── Redis 客户端 ──

    def _get_redis(self):
        """返回可用的 redis 客户端；无 Redis 返回 None。

        连接失败会缓存"不可用"状态，避免每个 tick 都重试连接拖慢主循环。
        """
        if self._redis is not None:
            return self._redis
        if self._redis_unavailable:
            return None

        from django.conf import settings
        redis_url = getattr(settings, "REDIS_URL", None)
        if not redis_url:
            self._redis_unavailable = True
            return None

        try:
            import redis as redis_module
            client = redis_module.from_url(redis_url)
            client.ping()
            self._redis = client
            return client
        except Exception as e:
            self._redis_unavailable = True
            logger.warning(
                f"Redis 不可用，调度器降级为单机模式（无 HA，多实例会重复执行任务）: {e}"
            )
            return None

    # ── 选举 ──

    def try_acquire(self):
        """尝试成为 leader。

        Returns:
            True  —— 当前实例现在是 leader（新抢到 或 单机模式）
            False —— 别人持有锁，当前实例为 standby
        """
        r = self._get_redis()
        if r is None:
            # 单机模式：永远当 leader，保持改造前行为
            if not self._is_leader:
                logger.info("单机模式：本实例成为调度器 leader（无 HA）")
            self._is_leader = True
            return True

        acquired = r.set(LOCK_KEY, self._instance_id, nx=True, ex=LOCK_TTL)
        if acquired:
            self._is_leader = True
            logger.info(f"成为调度器 leader (instance={self._instance_id[:8]})")
            return True
        return False

    def renew(self):
        """续租（仅 leader 调用）。

        Returns:
            True  —— 续租成功，仍是 leader
            False —— 锁已丢失（被抢占或过期），已降级为 follower
        """
        if not self._is_leader:
            return False
        r = self._get_redis()
        if r is None:
            return True  # 单机模式永远 leader

        try:
            result = r.eval(_RENEW_SCRIPT, 1, LOCK_KEY, self._instance_id, LOCK_TTL)
        except Exception as e:
            # Redis 抖动 —— 视为锁可能丢失，保守降级，下个 tick 重新抢占
            logger.warning(f"续租失败，降级为 follower 等待重新抢占: {e}")
            self._is_leader = False
            return False

        if not result:
            self._is_leader = False
            logger.warning("调度器 leader 锁丢失（被抢占或过期），降级为 follower")
            return False
        return True

    def release(self):
        """主动释放锁 —— 优雅退出时调用，让 standby 立即接管。"""
        if not self._is_leader:
            return
        r = self._get_redis()
        if r is None:
            self._is_leader = False
            return
        try:
            r.eval(_RELEASE_SCRIPT, 1, LOCK_KEY, self._instance_id)
        except Exception as e:
            logger.warning(f"释放 leader 锁失败（锁会自动过期）: {e}")
        self._is_leader = False

    @property
    def is_leader(self):
        return self._is_leader

    @property
    def instance_id(self):
        return self._instance_id
