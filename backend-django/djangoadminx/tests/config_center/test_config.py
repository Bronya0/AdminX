import json

from django.test import override_settings

from djangoadminx.config_center.models import Config
from djangoadminx.tests.base import AdminXTestCase
from djangoadminx.tests.factories import UserFactory, ConfigFactory

TEST_FERNET_KEY = "VfExXUNXMlFdCJsWb0S1ibLyjfaeIlKfk3rP8-WLRCc="


@override_settings(FERNET_KEY=TEST_FERNET_KEY)
class TestConfigRead(AdminXTestCase):
    def setUp(self):
        self.admin = UserFactory.create_admin()
        self.auth(self.admin)
        ConfigFactory.create(key="SITE_NAME", value="MySite", group="site")
        ConfigFactory.create(key="MAX_COUNT", value="10", value_type="int", group="sys")
        ConfigFactory.create(key="ENABLED", value="true", value_type="bool", group="sys")
        ConfigFactory.create(key="DATA", value='{"a":1}', value_type="json", group="sys")

    def test_list_configs(self):
        resp = self.client.get("/api/v1/config/")
        self.assert_ok(resp)

    def test_get_by_group(self):
        resp = self.client.get("/api/v1/config/by_group/", {"group": "site"})
        data = self.assert_ok(resp)
        self.assertEqual(data.get("SITE_NAME"), "MySite")

    def test_get_value(self):
        resp = self.client.get("/api/v1/config/get_value/", {"key": "SITE_NAME"})
        data = self.assert_ok(resp)
        self.assertEqual(data["value"], "MySite")

    def test_get_value_not_found(self):
        resp = self.client.get("/api/v1/config/get_value/", {"key": "NONEXIST"})
        self.assert_fail(resp, 404)

    def test_parse_int_value(self):
        resp = self.client.get("/api/v1/config/get_value/", {"key": "MAX_COUNT"})
        data = self.assert_ok(resp)
        self.assertIsInstance(data["value"], int)
        self.assertEqual(data["value"], 10)

    def test_parse_bool_value(self):
        resp = self.client.get("/api/v1/config/get_value/", {"key": "ENABLED"})
        data = self.assert_ok(resp)
        self.assertIsInstance(data["value"], bool)
        self.assertTrue(data["value"])

    def test_model_has_query_indexes(self):
        index_names = {index.name for index in Config._meta.indexes}
        self.assertIn("config_group_active_key_idx", index_names)
        self.assertIn("config_active_type_created_idx", index_names)
        self.assertIn("config_act_enc_created_idx", index_names)


@override_settings(FERNET_KEY=TEST_FERNET_KEY)
class TestConfigCrud(AdminXTestCase):
    def setUp(self):
        self.admin = UserFactory.create_admin()
        self.auth(self.admin)

    def test_create_config(self):
        resp = self.client.post("/api/v1/config/", {
            "key": "TEST_KEY", "value": "test", "value_type": "string",
        })
        self.assert_created(resp)

    def test_create_duplicate_key(self):
        ConfigFactory.create(key="DUP", value="first")
        resp = self.client.post("/api/v1/config/", {
            "key": "DUP", "value": "second", "value_type": "string",
        })
        self.assert_fail(resp, 400)

    def test_update_config(self):
        c = ConfigFactory.create(key="OLD", value="old")
        resp = self.client.put(f"/api/v1/config/{c.id}/", {
            "key": "OLD", "value": "new", "value_type": "string",
        })
        self.assert_ok(resp)


@override_settings(FERNET_KEY=TEST_FERNET_KEY)
class TestConfigEncryption(AdminXTestCase):
    def setUp(self):
        self.admin = UserFactory.create_admin()
        self.auth(self.admin)

    def _create_encrypted(self, key="ENC_KEY", value="secret", value_type="string", **kw):
        resp = self.client.post("/api/v1/config/", {
            "key": key, "value": value, "value_type": value_type,
            "is_encrypted": True, **kw,
        }, format="json")
        return self.assert_created(resp)

    # ── API 读写 ──

    def test_create_encrypted_config_masks_value_in_response(self):
        resp = self.client.post("/api/v1/config/", {
            "key": "ENC", "value": "mysecret", "value_type": "string",
            "is_encrypted": True,
        }, format="json")
        data = self.assert_created(resp)
        self.assertEqual(data["value"], "")
        self.assertEqual(data["display_value"], "********")
        self.assertTrue(data["is_encrypted"])

    def test_list_shows_masked_display_value(self):
        self._create_encrypted()
        resp = self.client.get("/api/v1/config/")
        data = self.assert_ok(resp)
        config = next(c for c in data["results"] if c["key"] == "ENC_KEY")
        self.assertEqual(config["value"], "")
        self.assertEqual(config["display_value"], "********")
        self.assertTrue(config["is_encrypted"])

    def test_get_value_returns_decrypted_plaintext(self):
        self._create_encrypted()
        resp = self.client.get("/api/v1/config/get_value/", {"key": "ENC_KEY"})
        data = self.assert_ok(resp)
        self.assertEqual(data["value"], "secret")

    def test_get_by_group_returns_decrypted_value(self):
        self._create_encrypted(group="secrets")
        resp = self.client.get("/api/v1/config/by_group/", {"group": "secrets"})
        data = self.assert_ok(resp)
        self.assertEqual(data["ENC_KEY"], "secret")

    def test_update_encrypted_without_value_preserves_original(self):
        c = self._create_encrypted()
        resp = self.client.put(f"/api/v1/config/{c['id']}/", {
            "key": "ENC_KEY", "value_type": "string", "is_encrypted": True,
        }, format="json")
        self.assert_ok(resp)
        resp2 = self.client.get("/api/v1/config/get_value/", {"key": "ENC_KEY"})
        data = self.assert_ok(resp2)
        self.assertEqual(data["value"], "secret")

    def test_update_encrypted_with_new_value(self):
        c = self._create_encrypted()
        resp = self.client.put(f"/api/v1/config/{c['id']}/", {
            "key": "ENC_KEY", "value": "newsecret", "value_type": "string",
            "is_encrypted": True,
        }, format="json")
        self.assert_ok(resp)
        resp2 = self.client.get("/api/v1/config/get_value/", {"key": "ENC_KEY"})
        data = self.assert_ok(resp2)
        self.assertEqual(data["value"], "newsecret")

    def test_disable_encryption_rejected(self):
        c = self._create_encrypted()
        resp = self.client.put(f"/api/v1/config/{c['id']}/", {
            "key": "ENC_KEY", "value": "secret", "value_type": "string",
            "is_encrypted": False,
        }, format="json")
        self.assert_fail(resp, 400)
        resp2 = self.client.get("/api/v1/config/get_value/", {"key": "ENC_KEY"})
        data = self.assert_ok(resp2)
        self.assertEqual(data["value"], "secret")

    # ── 加密 + 类型组合 ──

    def test_encrypted_int_value_parsed_correctly(self):
        self._create_encrypted(key="ENC_INT", value="42", value_type="int")
        resp = self.client.get("/api/v1/config/get_value/", {"key": "ENC_INT"})
        data = self.assert_ok(resp)
        self.assertIsInstance(data["value"], int)
        self.assertEqual(data["value"], 42)

    def test_encrypted_bool_value_parsed_correctly(self):
        self._create_encrypted(key="ENC_BOOL", value="true", value_type="bool")
        resp = self.client.get("/api/v1/config/get_value/", {"key": "ENC_BOOL"})
        data = self.assert_ok(resp)
        self.assertIsInstance(data["value"], bool)
        self.assertTrue(data["value"])

    def test_encrypted_json_value_parsed_correctly(self):
        self._create_encrypted(key="ENC_JSON", value='{"a":1}', value_type="json")
        resp = self.client.get("/api/v1/config/get_value/", {"key": "ENC_JSON"})
        data = self.assert_ok(resp)
        self.assertEqual(data["value"], {"a": 1})

    # ── 筛选过滤 ──

    def test_filter_by_is_encrypted_true(self):
        self._create_encrypted()
        ConfigFactory.create(key="PLAIN", value="plain")
        resp = self.client.get("/api/v1/config/", {"is_encrypted": "true"})
        data = self.assert_ok(resp)
        self.assertTrue(all(c["is_encrypted"] for c in data["results"]))
        self.assertEqual(len(data["results"]), 1)

    def test_filter_by_is_encrypted_false(self):
        self._create_encrypted()
        ConfigFactory.create(key="PLAIN", value="plain")
        resp = self.client.get("/api/v1/config/", {"is_encrypted": "false"})
        data = self.assert_ok(resp)
        self.assertTrue(all(not c["is_encrypted"] for c in data["results"]))

    # ── 边界 ──

    def test_empty_value_with_encryption(self):
        resp = self.client.post("/api/v1/config/", {
            "key": "EMPTY_ENC", "value": "", "value_type": "string",
            "is_encrypted": True,
        }, format="json")
        self.assert_created(resp)
        resp2 = self.client.get("/api/v1/config/get_value/", {"key": "EMPTY_ENC"})
        data = self.assert_ok(resp2)
        self.assertEqual(data["value"], "")
        self.assertEqual(data["key"], "EMPTY_ENC")


@override_settings(FERNET_KEY=None)
class TestConfigEncryptionNoKey(AdminXTestCase):
    """不配置 FERNET_KEY 时，加密不应崩溃，值应降级为 <<解密失败>>"""

    def setUp(self):
        self.admin = UserFactory.create_admin()
        self.auth(self.admin)

    def test_create_encrypted_without_key_does_not_crash(self):
        resp = self.client.post("/api/v1/config/", {
            "key": "NOKEY", "value": "secret", "value_type": "string",
            "is_encrypted": True,
        }, format="json")
        self.assert_created(resp)
        resp2 = self.client.get("/api/v1/config/get_value/", {"key": "NOKEY"})
        data = self.assert_ok(resp2)
        self.assertEqual(data["value"], "<<解密失败>>")
