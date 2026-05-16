from djangoadminx.tests.base import AdminXTestCase
from djangoadminx.tests.factories import UserFactory, ConfigFactory


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
