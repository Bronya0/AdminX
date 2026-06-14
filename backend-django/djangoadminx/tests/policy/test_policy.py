"""policy — 密码策略"""

from django.test import override_settings
from djangoadminx.tests.base import AdminXTestCase
from djangoadminx.tests.factories import UserFactory
from djangoadminx.policy.models import PasswordPolicy, PasswordHistory


class TestPasswordPolicyModel:
    def test_default_policy(self):
        policy = PasswordPolicy(min_length=6, require_upper=False, require_lower=False, require_digit=False, require_special=False)
        valid, errors = policy.validate("abcdef")
        assert valid
        assert len(errors) == 0

    def test_min_length(self):
        policy = PasswordPolicy(min_length=8)
        valid, errors = policy.validate("short")
        assert not valid
        assert any("8" in e for e in errors)

    def test_require_upper(self):
        policy = PasswordPolicy(require_upper=True)
        valid, errors = policy.validate("alllower1!")
        assert not valid

    def test_require_special(self):
        policy = PasswordPolicy(require_special=True)
        valid, errors = policy.validate("NoSpecial1")
        assert not valid

    def test_history_reuse(self):
        from django.contrib.auth import get_user_model
        User = get_user_model()
        user = User.objects.create(username="hist_user")
        PasswordHistory.record(user, "oldpass1")

        policy = PasswordPolicy(history_count=1, require_upper=False, require_lower=False, require_digit=False, require_special=False)
        valid, errors = policy.validate("oldpass1", user)
        assert not valid
        assert "最近" in str(errors)

    def test_is_expired_never(self):
        policy = PasswordPolicy(expire_days=0)
        user = UserFactory.create_user()
        assert policy.is_expired(user) is False

    def test_is_expired_recent(self):
        from django.utils import timezone
        policy = PasswordPolicy(expire_days=90)
        user = UserFactory.create_user()
        user.last_login = timezone.now()
        assert policy.is_expired(user) is False

    def test_is_expired_past(self):
        from django.utils import timezone
        from datetime import timedelta
        policy = PasswordPolicy(expire_days=1)
        user = UserFactory.create_user()
        user.last_login = timezone.now() - timedelta(days=2)
        assert policy.is_expired(user) is True

    def test_get_instance_disabled(self):
        with override_settings(PASSWORD_POLICY_ENABLED=False):
            assert PasswordPolicy.get_instance() is None


class TestPasswordChange(AdminXTestCase):
    def setUp(self):
        self.user = UserFactory.create_admin()
        self.auth(self.user)

    def test_change_password(self):
        resp = self.client.post("/api/v1/policy/change-password/", {
            "old_password": "admin123",
            "new_password": "NewPass123!",
        })
        self.assert_ok(resp)
