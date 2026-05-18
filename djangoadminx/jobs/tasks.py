"""任务处理函数 — 被 APScheduler 动态导入调用"""

import json
import logging
from datetime import timedelta

from django.utils import timezone

logger = logging.getLogger("djangoadminx.jobs.tasks")


def ntp_sync(server="pool.ntp.org", version=3):
    """NTP 时间同步任务"""
    import ntplib
    client = ntplib.NTPClient()
    response = client.request(server, version=version)
    return json.dumps({
        "server": server,
        "offset": response.offset,
        "delay": response.delay,
        "time": str(response.tx_time),
    })


def sample_task():
    """示例任务"""
    logger.info("Sample task executed")
    return "ok"


def system_resource_monitor():
    """系统资源监控 — 定时检查 CPU/内存/磁盘使用率"""
    from djangoadminx.config_center.models import Config
    from djangoadminx.monitor.utils import SystemMonitor
    from djangoadminx.notification.utils import create_notification

    thresholds = {
        "cpu": {
            "warn": int(Config.get_value("CPU_WARN_THRESHOLD", default=80)),
            "crit": int(Config.get_value("CPU_CRIT_THRESHOLD", default=90)),
        },
        "mem": {
            "warn": int(Config.get_value("MEM_WARN_THRESHOLD", default=80)),
            "crit": int(Config.get_value("MEM_CRIT_THRESHOLD", default=90)),
        },
        "disk": {
            "warn": int(Config.get_value("DISK_WARN_THRESHOLD", default=85)),
            "crit": int(Config.get_value("DISK_CRIT_THRESHOLD", default=95)),
        },
    }

    alerts = []

    _, snapshot = SystemMonitor.record_snapshot(sample_interval=1)
    disk_usage = snapshot.get("disk_usage") or {}
    memory = snapshot.get("memory") or {}

    cpu_percent = snapshot["cpu"].get("percent", 0)
    if cpu_percent >= thresholds["cpu"]["crit"]:
        alerts.append(("error", "CPU 使用率严重过高",
                       f"当前 CPU 使用率 {cpu_percent}%，超过严重阈值 {thresholds['cpu']['crit']}%"))
    elif cpu_percent >= thresholds["cpu"]["warn"]:
        alerts.append(("warning", "CPU 使用率偏高",
                       f"当前 CPU 使用率 {cpu_percent}%，超过警告阈值 {thresholds['cpu']['warn']}%"))

    mem_percent = memory.get("percent", 0)
    mem_used_gb = memory.get("used", 0) / 1024 ** 3
    mem_total_gb = memory.get("total", 0) / 1024 ** 3
    if mem_percent >= thresholds["mem"]["crit"]:
        alerts.append(("error", "内存使用率严重过高",
                       f"当前内存使用率 {mem_percent:.1f}%（已用 {mem_used_gb:.1f}GB / 总计 {mem_total_gb:.1f}GB），超过严重阈值 {thresholds['mem']['crit']}%"))
    elif mem_percent >= thresholds["mem"]["warn"]:
        alerts.append(("warning", "内存使用率偏高",
                       f"当前内存使用率 {mem_percent:.1f}%（已用 {mem_used_gb:.1f}GB / 总计 {mem_total_gb:.1f}GB），超过警告阈值 {thresholds['mem']['warn']}%"))

    disk_percent = disk_usage.get("percent", 0)
    disk_used_gb = disk_usage.get("used", 0) / 1024 ** 3
    disk_total_gb = disk_usage.get("total", 0) / 1024 ** 3

    if disk_percent >= thresholds["disk"]["crit"]:
        alerts.append(("error", "磁盘使用率严重过高",
                       f"当前磁盘使用率 {disk_percent:.1f}%（已用 {disk_used_gb:.1f}GB / 总计 {disk_total_gb:.1f}GB），超过严重阈值 {thresholds['disk']['crit']}%"))
    elif disk_percent >= thresholds["disk"]["warn"]:
        alerts.append(("warning", "磁盘使用率偏高",
                       f"当前磁盘使用率 {disk_percent:.1f}%（已用 {disk_used_gb:.1f}GB / 总计 {disk_total_gb:.1f}GB），超过警告阈值 {thresholds['disk']['warn']}%"))

    from djangoadminx.notification.models import Notification

    cooldown = timezone.now() - timedelta(minutes=60)
    for ntype, title, content in alerts:
        exists = Notification.objects.filter(
            title=title, created_at__gte=cooldown, user__isnull=True,
        ).exists()
        if not exists:
            create_notification(title=title, content=content, notification_type=ntype)
            logger.warning("资源告警已创建: [%s] %s", ntype, title)

    return (
        f"CPU: {cpu_percent}% | 内存: {mem_percent:.1f}% | "
        f"磁盘: {disk_percent:.1f}% | 告警: {len(alerts)} 条"
    )
