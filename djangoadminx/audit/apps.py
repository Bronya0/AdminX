from django.apps import AppConfig


class AuditConfig(AppConfig):
    name = "djangoadminx.audit"
    verbose_name = "审计日志"

    def ready(self):
        from . import signals  # noqa: F401