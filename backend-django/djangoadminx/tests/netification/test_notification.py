"""notification — via API"""

from djangoadminx.tests.base import AdminXTestCase
from djangoadminx.tests.factories import UserFactory


class TestNotification(AdminXTestCase):
    def setUp(self):
        self.admin = UserFactory.create_admin()
        self.auth(self.admin)

    def test_list_notifications(self):
        resp = self.client.get("/api/v1/notification/messages/")
        self.assert_ok(resp)

    def test_unread_count(self):
        resp = self.client.get("/api/v1/notification/unread_count/")
        self.assert_ok(resp)

    def test_mark_all_read(self):
        resp = self.client.post("/api/v1/notification/mark_all_read/")
        self.assert_ok(resp)


class TestWebhook(AdminXTestCase):
    def setUp(self):
        self.admin = UserFactory.create_admin()
        self.auth(self.admin)

    def test_create_webhook(self):
        resp = self.client.post("/api/v1/notification/webhooks/", {
            "name": "test", "url": "https://example.com/hook",
        })
        self.assert_created(resp)

    def test_list_webhooks(self):
        resp = self.client.get("/api/v1/notification/webhooks/")
        self.assert_ok(resp)
