"""
审计日志 Mixin — ViewSet 中显式调用记录增删改

用法:
  标准 ViewSet（自动接入）:
    class MyViewSet(AuditLogMixin, viewsets.ModelViewSet):
        ...

  需要自定义 perform_create 的 ViewSet:
    class MyViewSet(AuditLogMixin, viewsets.ModelViewSet):
        def perform_create(self, serializer):
            instance = serializer.save(...)
            self._log_audit_create(instance)
"""
import json
import logging

from .utils import get_operator, get_operator_ip, serialize_for_json

logger = logging.getLogger("djangoadminx.audit")


class AuditLogMixin:
    """审计日志 Mixin — 自动记录增删改"""

    def perform_create(self, serializer):
        instance = serializer.save()
        self._log_audit_create(instance)

    def perform_update(self, serializer):
        old_vals = serialize_for_json(serializer.instance)
        instance = serializer.save()
        new_vals = serialize_for_json(instance)
        self._log_audit_update(instance, old_vals, new_vals)

    def perform_destroy(self, instance):
        old_vals = serialize_for_json(instance)
        super().perform_destroy(instance)
        self._log_audit_delete(instance, old_vals)

    def _log_audit_create(self, instance):
        from .models import AuditLog

        request = getattr(self, "request", None)
        new_vals = serialize_for_json(instance)
        AuditLog.objects.create(
            action=AuditLog.ActionChoices.CREATE,
            model_name=instance._meta.label,
            object_id=str(instance.pk),
            object_repr=str(instance)[:255],
            operator=get_operator(request),
            operator_ip=get_operator_ip(request),
            new_values=new_vals,
            diff_summary=f"创建 {instance._meta.verbose_name}",
        )

    def _log_audit_update(self, instance, old_vals, new_vals):
        from .models import AuditLog

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
        request = getattr(self, "request", None)
        AuditLog.objects.create(
            action=AuditLog.ActionChoices.UPDATE,
            model_name=instance._meta.label,
            object_id=str(instance.pk),
            object_repr=str(instance)[:255],
            operator=get_operator(request),
            operator_ip=get_operator_ip(request),
            old_values={k: v["old"] for k, v in diffs.items()},
            new_values={k: v["new"] for k, v in diffs.items()},
            diff_summary=f"更新 {instance._meta.verbose_name}: {changed_fields}",
        )

    def _log_audit_delete(self, instance, old_vals):
        from .models import AuditLog

        request = getattr(self, "request", None)
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
