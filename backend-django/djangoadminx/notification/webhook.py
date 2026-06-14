import hashlib
import hmac
import json
import logging

import requests
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
    """通知创建后分发到所有匹配的 webhook"""
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
    """通知创建时自动触发 webhook"""
    if created:
        dispatch(instance)
