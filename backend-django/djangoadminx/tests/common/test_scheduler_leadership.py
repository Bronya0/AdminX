"""common — scheduler leadership (Redis 锁 leader 选举)

测试策略:
- SchedulerLeader 的 Redis 行为用 unittest.mock 模拟 redis client，覆盖:
  抢锁成功/失败、续租成功/锁丢失、释放、Redis 异常降级。
- 无 Redis 时降级为单机 leader（保持改造前行为）。
- 不依赖真实 Redis，CI/本地均可跑。
- 继承 SimpleTestCase（无 DB），使其被 Django test runner 发现。
"""
from unittest import mock

from django.test import SimpleTestCase

from djangoadminx.common.scheduler_leadership import (
    SchedulerLeader,
    LOCK_KEY,
    LOCK_TTL,
)


class TestSchedulerLeaderNoRedis(SimpleTestCase):
    """无 Redis 时降级为单机 leader。"""

    def test_acquire_without_redis_becomes_leader(self):
        leader = SchedulerLeader()
        with mock.patch.object(leader, "_get_redis", return_value=None):
            self.assertIs(leader.try_acquire(), True)
            self.assertIs(leader.is_leader, True)

    def test_renew_without_redis_stays_leader(self):
        leader = SchedulerLeader()
        with mock.patch.object(leader, "_get_redis", return_value=None):
            leader.try_acquire()
            self.assertIs(leader.renew(), True)
            self.assertIs(leader.is_leader, True)

    def test_release_without_redis_clears_leader(self):
        leader = SchedulerLeader()
        with mock.patch.object(leader, "_get_redis", return_value=None):
            leader.try_acquire()
            leader.release()
            self.assertIs(leader.is_leader, False)


class TestSchedulerLeaderWithRedis(SimpleTestCase):
    """有 Redis 时走分布式锁逻辑。"""

    def _make_leader_with_redis(self, redis_client):
        """构造一个绑定 mock redis 的 leader。"""
        leader = SchedulerLeader()
        leader._redis = redis_client
        # 标记为"已尝试连接"，避免 _get_redis 再次发起真实连接
        leader._redis_unavailable = False
        return leader

    def test_acquire_success_when_lock_free(self):
        redis_client = mock.MagicMock()
        redis_client.set.return_value = True  # SET NX 成功
        leader = self._make_leader_with_redis(redis_client)

        self.assertIs(leader.try_acquire(), True)
        self.assertIs(leader.is_leader, True)
        redis_client.set.assert_called_once_with(
            LOCK_KEY, leader.instance_id, nx=True, ex=LOCK_TTL
        )

    def test_acquire_fail_when_lock_held(self):
        redis_client = mock.MagicMock()
        redis_client.set.return_value = False  # 锁已被别人持有
        leader = self._make_leader_with_redis(redis_client)

        self.assertIs(leader.try_acquire(), False)
        self.assertIs(leader.is_leader, False)

    def test_renew_success_when_still_owner(self):
        redis_client = mock.MagicMock()
        redis_client.set.return_value = True
        redis_client.eval.return_value = 1  # 续租成功
        leader = self._make_leader_with_redis(redis_client)
        leader.try_acquire()

        self.assertIs(leader.renew(), True)
        self.assertIs(leader.is_leader, True)
        redis_client.eval.assert_called_once()

    def test_renew_fail_when_lock_lost(self):
        """锁被别人抢占或过期 —— 续租返回 0，降级为 follower。"""
        redis_client = mock.MagicMock()
        redis_client.set.return_value = True
        redis_client.eval.return_value = 0  # owner 不匹配
        leader = self._make_leader_with_redis(redis_client)
        leader.try_acquire()

        self.assertIs(leader.renew(), False)
        self.assertIs(leader.is_leader, False)

    def test_renew_returns_false_when_not_leader(self):
        leader = SchedulerLeader()
        # 从未 acquire，not leader
        self.assertIs(leader.renew(), False)

    def test_renew_on_redis_exception_steps_down(self):
        """Redis 抖动抛异常 —— 保守降级，下个 tick 重新抢占。"""
        redis_client = mock.MagicMock()
        redis_client.set.return_value = True
        redis_client.eval.side_effect = Exception("connection reset")
        leader = self._make_leader_with_redis(redis_client)
        leader.try_acquire()

        self.assertIs(leader.renew(), False)
        self.assertIs(leader.is_leader, False)

    def test_release_when_owner(self):
        redis_client = mock.MagicMock()
        redis_client.set.return_value = True
        redis_client.eval.return_value = 1  # del 成功
        leader = self._make_leader_with_redis(redis_client)
        leader.try_acquire()

        leader.release()
        self.assertIs(leader.is_leader, False)
        redis_client.eval.assert_called_once()

    def test_release_when_not_owner_is_noop(self):
        """释放时发现锁已不属于自己 —— 不删别人的锁，仅清本地状态。"""
        redis_client = mock.MagicMock()
        redis_client.set.return_value = True
        redis_client.eval.return_value = 0  # owner 不匹配，未删
        leader = self._make_leader_with_redis(redis_client)
        leader.try_acquire()

        leader.release()
        self.assertIs(leader.is_leader, False)

    def test_release_when_not_leader_skips_redis(self):
        """从未 acquire 过 —— release 不碰 Redis。"""
        redis_client = mock.MagicMock()
        leader = self._make_leader_with_redis(redis_client)

        leader.release()
        redis_client.eval.assert_not_called()


class TestSchedulerLeaderTwoInstances(SimpleTestCase):
    """模拟两个调度器实例竞争同一把锁。"""

    def test_second_instance_fails_to_acquire_while_first_holds(self):
        # 用一个 dict 模拟 Redis 后端内存，跨两个 mock client 共享
        redis_state = {}

        def make_set():
            def _set(key, value, nx=False, ex=None):
                if nx and key in redis_state:
                    return False
                redis_state[key] = value
                return True
            return _set

        def make_eval(owner):
            def _eval(script, numkeys, *args):
                if "expire" in script:
                    # 续租：仅当 owner 匹配
                    return 1 if redis_state.get(LOCK_KEY) == owner else 0
                if "del" in script:
                    # 释放：仅当 owner 匹配才删
                    if redis_state.get(LOCK_KEY) == owner:
                        redis_state.pop(LOCK_KEY, None)
                        return 1
                    return 0
                return 0
            return _eval

        # 实例 1 抢锁成功
        leader1 = SchedulerLeader()
        c1 = mock.MagicMock()
        c1.set.side_effect = make_set()
        c1.eval.side_effect = make_eval(leader1.instance_id)
        leader1._redis = c1
        leader1._redis_unavailable = False
        self.assertIs(leader1.try_acquire(), True)

        # 实例 2 抢锁失败（锁被实例 1 持有）
        leader2 = SchedulerLeader()
        c2 = mock.MagicMock()
        c2.set.side_effect = make_set()
        leader2._redis = c2
        leader2._redis_unavailable = False
        self.assertIs(leader2.try_acquire(), False)
        self.assertIs(leader2.is_leader, False)

        # 实例 1 主动释放 —— instance 2 此后可抢到
        leader1.release()
        self.assertIs(leader1.is_leader, False)

        # 锁已释放，实例 2 重新抢成功
        self.assertIs(leader2.try_acquire(), True)
        self.assertIs(leader2.is_leader, True)

