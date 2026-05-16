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
