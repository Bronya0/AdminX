"""WebService 任务处理函数 — 被 APScheduler 动态导入调用"""

import json
import logging
from datetime import timedelta

from django.utils import timezone

logger = logging.getLogger("djangoadminx.webservice.tasks")


def call_webservice(webservice_id, params=None):
    """调用外部 WebService (通用任务)"""
    from djangoadminx.webservice.models import WebService
    from zeep import Client

    ws = WebService.objects.get(id=webservice_id)
    client = Client(ws.wsdl_url)
    method = getattr(client.service, ws.method)
    params = params or {}
    result = method(**params)
    logger.info(f"WebService call {ws.name}.{ws.method}: {result}")
    return str(result)


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
    """系统资源监控 — 定时检查 CPU/内存/磁盘使用率，超阈值时创建告警通知

    最佳实践阈值（可在配置中心调整）:
        CPU:  警告 80%, 严重 90%
        内存: 警告 80%, 严重 90%
        磁盘: 警告 85%, 严重 95%

    推荐调度方式: interval 每 5 分钟执行一次
    告警冷却: 同一指标 60 分钟内不重复通知
    """
    import psutil

    from djangoadminx.config_center.models import Config
    from djangoadminx.notification.utils import create_notification

    # 从配置中心读取阈值（带行业最佳实践默认值）
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

    # ── CPU 检查 ──
    cpu_percent = psutil.cpu_percent(interval=1)
    if cpu_percent >= thresholds["cpu"]["crit"]:
        alerts.append((
            "error",
            "CPU 使用率严重过高",
            f"当前 CPU 使用率 {cpu_percent}%，"
            f"超过严重阈值 {thresholds['cpu']['crit']}%",
        ))
    elif cpu_percent >= thresholds["cpu"]["warn"]:
        alerts.append((
            "warning",
            "CPU 使用率偏高",
            f"当前 CPU 使用率 {cpu_percent}%，"
            f"超过警告阈值 {thresholds['cpu']['warn']}%",
        ))

    # ── 内存检查 ──
    mem = psutil.virtual_memory()
    mem_percent = mem.percent
    mem_used_gb = mem.used / 1024 ** 3
    mem_total_gb = mem.total / 1024 ** 3
    if mem_percent >= thresholds["mem"]["crit"]:
        alerts.append((
            "error",
            "内存使用率严重过高",
            f"当前内存使用率 {mem_percent:.1f}%"
            f"（已用 {mem_used_gb:.1f}GB / 总计 {mem_total_gb:.1f}GB），"
            f"超过严重阈值 {thresholds['mem']['crit']}%",
        ))
    elif mem_percent >= thresholds["mem"]["warn"]:
        alerts.append((
            "warning",
            "内存使用率偏高",
            f"当前内存使用率 {mem_percent:.1f}%"
            f"（已用 {mem_used_gb:.1f}GB / 总计 {mem_total_gb:.1f}GB），"
            f"超过警告阈值 {thresholds['mem']['warn']}%",
        ))

    # ── 磁盘检查 ──
    try:
        disk = psutil.disk_usage("/")
        disk_percent = disk.percent
        disk_used_gb = disk.used / 1024 ** 3
        disk_total_gb = disk.total / 1024 ** 3
    except Exception as e:
        logger.warning("获取磁盘使用率失败: %s", e)
        disk_percent = 0

    if disk_percent >= thresholds["disk"]["crit"]:
        alerts.append((
            "error",
            "磁盘使用率严重过高",
            f"当前磁盘使用率 {disk_percent:.1f}%"
            f"（已用 {disk_used_gb:.1f}GB / 总计 {disk_total_gb:.1f}GB），"
            f"超过严重阈值 {thresholds['disk']['crit']}%",
        ))
    elif disk_percent >= thresholds["disk"]["warn"]:
        alerts.append((
            "warning",
            "磁盘使用率偏高",
            f"当前磁盘使用率 {disk_percent:.1f}%"
            f"（已用 {disk_used_gb:.1f}GB / 总计 {disk_total_gb:.1f}GB），"
            f"超过警告阈值 {thresholds['disk']['warn']}%",
        ))

    # ── 发送告警通知（带 60 分钟冷却） ──
    from djangoadminx.notification.models import Notification

    cooldown = timezone.now() - timedelta(minutes=60)
    for ntype, title, content in alerts:
        exists = Notification.objects.filter(
            title=title,
            created_at__gte=cooldown,
            user__isnull=True,
        ).exists()
        if not exists:
            create_notification(
                title=title,
                content=content,
                notification_type=ntype,
            )
            logger.warning("资源告警已创建: [%s] %s", ntype, title)

    # ── 返回摘要（存入 JobLog） ──
    return (
        f"CPU: {cpu_percent}% | "
        f"内存: {mem_percent:.1f}% | "
        f"磁盘: {disk_percent:.1f}% | "
        f"告警: {len(alerts)} 条"
    )