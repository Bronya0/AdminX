"""通知中心工具函数"""

from .models import Notification


def create_notification(
    title: str,
    content: str = "",
    notification_type: str = "info",
    user=None,
) -> Notification:
    """创建系统通知

    Args:
        title: 通知标题
        content: 通知内容
        notification_type: 通知类型 (info/success/warning/error)
        user: 接收用户，None=全员通知

    Returns:
        Notification 实例
    """
    return Notification.objects.create(
        title=title,
        content=content,
        notification_type=notification_type,
        user=user,
    )
