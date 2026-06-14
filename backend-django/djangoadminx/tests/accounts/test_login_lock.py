from datetime import timedelta

from django.test import override_settings
from django.utils import timezone
from djangoadminx.tests.base import AdminXTestCase
from djangoadminx.tests.factories import UserFactory
from djangoadminx.accounts.models import LoginLock


@override_settings(
    LOGIN_LOCK_ENABLED=True,
    LOGIN_MAX_ATTEMPTS=3,
    LOGIN_LOCK_DURATION=timedelta(minutes=1),
)
class TestLoginLock(AdminXTestCase):
    def setUp(self):
        self.user = UserFactory.create_admin()

    def test_lock_after_max_attempts(self):
        """连续失败达到阈值后锁定"""
        for i in range(3):
            self.client.post("/api/v1/accounts/login/", {
                "username": "admin", "password": "wrong",
            })
        resp = self.client.post("/api/v1/accounts/login/", {
            "username": "admin", "password": "wrong",
        })
        self.assert_fail(resp, 423)

    def test_lock_clears_after_expiry_and_login_succeeds(self):
        """锁定过期后自动清除锁，登录成功"""
        # 手动创建一条过期的锁定记录
        LoginLock.objects.create(
            username="admin",
            failed_count=5,
            locked_at=timezone.now() - timedelta(hours=1),
        )
        resp = self.client.post("/api/v1/accounts/login/", {
            "username": "admin", "password": "admin123",
        })
        self.assert_ok(resp)

        # 锁定记录应被清除
        self.assertFalse(LoginLock.objects.filter(username="admin").exists())
