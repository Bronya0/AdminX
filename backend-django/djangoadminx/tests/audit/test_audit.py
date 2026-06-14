"""audit — 审计日志 signals + mixins"""

from djangoadminx.tests.base import AdminXTestCase
from djangoadminx.tests.factories import UserFactory
from djangoadminx.audit.signals import set_current_request, get_current_request
from djangoadminx.audit.utils import serialize_for_json, get_operator, get_operator_ip


class TestAuditUtils:
    def test_serialize_for_json(self):
        from djangoadminx.accounts.models import User
        user = User.objects.create(username="ser_test")
        result = serialize_for_json(user)
        assert "username" in result
        assert result["username"] == "ser_test"
        user.delete()

    def test_get_operator_no_request(self):
        assert get_operator(None) == "system"

    def test_get_operator_ip_no_request(self):
        assert get_operator_ip(None) == ""


class TestAuditSignals(AdminXTestCase):
    def setUp(self):
        self.admin = UserFactory.create_admin()
        self.auth(self.admin)

    def test_create_creates_audit_log(self):
        from djangoadminx.audit.models import AuditLog
        # 创建配置会触发 AuditLogMixin
        count_before = AuditLog.objects.count()
        self.client.post("/api/v1/config/", {
            "key": "AUDIT_TEST", "value": "x", "value_type": "string",
        })
        count_after = AuditLog.objects.count()
        assert count_after > count_before

    def test_audit_log_list(self):
        resp = self.client.get("/api/v1/audit/")
        self.assert_ok(resp)

    def test_audit_log_detail(self):
        from djangoadminx.audit.models import AuditLog
        log = AuditLog.objects.create(
            action=AuditLog.ActionChoices.CREATE,
            model_name="test.Model",
            object_id="1",
            object_repr="test",
            operator="tester",
        )
        resp = self.client.get(f"/api/v1/audit/{log.id}/")
        self.assert_ok(resp)


class TestAuditSignalManager:
    def test_register(self):
        from djangoadminx.audit.signals import AuditSignalManager
        AuditSignalManager._registered = False
        AuditSignalManager.register()
        assert AuditSignalManager._registered is True
        # 重复注册不报错
        AuditSignalManager.register()

    def test_set_current_request(self):
        set_current_request("test_req")
        assert get_current_request() == "test_req"
