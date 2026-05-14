from django.apps import AppConfig


class AuditConfig(AppConfig):
    name = "djangoadminx.audit"
    verbose_name = "审计日志"

    def ready(self):
        from . import signals  # noqa: F401 — 为业务模块提供 _audit_save/_audit_delete 工具
        # 框架模型改用 AuditLogMixin 显式记录，不再走信号