import json

from django.db import migrations


def backfill_monitor_history_support(apps, schema_editor):
    ScheduleJob = apps.get_model("webservice", "ScheduleJob")
    Menu = apps.get_model("menu", "Menu")

    ScheduleJob.objects.filter(
        handler="djangoadminx.jobs.tasks.system_resource_monitor"
    ).update(trigger_config='{"minutes": 1}')

    menu = Menu.objects.filter(code="system:resource").first()
    if not menu:
        return

    try:
        allowed_paths = json.loads(menu.allowed_paths or "[]")
    except (TypeError, json.JSONDecodeError):
        allowed_paths = []

    history_path = "GET:/djangoadminx/api/v1/monitor/resources/history/"
    if history_path not in allowed_paths:
        allowed_paths.append(history_path)
        menu.allowed_paths = json.dumps(allowed_paths, ensure_ascii=False)
        menu.save(update_fields=["allowed_paths"])


class Migration(migrations.Migration):

    dependencies = [
        ("monitor", "0001_initial"),
        ("menu", "0002_menu_allowed_paths"),
        ("webservice", "0005_alter_schedulejob_handler"),
    ]

    operations = [
        migrations.RunPython(backfill_monitor_history_support, migrations.RunPython.noop),
    ]
