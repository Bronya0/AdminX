"""
审计日志信号 — 自动记录核心模型的增删改
"""
import json
import logging
import threading

from django.conf import settings

from .utils import get_operator, get_operator_ip, serialize_for_json

logger = logging.getLogger("djangoadminx.audit")

_request_local = threading.local()


def set_current_request(request):
    _request_local.request = request


def get_current_request():
    return getattr(_request_local, "request", None)


def _get_audit_models():
    raw = getattr(settings, "AUDIT_MODELS", None)
    if raw is None:
        # 这些模型由 AuditLogMixin 记录，信号不重复记录
        return []
    return raw if raw else []


def _import_model(model_path):
    from django.apps import apps
    try:
        app_label, model_name = model_path.rsplit(".", 1)
        return apps.get_model(app_label, model_name)
    except (LookupError, ValueError):
        return None


class AuditSignalManager:
    """审计信号管理器"""

    _registered = False

    @classmethod
    def register(cls):
        if cls._registered:
            return
        cls._registered = True

        for model_path in _get_audit_models():
            model = _import_model(model_path)
            if model is None:
                logger.warning(f"审计模型未找到: {model_path}")
                continue

            from django.db.models.signals import post_save, pre_delete

            uid_save = f"audit_save_{model_path}"
            uid_delete = f"audit_delete_{model_path}"

            post_save.connect(_audit_save, sender=model, weak=False, dispatch_uid=uid_save)
            pre_delete.connect(_audit_delete, sender=model, weak=False, dispatch_uid=uid_delete)


def _audit_save(sender, instance, created, **kwargs):
    """post_save — 记录 create/update"""
    from .models import AuditLog

    request = get_current_request()
    operator = get_operator(request)
    operator_ip = get_operator_ip(request)

    if created:
        new_vals = serialize_for_json(instance)
        AuditLog.objects.create(
            action=AuditLog.ActionChoices.CREATE,
            model_name=instance._meta.label,
            object_id=str(instance.pk),
            object_repr=str(instance)[:255],
            operator=operator,
            operator_ip=operator_ip,
            new_values=new_vals,
            diff_summary=f"创建 {instance._meta.verbose_name}",
        )
    else:
        try:
            old_obj = sender.objects.get(pk=instance.pk)
            old_vals = serialize_for_json(old_obj)
            new_vals = serialize_for_json(instance)
        except sender.DoesNotExist:
            return

        diffs = {}
        for key in new_vals:
            if key in ("updated_at", "last_login"):
                continue
            try:
                old_str = json.dumps(old_vals.get(key), ensure_ascii=False, default=str)
                new_str = json.dumps(new_vals[key], ensure_ascii=False, default=str)
            except Exception:
                continue
            if old_str != new_str:
                diffs[key] = {"old": old_vals.get(key), "new": new_vals[key]}

        if not diffs:
            return

        changed_fields = ", ".join(diffs.keys())[:500]
        AuditLog.objects.create(
            action=AuditLog.ActionChoices.UPDATE,
            model_name=instance._meta.label,
            object_id=str(instance.pk),
            object_repr=str(instance)[:255],
            operator=operator,
            operator_ip=operator_ip,
            old_values={k: v["old"] for k, v in diffs.items()},
            new_values={k: v["new"] for k, v in diffs.items()},
            diff_summary=f"更新 {instance._meta.verbose_name}: {changed_fields}",
        )


def _audit_delete(sender, instance, **kwargs):
    """pre_delete — 记录 delete"""
    from .models import AuditLog

    request = get_current_request()
    old_vals = serialize_for_json(instance)
    AuditLog.objects.create(
        action=AuditLog.ActionChoices.DELETE,
        model_name=instance._meta.label,
        object_id=str(instance.pk),
        object_repr=str(instance)[:255],
        operator=get_operator(request),
        operator_ip=get_operator_ip(request),
        old_values=old_vals,
        diff_summary=f"删除 {instance._meta.verbose_name}: {instance}",
    )
