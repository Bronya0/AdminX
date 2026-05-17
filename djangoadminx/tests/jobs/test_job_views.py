"""webservice — 任务执行、状态、重载、tasks"""

from django.test import override_settings
from djangoadminx.tests.base import AdminXTestCase
from djangoadminx.tests.factories import UserFactory


class TestScheduleJobViews(AdminXTestCase):
    def setUp(self):
        self.admin = UserFactory.create_admin()
        self.auth(self.admin)

    def test_status_endpoint(self):
        resp = self.client.get("/api/v1/jobs/status/")
        self.assert_ok(resp)

    def test_reload_endpoint(self):
        resp = self.client.post("/api/v1/jobs/reload/")
        self.assert_ok(resp)

    @override_settings(DEBUG=True)
    def test_run_once_with_result(self):
        from djangoadminx.jobs.models import ScheduleJob
        job = ScheduleJob.objects.create(
            name="RunTest2", handler="djangoadminx.jobs.tasks.sample_task",
            trigger_type="interval",
        )
        resp = self.client.post(f"/api/v1/jobs/{job.id}/run_once/")
        data = self.assert_ok(resp)
        assert data["result"] == "ok"

    def test_run_once_invalid_job(self):
        resp = self.client.post("/api/v1/jobs/00000000-0000-0000-0000-000000000000/run_once/")
        self.assert_fail(resp, 404)

    def test_list_job_logs(self):
        resp = self.client.get("/api/v1/job-logs/")
        self.assert_ok(resp)


class TestTasks:
    def test_sample_task(self):
        from djangoadminx.jobs.tasks import sample_task
        assert sample_task() == "ok"

    @override_settings()
    def test_system_resource_monitor(self):
        from djangoadminx.jobs.tasks import system_resource_monitor
        result = system_resource_monitor()
        assert result is not None
        assert "CPU" in result

    def test_ntp_sync_can_run(self):
        """ntp_sync 需要网络，只测函数可导入"""
        from djangoadminx.jobs.tasks import ntp_sync
        assert ntp_sync is not None
