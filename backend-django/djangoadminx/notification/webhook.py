import hashlib
import hmac
import json
import logging
import threading

import requests
from django.db import transaction
from django.db.models.signals import post_save
from django.dispatch import receiver

from .models import Notification, WebhookConfig, WebhookLog

logger = logging.getLogger("djangoadminx.notification")


def send_webhook(config, notification):
    """发送单条 webhook"""
    payload = {
        "event": notification.notification_type,
        "title": notification.title,
        "content": notification.content,
        "created_at": notification.created_at.isoformat(),
    }
    body = json.dumps(payload, ensure_ascii=False)
    headers = {"Content-Type": "application/json"}
    if config.secret:
        sig = hmac.new(config.secret.encode(), body.encode(), hashlib.sha256).hexdigest()
        headers["X-Signature"] = sig
    try:
        resp = requests.post(config.url, data=body, headers=headers, timeout=10)
        _log_result(config, notification, "success", resp.status_code, resp.text)
        logger.info(f"Webhook sent: {config.name} -> {resp.status_code}")
    except Exception as e:
        _log_result(config, notification, "failed", error_message=str(e))
        logger.warning(f"Webhook failed: {config.name} -> {e}")


def _log_result(config, notification, status, response_status=None, response_body="", error_message=""):
    WebhookLog.objects.create(
        webhook=config,
        notification=notification,
        status=status,
        response_status=response_status,
        response_body=(response_body or "")[:2000],
        error_message=error_message,
    )


def dispatch(notification):
    """通知创建后分发到所有匹配的 webhook。

    在独立线程中执行，避免阻塞创建通知的请求线程（webhook 端点响应慢会拖垮整个 API）。
    使用 daemon 线程，进程退出时自动结束。
    """
    t = threading.Thread(target=_dispatch_sync, args=(notification.pk,), daemon=True)
    t.start()


def _dispatch_sync(notification_pk):
    """实际执行分发（在工作线程中运行）。

    通过 pk 重新查询 notification，避免跨线程传递 ORM 对象（且原对象此时可能
    已不在请求上下文，调用方持有的 instance 引用在信号处理后才最终落库）。
    """
    try:
        notification = Notification.objects.get(pk=notification_pk)
    except Notification.DoesNotExist:
        logger.warning(f"Webhook 分发失败: notification {notification_pk} 不存在")
        return

    configs = WebhookConfig.objects.filter(is_active=True)
    for config in configs:
        if not _matches_events(config.events, notification.notification_type):
            continue
        send_webhook(config, notification)


def _matches_events(events_csv, notification_type):
    if not events_csv:
        return True
    events = [e.strip() for e in events_csv.split(",") if e.strip()]
    return notification_type in events


@receiver(post_save, sender=Notification)
def on_notification_created(sender, instance, created, **kwargs):
    """通知创建时自动触发 webhook。

    注意：post_save 在事务内触发，此时若创建通知的请求回滚，webhook 已发出会造成"幽灵通知"。
    使用 transaction.on_commit 确保只在事务成功提交后才派发，与 DB 状态一致。
    """
    if not created:
        return
    # on_commit 回调在事务提交后执行；非事务环境（atomic 关闭）会立即执行
    transaction.on_commit(lambda: dispatch(instance))
