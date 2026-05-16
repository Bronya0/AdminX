"""common — scheduler"""

from django.utils import timezone
from datetime import timedelta
from djangoadminx.common.scheduler import SchedulerManager, HEARTBEAT_ID
from djangoadminx.webservice.models import SchedulerHeartbeat


class TestSchedulerManager:
    def test_write_heartbeat(self):
        SchedulerManager.write_heartbeat()
        hb = SchedulerHeartbeat.objects.filter(id=HEARTBEAT_ID).first()
        assert hb is not None
        assert hb.last_heartbeat is not None

    def test_is_alive_with_heartbeat(self):
        SchedulerManager.write_heartbeat()
        assert SchedulerManager.is_alive() is True

    def test_is_alive_expired(self):
        SchedulerHeartbeat.objects.update_or_create(
            id=HEARTBEAT_ID,
            defaults={"last_heartbeat": timezone.now() - timedelta(minutes=5)},
        )
        assert SchedulerManager.is_alive() is False

    def test_clear_heartbeat(self):
        SchedulerManager.write_heartbeat()
        SchedulerManager.clear_heartbeat()
        assert SchedulerHeartbeat.objects.filter(id=HEARTBEAT_ID).count() == 0

    def test_notify_reload(self):
        SchedulerManager.notify_reload()
        hb = SchedulerHeartbeat.objects.filter(id=HEARTBEAT_ID).first()
        assert hb is not None
        assert hb.reload_pending is True
