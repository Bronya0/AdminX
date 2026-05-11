"""
demo_blog AppConfig

业务 app 的标准写法：
  1. name 使用 "apps.{app_name}" 前缀，与框架代码 (djangoadminx/) 区分
  2. verbose_name 使用中文
  3. ready() 中连接审计信号
"""
import logging

from django.apps import AppConfig

logger = logging.getLogger("apps.demo_blog")


class DemoBlogConfig(AppConfig):
    name = "apps.demo_blog"
    verbose_name = "示例博客"

    def ready(self):
        """
        应用启动时注册审计信号。
        框架的审计信号处理器在 djangoadminx.audit.signals 中定义，
        通过 post_save / pre_delete 自动记录所有核心模型的变更。
        """
        try:
            from django.db.models.signals import post_save, pre_delete
            from djangoadminx.audit.signals import _audit_save, _audit_delete

            Category = self.get_model("Category")
            Post = self.get_model("Post")

            post_save.connect(_audit_save, sender=Category, weak=False, dispatch_uid="audit_save_demo_category")
            pre_delete.connect(_audit_delete, sender=Category, weak=False, dispatch_uid="audit_delete_demo_category")
            post_save.connect(_audit_save, sender=Post, weak=False, dispatch_uid="audit_save_demo_post")
            pre_delete.connect(_audit_delete, sender=Post, weak=False, dispatch_uid="audit_delete_demo_post")

            logger.debug("Audit signals connected for demo_blog")
        except Exception as e:
            logger.warning(f"Audit signal registration skipped: {e}")