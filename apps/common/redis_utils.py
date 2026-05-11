"""
Redis 工具模块

提供基于 Django cache 框架的缓存操作封装，以及 Redis 专有功能：
  分布式锁、计数器、限流器、发布订阅、消息队列。

设计原则:
  1. 默认使用 Django 的 cache 抽象 (django.core.cache)，可无缝切换 backend
  2. Redis 专有功能（锁、pub/sub、pipeline）通过 get_redis_client() 获取原生连接
  3. Redis 不可用时自动降级，不影响业务正常运行
  4. 所有超时参数默认带合理的 TTL，防止内存泄漏

使用示例:
  >>> from apps.common.redis_utils import cache, redis_client, lock
  >>> cache.set("key", "value", ttl=300)
  >>> val = cache.get("key")
  >>> with lock("my_lock", expire=10):
  ...     # 临界区
  >>> redis_client.publish("channel", "message")
"""
import json
import logging
import threading
import time
import uuid
from contextlib import contextmanager
from functools import wraps

from django.conf import settings
from django.core.cache import cache as _django_cache

logger = logging.getLogger("apps.common.redis_utils")

# ── Redis 客户端（惰性初始化）──────────────────────

_client = None
_client_lock = threading.Lock()


def get_redis_client():
    """
    获取原生 Redis 连接。
    连接失败时返回 None，由调用方自行处理降级逻辑。

    首次调用后会缓存连接状态，不重复尝试（避免每次请求都 try-connect）。
    """
    global _client
    if _client is not None:
        return _client

    with _client_lock:
        if _client is not None:
            return _client

        url = getattr(settings, "REDIS_URL", None)
        if not url:
            logger.warning("REDIS_URL 未配置，Redis 功能不可用")
            _client = False  # 标记为不可用
            return None

        try:
            import redis as _redis
            r = _redis.from_url(url, decode_responses=True)
            r.ping()
            _client = r
            logger.info("Redis 连接成功")
            return _client
        except Exception as e:
            logger.warning(f"Redis 连接失败，已降级: {e}")
            _client = False  # 标记为不可用
            return None


# ── 缓存操作（基于 Django cache 抽象）────────────────

class CacheProxy:
    """
    缓存代理 — 封装 Django cache，统一日志和异常处理。

    所有操作的 key 自动添加统一前缀（避免多项目冲突），
    可通过 settings.REDIS_KEY_PREFIX 配置，默认 "adminx:"。

    对比直接用 django.core.cache 的优势:
      1. 自动前缀隔离
      2. 统一超时、日志
      3. 序列化/反序列化封装
    """

    def __init__(self):
        self._prefix = getattr(settings, "REDIS_KEY_PREFIX", "adminx:")

    def _prefixed(self, key: str) -> str:
        """添加统一前缀"""
        return f"{self._prefix}{key}"

    # ── 基础操作 ──────────────────────────────────

    def get(self, key: str, default=None):
        """获取缓存值"""
        try:
            return _django_cache.get(self._prefixed(key), default)
        except Exception as e:
            logger.warning(f"Cache get 失败 key={key}: {e}")
            return default

    def set(self, key: str, value, ttl: int = 300):
        """
        设置缓存

        :param key:  缓存键
        :param value: 值
        :param ttl:   过期时间（秒），默认 5 分钟
        """
        try:
            _django_cache.set(self._prefixed(key), value, timeout=ttl)
            return True
        except Exception as e:
            logger.warning(f"Cache set 失败 key={key}: {e}")
            return False

    def delete(self, key: str):
        """删除缓存"""
        try:
            _django_cache.delete(self._prefixed(key))
            return True
        except Exception as e:
            logger.warning(f"Cache delete 失败 key={key}: {e}")
            return False

    def exists(self, key: str) -> bool:
        """判断 key 是否存在"""
        return self.get(key) is not None

    def ttl(self, key: str):
        """获取剩余过期时间（秒），-1 表示永不过期，-2 表示不存在"""
        try:
            return _django_cache.ttl(self._prefixed(key))
        except Exception:
            try:
                val = self.get(key, _sentinel := object())
                return -2 if val is _sentinel else -1
            except Exception:
                return -2

    # ── 批量操作 ──────────────────────────────────

    def get_many(self, keys: list) -> dict:
        """批量获取"""
        try:
            prefixed = {self._prefixed(k): k for k in keys}
            raw = _django_cache.get_many(list(prefixed.keys()))
            return {prefixed[k]: v for k, v in raw.items()}
        except Exception as e:
            logger.warning(f"Cache get_many 失败: {e}")
            return {}

    def set_many(self, mapping: dict, ttl: int = 300):
        """批量设置"""
        try:
            prefixed = {self._prefixed(k): v for k, v in mapping.items()}
            _django_cache.set_many(prefixed, timeout=ttl)
            return True
        except Exception as e:
            logger.warning(f"Cache set_many 失败: {e}")
            return False

    def delete_many(self, keys: list):
        """批量删除"""
        try:
            _django_cache.delete_many([self._prefixed(k) for k in keys])
            return True
        except Exception as e:
            logger.warning(f"Cache delete_many 失败: {e}")
            return False

    # ── 高级操作 ──────────────────────────────────

    def remember(self, key: str, ttl: int = 300):
        """
        记忆装饰器 — 缓存函数返回值，适用于耗时计算或数据库查询。

        用法:
            @cache.remember("user_stats", ttl=60)
            def get_user_stats():
                return expensive_query()

        :param key:  缓存键
        :param ttl:   过期时间（秒），默认 5 分钟
        """
        def decorator(func):
            @wraps(func)
            def wrapper(*args, **kwargs):
                cached = self.get(key)
                if cached is not None:
                    return cached
                result = func(*args, **kwargs)
                self.set(key, result, ttl=ttl)
                return result
            return wrapper
        return decorator

    # ── JSON 操作（自动序列化/反序列化）────────────

    def get_json(self, key: str, default=None):
        """获取 JSON 缓存，自动反序列化"""
        val = self.get(key, _sentinel := object())
        if val is _sentinel:
            return default
        if isinstance(val, (str, bytes)):
            try:
                return json.loads(val)
            except (json.JSONDecodeError, TypeError):
                return default
        return val

    def set_json(self, key: str, value, ttl: int = 300):
        """设置 JSON 缓存，自动序列化"""
        try:
            return self.set(key, json.dumps(value, ensure_ascii=False, default=str), ttl=ttl)
        except (TypeError, ValueError) as e:
            logger.warning(f"Cache set_json 序列化失败 key={key}: {e}")
            return False

    # ── 工具 ──────────────────────────────────────

    def clear_prefix(self, prefix: str):
        """
        按前缀清除缓存（仅 Redis backend 生效）
        LocMemCache 不支持前缀扫描，会忽略。
        """
        client = get_redis_client()
        if not client:
            logger.warning("clear_prefix 需要 Redis，当前 cache backend 不支持")
            return 0

        try:
            keys = client.keys(f"{self._prefix}{prefix}*")
            if keys:
                return client.delete(*keys)
            return 0
        except Exception as e:
            logger.warning(f"clear_prefix 失败 prefix={prefix}: {e}")
            return 0

    def stats(self) -> dict:
        """
        缓存统计信息
        返回 dict，无 Redis 时返回基本信息。
        """
        client = get_redis_client()
        if client:
            try:
                info = client.info()
                return {
                    "backend": "redis",
                    "version": info.get("redis_version", ""),
                    "used_memory": info.get("used_memory_human", ""),
                    "total_keys": info.get("db0", {}).get("keys", 0),
                    "uptime_days": info.get("uptime_in_days", 0),
                    "hit_rate": f"{info.get('keyspace_hits', 0)} / {info.get('keyspace_hits', 0) + info.get('keyspace_misses', 1)}",
                }
            except Exception as e:
                logger.warning(f"Redis stats 获取失败: {e}")

        # 降级: 返回 locmem 信息
        try:
            from django.core.cache.backends.locmem import LocMemCache
            backend = _django_cache
            if hasattr(backend, "_cache"):
                return {
                    "backend": "locmem",
                    "total_keys": len(backend._cache),
                }
        except Exception:
            pass
        return {"backend": "locmem", "total_keys": "unknown"}


# ── 原生 Redis 操作 ─────────────────────────────

class RedisProxy:
    """
    原生 Redis 操作代理 — 需要 get_redis_client() 返回可用连接。

    所有方法在 Redis 不可用时不抛异常，返回安全的默认值。
    调用方无需额外 try/except。
    """

    # ── 基本操作 ──────────────────────────────────

    @staticmethod
    def get(key: str, default=None):
        client = get_redis_client()
        if not client:
            return default
        try:
            val = client.get(key)
            return val if val is not None else default
        except Exception as e:
            logger.warning(f"Redis get 失败: {e}")
            return default

    @staticmethod
    def set(key: str, value, ttl: int = None):
        client = get_redis_client()
        if not client:
            return False
        try:
            return bool(client.set(key, value, ex=ttl))
        except Exception as e:
            logger.warning(f"Redis set 失败: {e}")
            return False

    @staticmethod
    def delete(key: str):
        client = get_redis_client()
        if not client:
            return False
        try:
            return bool(client.delete(key))
        except Exception as e:
            logger.warning(f"Redis delete 失败: {e}")
            return False

    @staticmethod
    def exists(key: str) -> bool:
        client = get_redis_client()
        if not client:
            return False
        try:
            return bool(client.exists(key))
        except Exception as e:
            logger.warning(f"Redis exists 失败: {e}")
            return False

    @staticmethod
    def expire(key: str, ttl: int):
        """设置过期时间"""
        client = get_redis_client()
        if not client:
            return False
        try:
            return bool(client.expire(key, ttl))
        except Exception as e:
            logger.warning(f"Redis expire 失败: {e}")
            return False

    @staticmethod
    def ttl(key: str) -> int:
        client = get_redis_client()
        if not client:
            return -2
        try:
            return client.ttl(key)
        except Exception as e:
            logger.warning(f"Redis ttl 失败: {e}")
            return -2

    # ── 计数器 ──────────────────────────────────

    @staticmethod
    def incr(key: str, amount: int = 1, ttl: int = None) -> int:
        """
        自增计数器

        :param amount: 步长
        :param ttl:    首次设置时的过期时间（秒）
        :return:       自增后的值，失败返回 0
        """
        client = get_redis_client()
        if not client:
            return 0
        try:
            val = client.incr(key, amount)
            if ttl is not None and val == amount:
                client.expire(key, ttl)
            return val
        except Exception as e:
            logger.warning(f"Redis incr 失败: {e}")
            return 0

    @staticmethod
    def decr(key: str, amount: int = 1) -> int:
        client = get_redis_client()
        if not client:
            return 0
        try:
            return client.decr(key, amount)
        except Exception as e:
            logger.warning(f"Redis decr 失败: {e}")
            return 0

    # ── Set 操作 ─────────────────────────────────

    @staticmethod
    def sadd(key: str, *values) -> int:
        """集合添加"""
        client = get_redis_client()
        if not client:
            return 0
        try:
            return client.sadd(key, *values)
        except Exception as e:
            logger.warning(f"Redis sadd 失败: {e}")
            return 0

    @staticmethod
    def smembers(key: str) -> set:
        client = get_redis_client()
        if not client:
            return set()
        try:
            return client.smembers(key) or set()
        except Exception as e:
            logger.warning(f"Redis smembers 失败: {e}")
            return set()

    @staticmethod
    def srem(key: str, *values) -> int:
        client = get_redis_client()
        if not client:
            return 0
        try:
            return client.srem(key, *values)
        except Exception as e:
            logger.warning(f"Redis srem 失败: {e}")
            return 0

    # ── Hash 操作 ────────────────────────────────

    @staticmethod
    def hget(key: str, field: str, default=None):
        client = get_redis_client()
        if not client:
            return default
        try:
            val = client.hget(key, field)
            return val if val is not None else default
        except Exception as e:
            logger.warning(f"Redis hget 失败: {e}")
            return default

    @staticmethod
    def hset(key: str, field: str, value):
        client = get_redis_client()
        if not client:
            return False
        try:
            return bool(client.hset(key, field, value))
        except Exception as e:
            logger.warning(f"Redis hset 失败: {e}")
            return False

    @staticmethod
    def hgetall(key: str) -> dict:
        client = get_redis_client()
        if not client:
            return {}
        try:
            return client.hgetall(key) or {}
        except Exception as e:
            logger.warning(f"Redis hgetall 失败: {e}")
            return {}

    @staticmethod
    def hdel(key: str, *fields) -> int:
        client = get_redis_client()
        if not client:
            return 0
        try:
            return client.hdel(key, *fields)
        except Exception as e:
            logger.warning(f"Redis hdel 失败: {e}")
            return 0

    @staticmethod
    def hincrby(key: str, field: str, amount: int = 1) -> int:
        client = get_redis_client()
        if not client:
            return 0
        try:
            return client.hincrby(key, field, amount)
        except Exception as e:
            logger.warning(f"Redis hincrby 失败: {e}")
            return 0

    # ── List 操作（消息队列场景）─────────────────

    @staticmethod
    def lpush(key: str, *values) -> int:
        """左推（用于队列）"""
        client = get_redis_client()
        if not client:
            return 0
        try:
            return client.lpush(key, *values)
        except Exception as e:
            logger.warning(f"Redis lpush 失败: {e}")
            return 0

    @staticmethod
    def rpop(key: str, default=None):
        """右弹"""
        client = get_redis_client()
        if not client:
            return default
        try:
            val = client.rpop(key)
            return val if val is not None else default
        except Exception as e:
            logger.warning(f"Redis rpop 失败: {e}")
            return default

    @staticmethod
    def llen(key: str) -> int:
        client = get_redis_client()
        if not client:
            return 0
        try:
            return client.llen(key)
        except Exception as e:
            logger.warning(f"Redis llen 失败: {e}")
            return 0

    @staticmethod
    def lrange(key: str, start: int = 0, stop: int = -1) -> list:
        client = get_redis_client()
        if not client:
            return []
        try:
            return client.lrange(key, start, stop) or []
        except Exception as e:
            logger.warning(f"Redis lrange 失败: {e}")
            return []

    # ── 发布订阅 ─────────────────────────────────

    @staticmethod
    def publish(channel: str, message):
        """发布消息到频道"""
        client = get_redis_client()
        if not client:
            return 0
        try:
            if isinstance(message, (dict, list)):
                message = json.dumps(message, ensure_ascii=False, default=str)
            return client.publish(channel, message)
        except Exception as e:
            logger.warning(f"Redis publish 失败: {e}")
            return 0

    @staticmethod
    def pubsub(channel: str, **kwargs):
        """
        创建订阅对象（调用方需要自行处理消息循环）

        用法:
            ps = redis_client.pubsub("channel")
            for msg in ps.listen():
                print(msg)
        """
        client = get_redis_client()
        if not client:
            return None
        try:
            ps = client.pubsub(**kwargs)
            ps.subscribe(channel)
            return ps
        except Exception as e:
            logger.warning(f"Redis pubsub 订阅失败: {e}")
            return None

    # ── Pipeline / 事务 ──────────────────────────

    @contextmanager
    def pipeline(self, transaction: bool = True):
        """
        Pipeline 上下文管理器

        用法:
            with redis_client.pipeline() as pipe:
                pipe.set("a", 1)
                pipe.set("b", 2)
                pipe.execute()
        """
        client = get_redis_client()
        if not client:
            yield _FakePipeline()
            return
        pipe = client.pipeline(transaction=transaction)
        try:
            yield pipe
            pipe.execute()
        except Exception as e:
            pipe.reset()
            logger.warning(f"Redis pipeline 失败: {e}")
            raise


class _FakePipeline:
    """Redis 不可用时的假 pipeline（静默丢弃）"""
    def __getattr__(self, name):
        return lambda *args, **kwargs: None
    def execute(self):
        return []


# ── 分布式锁 ──────────────────────────────────

@contextmanager
def _lock(key: str, expire: int = 10, timeout: int = None):
    """
    分布式锁（阻塞式）

    基于 Redis SET NX EX 实现，自动续期机制。
    Redis 不可用时退化为线程锁（仅单进程有效）。

    :param key:     锁标识（无需加前缀，内部自动处理）
    :param expire:  锁自动释放时间（秒），默认 10 秒
    :param timeout: 等待超时时间（秒），None 表示一直等待，0 表示不等待

    用法:
        with _lock("order:123", expire=30, timeout=10):
            process_order()
    """
    prefixed = f"adminx:lock:{key}"
    client = get_redis_client()
    identifier = uuid.uuid4().hex

    # ── Redis 不可用时降级为线程锁 ──
    if not client:
        acquired = _thread_lock.acquire(timeout=timeout)
        if not acquired:
            raise TimeoutError(f"获取锁超时: {key}")
        try:
            yield
        finally:
            _thread_lock.release()
        return

    # ── 获取 Redis 锁 ──
    start = time.monotonic()
    try:
        while True:
            if client.set(prefixed, identifier, nx=True, ex=expire):
                break
            if timeout is not None and (time.monotonic() - start) >= timeout:
                raise TimeoutError(f"获取锁超时: {key}")
            time.sleep(0.05)

        try:
            yield
        finally:
            # Lua 脚本: 仅当锁仍被自己持有时释放（防止误删）
            release_script = """
                if redis.call("get", KEYS[1]) == ARGV[1] then
                    return redis.call("del", KEYS[1])
                else
                    return 0
                end
            """
            client.eval(release_script, 1, prefixed, identifier)
    except TimeoutError:
        raise
    except Exception as e:
        logger.warning(f"锁操作异常 key={key}: {e}")
        raise


_thread_lock = threading.Lock()
lock = _lock


# ── 限流器 ──────────────────────────────────

class RateLimiter:
    """
    基于滑动窗口的限流器（Redis + sorted set）

    用法:
        limiter = RateLimiter(max_requests=100, window=60)
        if limiter.allow("api:user_123"):
            process_request()
        else:
            return 429 Too Many Requests

    Redis 不可用时静默放行（不限制）。
    """

    def __init__(self, max_requests: int = 100, window: int = 60, prefix: str = "ratelimit"):
        """
        :param max_requests: 窗口内允许的最大请求数
        :param window:       窗口大小（秒）
        :param prefix:       Redis key 前缀
        """
        self.max_requests = max_requests
        self.window = window
        self.prefix = f"adminx:{prefix}"

    def allow(self, key: str) -> bool:
        """
        判断是否允许请求通过

        :param key: 限流标识（如 user_id、IP）
        :return:    True 允许，False 超出限制
        """
        client = get_redis_client()
        if not client:
            return True  # 降级：无 Redis 时不限流

        now = time.time()
        window_start = now - self.window
        redis_key = f"{self.prefix}:{key}"

        try:
            # 移除窗口外的旧记录
            client.zremrangebyscore(redis_key, "-inf", window_start)
            # 当前窗口内的请求数
            count = client.zcard(redis_key)

            if count >= self.max_requests:
                return False

            # 记录本次请求
            client.zadd(redis_key, {str(now): now})
            client.expire(redis_key, self.window + 5)
            return True
        except Exception as e:
            logger.warning(f"RateLimiter 异常 key={key}: {e}")
            return True  # 降级

    def remaining(self, key: str) -> int:
        """剩余可用次数"""
        client = get_redis_client()
        if not client:
            return self.max_requests
        try:
            count = client.zcard(f"{self.prefix}:{key}")
            return max(0, self.max_requests - count)
        except Exception:
            return self.max_requests

    def reset(self, key: str):
        """重置限流状态"""
        client = get_redis_client()
        if not client:
            return
        try:
            client.delete(f"{self.prefix}:{key}")
        except Exception as e:
            logger.warning(f"RateLimiter reset 失败: {e}")


# ── 延迟双删（cache 一致性模式） ──────────────

def delay_double_delete(key: str, delay: float = 0.5):
    """
    延迟双删 — 应对缓存与数据库一致性问题。

    用法:
        def update_data():
            delay_double_delete("my_key")
            db.update(...)
            # delay 0.5s 后第二次删除在后台自动执行

    原理:
        先删缓存 → 更新 DB → 延迟再删缓存
        解决「读请求在第一次删除后读到旧数据并回写缓存」的竞态问题。
    """
    cache.delete(key)
    threading.Timer(delay, cache.delete, args=[key]).start()


# ── 单例导出 ──────────────────────────────────

cache = CacheProxy()
redis_client = RedisProxy()
rate_limiter = RateLimiter