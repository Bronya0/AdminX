import time
from collections import defaultdict
from datetime import timedelta

import psutil
from django.utils import timezone


HISTORY_RANGES = {
    "1h": 3600,
    "6h": 6 * 3600,
    "24h": 24 * 3600,
    "7d": 7 * 24 * 3600,
}

HISTORY_INTERVALS = {
    "1m": 60,
    "5m": 5 * 60,
    "15m": 15 * 60,
    "1h": 3600,
}

RECOMMENDED_INTERVALS = {
    "1h": "1m",
    "6h": "5m",
    "24h": "15m",
    "7d": "1h",
}

SNAPSHOT_RETENTION_DAYS = 30


class SystemMonitor:
    """系统资源监控"""

    @staticmethod
    def cpu(sample_interval=0):
        return {
            "percent": psutil.cpu_percent(interval=sample_interval),
            "count": psutil.cpu_count(),
            "freq": psutil.cpu_freq()._asdict() if psutil.cpu_freq() else {},
        }

    @staticmethod
    def memory():
        return psutil.virtual_memory()._asdict()

    @staticmethod
    def disk():
        partitions = []
        for partition in psutil.disk_partitions():
            try:
                usage = psutil.disk_usage(partition.mountpoint)
                partitions.append({
                    "device": partition.device,
                    "mountpoint": partition.mountpoint,
                    "fstype": partition.fstype,
                    "total": usage.total,
                    "used": usage.used,
                    "free": usage.free,
                    "percent": usage.percent,
                })
            except PermissionError:
                continue
        return partitions

    @staticmethod
    def disk_io():
        io = psutil.disk_io_counters()
        return io._asdict() if io else {}

    @staticmethod
    def network_io():
        io = psutil.net_io_counters()
        return io._asdict() if io else {}

    @staticmethod
    def netstat():
        try:
            conns = psutil.net_connections()
        except psutil.AccessDenied:
            return {"error": "权限不足，无法获取连接状态"}
        stats = {"LISTEN": 0, "ESTABLISHED": 0, "TIME_WAIT": 0, "CLOSE_WAIT": 0, "OTHER": 0}
        for conn in conns:
            status = conn.status if conn.status else "OTHER"
            if status in stats:
                stats[status] += 1
            else:
                stats["OTHER"] += 1
        return stats

    @staticmethod
    def load_avg():
        try:
            return psutil.getloadavg()
        except Exception:
            return None

    @staticmethod
    def aggregate_disk_usage(partitions):
        if not partitions:
            return {"total": 0, "used": 0, "free": 0, "percent": 0}

        total = sum(item["total"] for item in partitions)
        used = sum(item["used"] for item in partitions)
        free = sum(item["free"] for item in partitions)
        percent = (used / total * 100) if total else 0
        return {
            "total": total,
            "used": used,
            "free": free,
            "percent": round(percent, 2),
        }

    @classmethod
    def snapshot(cls, sample_interval=0):
        cpu = cls.cpu(sample_interval=sample_interval)
        memory = cls.memory()
        partitions = cls.disk()
        disk_io = cls.disk_io()

        return {
            "cpu": {key: value for key, value in cpu.items() if key != "freq"},
            "memory": memory,
            "disk": partitions,
            "disk_usage": cls.aggregate_disk_usage(partitions),
            "disk_io": disk_io,
            "network_io": cls.network_io(),
            "load_avg": cls.load_avg(),
            "timestamp": time.time(),
        }

    @classmethod
    def all(cls):
        snapshot = cls.snapshot(sample_interval=0)
        snapshot.pop("disk_usage", None)
        return snapshot

    @classmethod
    def record_snapshot(cls, sample_interval=1):
        from .models import SystemMetricSnapshot

        snapshot = cls.snapshot(sample_interval=sample_interval)
        disk_io = snapshot.get("disk_io") or {}
        disk_usage = snapshot.get("disk_usage") or {}

        record = SystemMetricSnapshot.objects.create(
            collected_at=timezone.now(),
            cpu_percent=snapshot["cpu"].get("percent", 0),
            memory_percent=snapshot["memory"].get("percent", 0),
            disk_percent=disk_usage.get("percent", 0),
            disk_read_bytes=disk_io.get("read_bytes", 0),
            disk_write_bytes=disk_io.get("write_bytes", 0),
        )

        now = timezone.now()
        if now.hour == 0 and now.minute < 5:
            SystemMetricSnapshot.objects.filter(
                collected_at__lt=now - timedelta(days=SNAPSHOT_RETENTION_DAYS)
            ).delete()

        return record, snapshot

    @staticmethod
    def _resolve_range(range_key):
        if range_key not in HISTORY_RANGES:
            raise ValueError("不支持的时间范围")
        return HISTORY_RANGES[range_key]

    @staticmethod
    def _resolve_interval(range_key, interval_key):
        if not interval_key or interval_key == "auto":
            interval_key = RECOMMENDED_INTERVALS[range_key]
        if interval_key not in HISTORY_INTERVALS:
            raise ValueError("不支持的聚合间隔")
        return interval_key, HISTORY_INTERVALS[interval_key]

    @classmethod
    def history(cls, range_key="1h", interval_key="auto"):
        from .models import SystemMetricSnapshot

        range_seconds = cls._resolve_range(range_key)
        resolved_interval_key, interval_seconds = cls._resolve_interval(range_key, interval_key)

        end_at = timezone.now()
        start_at = end_at - timedelta(seconds=range_seconds)
        baseline_at = start_at - timedelta(seconds=interval_seconds)

        snapshots = list(
            SystemMetricSnapshot.objects.filter(collected_at__gte=baseline_at, collected_at__lte=end_at)
            .order_by("collected_at")
            .values(
                "collected_at",
                "cpu_percent",
                "memory_percent",
                "disk_percent",
                "disk_read_bytes",
                "disk_write_bytes",
            )
        )

        buckets = defaultdict(lambda: {
            "cpu_sum": 0.0,
            "memory_sum": 0.0,
            "disk_sum": 0.0,
            "count": 0,
            "last": None,
        })

        start_ts = start_at.timestamp()
        baseline = None
        for item in snapshots:
            ts = item["collected_at"].timestamp()
            if ts < start_ts:
                baseline = item
                continue

            bucket_index = int((ts - start_ts) // interval_seconds)
            bucket = buckets[bucket_index]
            bucket["cpu_sum"] += item["cpu_percent"]
            bucket["memory_sum"] += item["memory_percent"]
            bucket["disk_sum"] += item["disk_percent"]
            bucket["count"] += 1
            bucket["last"] = item

        points = []
        previous = baseline
        total_buckets = int(range_seconds // interval_seconds)
        if range_seconds % interval_seconds:
            total_buckets += 1

        for bucket_index in range(total_buckets):
            bucket = buckets.get(bucket_index)
            if not bucket or not bucket["count"]:
                continue

            current = bucket["last"]
            elapsed = 0
            read_mbps = 0.0
            write_mbps = 0.0

            if previous:
                elapsed = (current["collected_at"] - previous["collected_at"]).total_seconds()
            if previous and elapsed > 0:
                read_delta = max(current["disk_read_bytes"] - previous["disk_read_bytes"], 0)
                write_delta = max(current["disk_write_bytes"] - previous["disk_write_bytes"], 0)
                read_mbps = round(read_delta / elapsed / 1024 / 1024, 3)
                write_mbps = round(write_delta / elapsed / 1024 / 1024, 3)

            points.append({
                "timestamp": current["collected_at"].timestamp(),
                "cpu_percent": round(bucket["cpu_sum"] / bucket["count"], 2),
                "memory_percent": round(bucket["memory_sum"] / bucket["count"], 2),
                "disk_percent": round(bucket["disk_sum"] / bucket["count"], 2),
                "disk_read_mbps": read_mbps,
                "disk_write_mbps": write_mbps,
            })
            previous = current

        return {
            "range": range_key,
            "interval": resolved_interval_key,
            "from": start_at.timestamp(),
            "to": end_at.timestamp(),
            "points": points,
        }
