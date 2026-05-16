"""monitor — system monitoring"""

from djangoadminx.tests.base import AdminXTestCase
from djangoadminx.tests.factories import UserFactory
from djangoadminx.monitor.utils import SystemMonitor


class TestMonitorAPI(AdminXTestCase):
    def setUp(self):
        self.admin = UserFactory.create_admin()
        self.auth(self.admin)

    def test_resources(self):
        resp = self.client.get("/api/v1/monitor/resources/")
        self.assert_ok(resp)

    def test_netstat(self):
        resp = self.client.get("/api/v1/monitor/netstat/")
        self.assert_ok(resp)

    def test_unauthorized(self):
        self.client.credentials()
        resp = self.client.get("/api/v1/monitor/resources/")
        self.assert_fail(resp, 401)


class TestSystemMonitorUtils:
    def test_cpu(self):
        r = SystemMonitor.cpu()
        assert "percent" in r

    def test_memory(self):
        r = SystemMonitor.memory()
        assert "total" in r

    def test_disk(self):
        assert isinstance(SystemMonitor.disk(), list)

    def test_disk_io(self):
        assert isinstance(SystemMonitor.disk_io(), dict)

    def test_all(self):
        r = SystemMonitor.all()
        assert "cpu" in r and "memory" in r

    def test_netstat(self):
        r = SystemMonitor.netstat()
        if "error" not in r:
            assert "LISTEN" in r
