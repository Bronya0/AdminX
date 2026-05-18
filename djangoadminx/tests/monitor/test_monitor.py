"""monitor — system monitoring"""

from datetime import timedelta

from django.utils import timezone

from djangoadminx.tests.base import AdminXTestCase
from djangoadminx.tests.factories import UserFactory
from djangoadminx.monitor.models import SystemMetricSnapshot
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

    def test_resources_history(self):
        now = timezone.now()
        SystemMetricSnapshot.objects.bulk_create([
            SystemMetricSnapshot(
                collected_at=now - timedelta(minutes=2),
                cpu_percent=10,
                memory_percent=20,
                disk_percent=30,
                disk_read_bytes=1024 * 1024,
                disk_write_bytes=2 * 1024 * 1024,
            ),
            SystemMetricSnapshot(
                collected_at=now - timedelta(minutes=1),
                cpu_percent=15,
                memory_percent=25,
                disk_percent=35,
                disk_read_bytes=3 * 1024 * 1024,
                disk_write_bytes=5 * 1024 * 1024,
            ),
        ])

        resp = self.client.get("/api/v1/monitor/resources/history/?range=1h&interval=1m")
        data = self.assert_ok(resp)
        assert data["interval"] == "1m"
        assert len(data["points"]) >= 1

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

    def test_history(self):
        now = timezone.now()
        SystemMetricSnapshot.objects.bulk_create([
            SystemMetricSnapshot(
                collected_at=now - timedelta(minutes=2),
                cpu_percent=10,
                memory_percent=20,
                disk_percent=30,
                disk_read_bytes=1024,
                disk_write_bytes=2048,
            ),
            SystemMetricSnapshot(
                collected_at=now - timedelta(minutes=1),
                cpu_percent=20,
                memory_percent=30,
                disk_percent=40,
                disk_read_bytes=4096,
                disk_write_bytes=8192,
            ),
        ])

        history = SystemMonitor.history(range_key="1h", interval_key="1m")
        assert history["interval"] == "1m"
        assert history["points"]

    def test_netstat(self):
        r = SystemMonitor.netstat()
        if "error" not in r:
            assert "LISTEN" in r
