"""
审计日志信号 — 自动记录核心模型的增删改

通过 thread-local 获取当前请求的用户（由 RequestContextMiddleware 注入）。
"""
import json
import logging
import threading
from copy import copy

logger = logging.getLogger("djangoadminx.audit")

# thread-local 存储当前请求
_request_local = threading.local()


def set_current_request(request):
    """中间件调用 — 将当前请求存入 thread-local"""
    _request_local.request = request


def get_current_request():
    """从 thread-local 获取当前请求"""
    return getattr(_request_local, "request", None)


def _serialize_for_json(obj):
    """将模型实例转为 JSON 可序列化的 dict"""
    result = {}
    for field in obj._meta.fields:
        val = getattr(obj, field.name)
        if hasattr(val, "isoformat"):
            val = val.isoformat() if val else None
        elif hasattr(val, "pk"):
            val = str(val.pk) if val and val.pk else None
        elif isinstance(val, (__import__("uuid").UUID,)):
            val = str(val)
        result[field.name] = val
    return result


def _get_audit_models():
    """获取需要审计的模型列表"""
    from django.conf import settings

    return getattr(settings, "AUDIT_MODELS", None) or [
        "accounts.User",
        "accounts.Role",
        "menu.Menu",
        "config_center.Config",
        "cluster.ClusterNode",
        "webservice.WebService",
        "webservice.ScheduleJob",
    ]


def _import_model(model_path):
    """按 'app_label.ModelName' 导入模型"""
    from django.apps import apps
    try:
        app_label, model_name = model_path.rsplit(".", 1)
        return apps.get_model(app_label, model_name)
    except (LookupError, ValueError):
        return None


class AuditSignalManager:
    """审计信号管理器 — 注册所有需审计模型的信号"""

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


def _get_operator():
    """从 thread-local request 获取操作人"""
    request = get_current_request()
    if request and hasattr(request, "user") and request.user.is_authenticated:
        return request.user.username or str(request.user)
    return "system"


def _get_operator_ip():
    request = get_current_request()
    if request:
        return request.META.get("REMOTE_ADDR", "")
    return ""


def _audit_save(sender, instance, created, **kwargs):
    """post_save — 记录 create/update"""
    from .models import AuditLog

    if created:
        new_vals = _serialize_for_json(instance)
        AuditLog.objects.create(
            action=AuditLog.ActionChoices.CREATE,
            model_name=instance._meta.label,
            object_id=str(instance.pk),
            object_repr=str(instance)[:255],
            operator=_get_operator(),
            operator_ip=_get_operator_ip(),
            new_values=new_vals,
            diff_summary=f"创建 {instance._meta.verbose_name}",
        )
    else:
        try:
            old_obj = sender.objects.get(pk=instance.pk)
            old_vals = _serialize_for_json(old_obj)
            new_vals = _serialize_for_json(instance)
        except sender.DoesNotExist:
            return

        diffs = {}
        for key in new_vals:
            if key in ("updated_at", "last_login",):
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
            operator=_get_operator(),
            operator_ip=_get_operator_ip(),
            old_values={k: v["old"] for k, v in diffs.items()},
            new_values={k: v["new"] for k, v in diffs.items()},
            diff_summary=f"更新 {instance._meta.verbose_name}: {changed_fields}",
        )


def _audit_delete(sender, instance, **kwargs):
    """pre_delete — 记录 delete"""
    from .models import AuditLog

    old_vals = _serialize_for_json(instance)
    AuditLog.objects.create(
        action=AuditLog.ActionChoices.DELETE,
        model_name=instance._meta.label,
        object_id=str(instance.pk),
        object_repr=str(instance)[:255],
        operator=_get_operator(),
        operator_ip=_get_operator_ip(),
        old_values=old_vals,
        diff_summary=f"删除 {instance._meta.verbose_name}: {instance}",
    )