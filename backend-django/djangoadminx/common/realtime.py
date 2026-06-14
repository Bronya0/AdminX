"""实时日志/事件广播 — 基于 Channels Channel Layer。

集群下任意 web 节点调用 broadcast_log()，所有订阅 "logs" group 的
LogConsumer（无论在哪台机器）都会收到消息并转发给浏览器。

- 配置了 channels_redis（生产）：跨节点广播，所有浏览器共享同一份实时流。
- 配置了 InMemoryChannelLayer（本地无 Redis）：仅进程内广播，不报错。
- Channel Layer 未配置：静默跳过。

用法（同步代码）:
    from djangoadminx.common.realtime import broadcast_log_sync
    broadcast_log_sync("任务执行完成", level="INFO")

用法（异步代码）:
    from djangoadminx.common.realtime import broadcast_log
    await broadcast_log("任务执行完成")
"""
import logging

from asgiref.sync import async_to_sync

logger = logging.getLogger("djangoadminx.realtime")

LOGS_GROUP = "logs"


async def broadcast_log(line: str, level: str = "INFO"):
    """向 'logs' group 广播一条日志。

    所有订阅该 group 的 LogConsumer 的 log_message handler 会被触发，
    再由它把 {"line", "level"} 发给浏览器。
    """
    from channels.layers import get_channel_layer

    layer = get_channel_layer()
    if layer is None:
        return  # Channel Layer 未配置，静默跳过
    try:
        await layer.group_send(
            LOGS_GROUP,
            {"type": "log.message", "line": line, "level": level},
        )
    except Exception as e:
        logger.warning(f"广播日志失败: {e}")


# 同步封装 — 给 views/signals/scheduler 等同步上下文用
broadcast_log_sync = async_to_sync(broadcast_log)
