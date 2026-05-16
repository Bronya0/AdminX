import json

from django.test import override_settings
from rest_framework.test import APITestCase
from rest_framework_simplejwt.tokens import RefreshToken
from rest_framework.throttling import SimpleRateThrottle


@override_settings(
    DEFAULT_THROTTLE_RATES={"anon": None, "user": None},
)
class AdminXTestCase(APITestCase):
    """测试基类"""

    @classmethod
    def setUpClass(cls):
        super().setUpClass()
        # DRF 实例化时会缓存 THROTTLE_RATES，强制覆盖
        SimpleRateThrottle.THROTTLE_RATES = {"anon": None, "user": None}

    def parse(self, response):
        response.render()
        return json.loads(response.content)

    def assert_code(self, response, expected_code):
        data = self.parse(response)
        self.assertEqual(data["code"], expected_code, msg=data.get("msg", ""))
        return data.get("data")

    def assert_ok(self, response):
        return self.assert_code(response, 200)

    def assert_created(self, response):
        return self.assert_code(response, 201)

    def assert_no_content(self, response):
        return self.assert_code(response, 204)

    def assert_fail(self, response, code=400):
        return self.assert_code(response, code)

    def auth(self, user):
        refresh = RefreshToken.for_user(user)
        self.client.credentials(HTTP_AUTHORIZATION=f"Bearer {refresh.access_token}")
