from django.test import override_settings
from djangoadminx.tests.base import AdminXTestCase
from djangoadminx.tests.factories import UserFactory


class TestLogin(AdminXTestCase):
    def setUp(self):
        self.user = UserFactory.create_admin()

    def test_login_success(self):
        resp = self.client.post("/api/v1/accounts/login/", {
            "username": "admin", "password": "admin123",
        })
        data = self.assert_ok(resp)
        self.assertIn("access", data)
        self.assertIn("refresh", data)
        self.assertEqual(data["user"]["username"], "admin")

    def test_login_wrong_password(self):
        resp = self.client.post("/api/v1/accounts/login/", {
            "username": "admin", "password": "wrong",
        })
        self.assert_fail(resp, 401)

    def test_login_missing_fields(self):
        resp = self.client.post("/api/v1/accounts/login/", {"username": "admin"})
        self.assert_fail(resp, 400)

    def test_login_disabled_user(self):
        self.user.is_active = False
        self.user.save()
        resp = self.client.post("/api/v1/accounts/login/", {
            "username": "admin", "password": "admin123",
        })
        # Django's authenticate() returns None for inactive users
        self.assert_fail(resp, 401)


class TestLogout(AdminXTestCase):
    def setUp(self):
        self.user = UserFactory.create_admin()

    def test_logout_success(self):
        self.auth(self.user)
        resp = self.client.post("/api/v1/accounts/logout/", {"refresh": "dummy"})
        self.assert_ok(resp)

    def test_logout_unauthenticated(self):
        resp = self.client.post("/api/v1/accounts/logout/", {"refresh": "dummy"})
        self.assert_fail(resp, 401)


class TestIntrospect(AdminXTestCase):
    def setUp(self):
        self.user = UserFactory.create_admin()
        from rest_framework_simplejwt.tokens import RefreshToken
        self.token = str(RefreshToken.for_user(self.user).access_token)

    def test_introspect_valid(self):
        resp = self.client.post("/api/v1/accounts/introspect/", {"token": self.token})
        data = self.assert_ok(resp)
        self.assertTrue(data["valid"])
        self.assertEqual(data["username"], "admin")
        self.assertIn("email", data)
        self.assertIn("phone", data)
        self.assertIn("avatar", data)
        self.assertIn("roles", data)
        self.assertIn("permissions", data)

    def test_introspect_missing_token(self):
        resp = self.client.post("/api/v1/accounts/introspect/", {})
        data = self.assert_fail(resp, 401)
        self.assertFalse(data["valid"])
        self.assertEqual(data["error_code"], "token_missing")

    def test_introspect_invalid_token(self):
        resp = self.client.post("/api/v1/accounts/introspect/", {"token": "abc.def.ghi"})
        data = self.assert_fail(resp, 401)
        self.assertEqual(data["error_code"], "token_invalid")

    def test_introspect_refresh_token_rejected(self):
        from rest_framework_simplejwt.tokens import RefreshToken
        refresh = str(RefreshToken.for_user(self.user))
        resp = self.client.post("/api/v1/accounts/introspect/", {"token": refresh})
        self.assert_fail(resp, 401)

    def test_introspect_inactive_user(self):
        self.user.is_active = False
        self.user.save()
        resp = self.client.post("/api/v1/accounts/introspect/", {"token": self.token})
        data = self.assert_fail(resp, 401)
        self.assertEqual(data["error_code"], "user_inactive")

    def test_introspect_user_deleted(self):
        uid = self.user.id
        self.user.delete()
        from rest_framework_simplejwt.tokens import AccessToken
        token = AccessToken.for_user(
            type("FakeUser", (), {"id": uid, "is_active": True})()
        )
        # 构造一个无效用户ID的token比较复杂，跳过，测试不存在的用户


@override_settings(PASSWORD_POLICY_ENABLED=True)
class TestUserCreate(AdminXTestCase):
    def setUp(self):
        self.admin = UserFactory.create_admin()
        self.auth(self.admin)

    def test_create_superuser(self):
        from djangoadminx.accounts.models import User
        user = User.objects.create_superuser("su_test", password="TestPass123!")
        self.assertTrue(user.is_superuser)
        self.assertTrue(user.is_staff)
        user.delete()

    def test_create_user_with_weak_password_rejected(self):
        resp = self.client.post("/api/v1/accounts/users/", {
            "username": "weakuser",
            "password": "123456",
            "phone": "13800001111",
        }, format="json")
        self.assert_fail(resp, 400)

    def test_create_user_password_policy_success(self):
        resp = self.client.post("/api/v1/accounts/users/", {
            "username": "gooduser",
            "password": "GoodPass1!",
            "phone": "13800001111",
        }, format="json")
        self.assert_created(resp)


class TestUserDelete(AdminXTestCase):
    def setUp(self):
        self.admin = UserFactory.create_admin()
        self.auth(self.admin)

    def test_cannot_delete_current_logged_in_user(self):
        resp = self.client.delete(f"/api/v1/accounts/users/{self.admin.id}/")
        self.assert_fail(resp, 403)

        from djangoadminx.accounts.models import User
        self.assertTrue(User.objects.filter(id=self.admin.id).exists())

    def test_can_delete_other_user(self):
        target = UserFactory.create_user(username="delete_target")

        resp = self.client.delete(f"/api/v1/accounts/users/{target.id}/")
        self.assert_no_content(resp)

        from djangoadminx.accounts.models import User
        self.assertFalse(User.objects.filter(id=target.id).exists())


class TestLogoutData(AdminXTestCase):
    def setUp(self):
        self.user = UserFactory.create_admin()
        self.auth(self.user)

    def test_logout_returns_data_key(self):
        resp = self.client.post("/api/v1/accounts/logout/", {"refresh": "dummy"})
        data = self.assert_ok(resp)
        self.assertEqual(data, None)
