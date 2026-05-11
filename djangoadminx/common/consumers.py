import json
import os
import logging

from channels.generic.websocket import AsyncWebsocketConsumer
from django.conf import settings

logger = logging.getLogger("realtime")


class LogConsumer(AsyncWebsocketConsumer):
    """WebSocket 实时日志推送"""

    async def connect(self):
        await self.accept()
        log_file = settings.BASE_DIR / "logs" / "realtime.log"
        await self.tail_log(log_file)

    async def tail_log(self, log_file):
        """异步 tail 日志文件"""
        if not log_file.exists():
            await self.send(text_data=json.dumps({"line": "[日志文件不存在]", "level": "INFO"}))
            await self.close()
            return

        # 读取已有内容
        with open(log_file, "r", encoding="utf-8") as f:
            for line in f:
                await self.send(text_data=json.dumps({"line": line.rstrip()}))

        # 持续 tail
        import asyncio
        with open(log_file, "r", encoding="utf-8") as f:
            f.seek(0, os.SEEK_END)
            while True:
                line = f.readline()
                if line:
                    await self.send(text_data=json.dumps({"line": line.rstrip()}))
                else:
                    await asyncio.sleep(0.5)