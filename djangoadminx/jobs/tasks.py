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


def component_health_monitor():
    """组件健康监控 — 检查系统组件和业务组件状态，异常时写通知"""
    from datetime import timedelta

    from django.db import connection
    from django.core.cache import cache
    from django.utils import timezone

    from djangoadminx.cluster.models import ServiceComponent
    from djangoadminx.common.scheduler import SchedulerManager
    from djangoadminx.notification.utils import create_notification
    from djangoadminx.notification.models import Notification

    alerts = []

    # ── 系统组件 ──
    try:
        connection.ensure_connection()
    except Exception as e:
        alerts.append(("error", "数据库连接异常", f"数据库无法连接: {e}"))
    else:
        alerts.append(("info", "数据库连接正常", "数据库连接正常"))

    try:
        cache.set("__component_ping__", "1", 5)
    except Exception as e:
        alerts.append(("error", "缓存服务异常", f"缓存服务无法连接: {e}"))
    else:
        backend_cls = type(cache).__name__
        backend_name = "Redis" if "Redis" in backend_cls else ("Memcached" if "Memcache" in backend_cls else backend_cls)
        alerts.append(("info", "缓存服务正常", f"缓存后端: {backend_name}"))

    if not SchedulerManager.is_alive():
        alerts.append(("error", "调度器未运行", "调度器进程未响应心跳，可能已停止"))

    # ── 业务组件 ──
    offline_components = ServiceComponent.objects.all()
    offline_names = []
    for comp in offline_components:
        if comp.status == "offline":
            offline_names.append(f"{comp.name}({comp.app_label})")

    if offline_names:
        alerts.append((
            "warning",
            "业务组件离线",
            f"以下 {len(offline_names)} 个业务组件未上报心跳（离线超过{ServiceComponent.HEARTBEAT_TTL}s）:\n" +
            "\n".join(f"  - {n}" for n in offline_names),
        ))

    # ── 写通知（60 分钟冷却） ──
    cooldown = timezone.now() - timedelta(minutes=60)
    count = 0
    for ntype, title, content in alerts:
        # info 类型不写入通知（仅用于正常状态确认）
        if ntype == "info":
            continue
        exists = Notification.objects.filter(
            title=title, created_at__gte=cooldown, user__isnull=True,
        ).exists()
        if not exists:
            create_notification(title=title, content=content, notification_type=ntype)
            count += 1
            logger.warning("组件健康告警: [%s] %s", ntype, title)

    return f"系统组件: {'OK' if count == 0 else f'{count} 条告警'} | 业务组件: {len(list(ServiceComponent.objects.all()))} 个（离线 {len(offline_names)}）"


def cleanup_job_logs():
    """清理过期任务日志 — 防止 JobLog 无限累积"""
    from datetime import timedelta

    from django.utils import timezone

    from djangoadminx.config_center.models import Config
    from djangoadminx.jobs.models import JobLog

    days = int(Config.get_value("JOB_LOG_RETENTION_DAYS", default=30))
    cutoff = timezone.now() - timedelta(days=days)
    deleted, _ = JobLog.objects.filter(started_at__lt=cutoff).delete()
    logger.info("清理任务日志: 保留 %d 天, 删除 %d 条", days, deleted)
    return f"保留 {days} 天, 已删除 {deleted} 条"
