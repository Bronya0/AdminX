import time
from functools import lru_cache

import psutil


class SystemMonitor:
    """系统资源监控"""

    @staticmethod
    def cpu():
        return {
            "percent": psutil.cpu_percent(interval=1),
            "count": psutil.cpu_count(),
            "freq": psutil.cpu_freq()._asdict() if psutil.cpu_freq() else {},
        }

    @staticmethod
    def memory():
        return psutil.virtual_memory()._asdict()

    @staticmethod
    def disk():
        partitions = []
        for p in psutil.disk_partitions():
            try:
                usage = psutil.disk_usage(p.mountpoint)
                partitions.append({
                    "device": p.device,
                    "mountpoint": p.mountpoint,
                    "fstype": p.fstype,
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
        return psutil.disk_io_counters()._asdict()

    @staticmethod
    def network_io():
        return psutil.net_io_counters()._asdict()

    @staticmethod
    def netstat():
        conns = psutil.net_connections()
        stats = {"LISTEN": 0, "ESTABLISHED": 0, "TIME_WAIT": 0, "CLOSE_WAIT": 0, "OTHER": 0}
        for c in conns:
            st = c.status if c.status else "OTHER"
            if st in stats:
                stats[st] += 1
            else:
                stats["OTHER"] += 1
        return stats

    @staticmethod
    def load_avg():
        try:
            return psutil.getloadavg()
        except Exception:
            return None

    @classmethod
    def all(cls):
        cpu = cls.cpu()
        return {
            "cpu": {k: v for k, v in cpu.items() if k != "freq"},
            "memory": cls.memory(),
            "disk": cls.disk(),
            "disk_io": cls.disk_io(),
            "network_io": cls.network_io(),
            "load_avg": cls.load_avg(),
            "timestamp": time.time(),
        }