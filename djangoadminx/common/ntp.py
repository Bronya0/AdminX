"""NTP 时间同步工具"""
import logging
import socket
from datetime import datetime, timezone

import ntplib

logger = logging.getLogger("djangoadminx.ntp")

DEFAULT_SERVERS = ["ntp.aliyun.com", "ntp.tencent.com", "pool.ntp.org"]


class NtpClient:
    """NTP 时间同步客户端"""

    def __init__(self, server: str, timeout: int = 5):
        self.server = server
        self.timeout = timeout

    def query(self) -> dict | None:
        """查询 NTP 时间，返回 {offset_seconds, server_time, local_time} 或 None"""
        try:
            client = ntplib.NTPClient()
            resp = client.request(self.server, timeout=self.timeout)
            return {
                "server": self.server,
                "offset": round(resp.offset, 3),
                "delay": round(resp.delay * 1000, 1),
                "server_time": datetime.fromtimestamp(resp.tx_time, tz=timezone.utc).isoformat(),
                "local_time": datetime.now(timezone.utc).isoformat(),
            }
        except (ntplib.NTPException, socket.error, OSError) as e:
            logger.warning("NTP query failed [%s]: %s", self.server, e)
            return None

    def sync(self) -> bool:
        """尝试同步系统时间（需要管理员权限），返回是否成功"""
        result = self.query()
        if result is None:
            return False
        try:
            import os
            import platform
            import subprocess
            import time as _time

            tx_time = result["server_time"]
            dt = datetime.fromisoformat(tx_time)
            ts = dt.strftime("%Y-%m-%d %H:%M:%S")

            system = platform.system()
            if system == "Windows":
                subprocess.run(
                    ["powershell", "-Command", f"Set-Date -Date '{ts}'"],
                    capture_output=True, timeout=10, check=False,
                )
            elif system == "Linux":
                subprocess.run(
                    ["sudo", "date", "-s", ts],
                    capture_output=True, timeout=10, check=False,
                )
            else:
                logger.warning("Unsupported platform for time sync: %s", system)
                return False

            logger.info("System time synced to %s via %s", ts, self.server)
            return True
        except Exception as e:
            logger.error("Failed to set system time: %s", e)
            return False


def sync_time(server: str | None = None) -> dict:
    """统一的 NTP 同步入口，返回 {success, server, offset, ...}"""
    if not server:
        server = DEFAULT_SERVERS[0]
    client = NtpClient(server)
    result = client.query()
    if result is None:
        return {"success": False, "server": server, "error": "无法连接到 NTP 服务器"}
    return {"success": True, **result}
