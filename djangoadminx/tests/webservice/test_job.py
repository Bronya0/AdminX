from django.test import override_settings

from djangoadminx.tests.base import AdminXTestCase
from djangoadminx.tests.factories import UserFactory


class TestScheduleJob(AdminXTestCase):
    def setUp(self):
        self.admin = UserFactory.create_admin()
        self.auth(self.admin)

    def test_create_python_job(self):
        resp = self.client.post("/api/v1/webservice/jobs/", {
            "name": "测试任务",
            "command_type": "python",
            "handler": "djangoadminx.webservice.tasks.sample_task",
            "trigger_type": "interval",
            "trigger_config": '{"minutes": 5}',
        })
        data = self.assert_created(resp)
        self.assertEqual(data["name"], "测试任务")

    def test_create_shell_job(self):
        resp = self.client.post("/api/v1/webservice/jobs/", {
            "name": "Shell任务",
            "command_type": "shell",
            "command": "echo hello",
            "trigger_type": "interval",
            "trigger_config": '{"minutes": 5}',
        })
        data = self.assert_created(resp)
        self.assertEqual(data["name"], "Shell任务")

    def test_create_job_missing_handler(self):
        resp = self.client.post("/api/v1/webservice/jobs/", {
            "name": "无效任务",
            "command_type": "python",
            "handler": "",
            "trigger_type": "interval",
        })
        self.assert_fail(resp, 400)

    @override_settings(DEBUG=True)
    def test_run_once_python(self):
        from djangoadminx.webservice.models import ScheduleJob
        job = ScheduleJob.objects.create(
            name="RunTest",
            handler="djangoadminx.webservice.tasks.sample_task",
            trigger_type="interval",
        )
        resp = self.client.post(f"/api/v1/webservice/jobs/{job.id}/run_once/")
        data = self.assert_ok(resp)
        self.assertIn("result", data)

    def test_toggle_active(self):
        from djangoadminx.webservice.models import ScheduleJob
        job = ScheduleJob.objects.create(
            name="ToggleTest", handler="djangoadminx.webservice.tasks.sample_task",
            trigger_type="interval", is_active=False,
        )
        resp = self.client.put(f"/api/v1/webservice/jobs/{job.id}/", {
            "name": "ToggleTest", "is_active": True,
            "handler": "djangoadminx.webservice.tasks.sample_task",
            "trigger_type": "interval",
        })
        self.assert_ok(resp)
