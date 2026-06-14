"""WebSocket 实时日志/事件推送。

集群模式下基于 Channels Channel Layer 的 group 广播：
任意 web 节点调用 djangoadminx.common.realtime.broadcast_log[_sync]()
都会把消息推给所有订阅 "logs" group 的 LogConsumer，
由其 log_message handler 转发给浏览器。

消息格式（保持与历史版本一致）: {"line": str, "level": str}
"""
import json

from channels.generic.websocket import AsyncWebsocketConsumer

from .realtime import LOGS_GROUP


class LogConsumer(AsyncWebsocketConsumer):
    """实时日志 WebSocket 消费者"""

    async def connect(self):
        # 加入广播 group —— 集群下任意节点的 broadcast_log 都会送达
        await self.channel_layer.group_add(LOGS_GROUP, self.channel_name)
        await self.accept()

    async def disconnect(self, code):
        await self.channel_layer.group_discard(LOGS_GROUP, self.channel_name)

    async def log_message(self, event):
        """group_send 的 type='log.message' 路由到这里。

        event 由 realtime.broadcast_log 构造: {line, level}
        转发为与历史版本一致的 JSON 协议给浏览器。
        """
        await self.send(text_data=json.dumps({
            "line": event.get("line", ""),
            "level": event.get("level", "INFO"),
        }))
