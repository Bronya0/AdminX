"""captcha — 验证码"""

from djangoadminx.tests.base import AdminXTestCase


class TestCaptcha(AdminXTestCase):
    def test_get_captcha_returns_json(self):
        resp = self.client.get("/api/v1/captcha/captcha/")
        data = self.assert_ok(resp)
        assert "captcha_id" in data
        assert "svg" in data
        assert data["svg"].startswith("<svg")

    def test_verify_success(self):
        resp = self.client.get("/api/v1/captcha/captcha/")
        data = self.assert_ok(resp)
        cid = data["captcha_id"]
        from django.core.cache import cache
        expected = cache.get(f"captcha:{cid}")
        assert expected is not None

        resp2 = self.client.post("/api/v1/captcha/captcha/verify/", {
            "captcha_id": cid, "captcha_text": expected,
        })
        data2 = self.assert_ok(resp2)
        assert data2["verified"] is True

    def test_verify_wrong(self):
        resp = self.client.get("/api/v1/captcha/captcha/")
        data = self.assert_ok(resp)
        resp2 = self.client.post("/api/v1/captcha/captcha/verify/", {
            "captcha_id": data["captcha_id"], "captcha_text": "WRONG",
        })
        d = self.assert_fail(resp2, 400)
        assert d["verified"] is False

    def test_verify_missing_id(self):
        resp = self.client.post("/api/v1/captcha/captcha/verify/", {
            "captcha_id": "", "captcha_text": "abc",
        })
        self.assert_fail(resp, 400)

    def test_verify_once_only(self):
        resp = self.client.get("/api/v1/captcha/captcha/")
        data = self.assert_ok(resp)
        cid = data["captcha_id"]
        from django.core.cache import cache
        expected = cache.get(f"captcha:{cid}")

        self.client.post("/api/v1/captcha/captcha/verify/", {
            "captcha_id": cid, "captcha_text": expected,
        })
        # 第二次应该失败（已消费）
        resp2 = self.client.post("/api/v1/captcha/captcha/verify/", {
            "captcha_id": cid, "captcha_text": expected,
        })
        self.assert_fail(resp2, 400)
