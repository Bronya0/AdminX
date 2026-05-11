"""
定时任务处理函数 — 被 APScheduler 动态导入调用

最佳实践:
  1. 每个函数都是独立入口，不依赖 request 上下文
  2. 通过 ScheduleJob 配置的 handler 路径调用: apps.demo_blog.tasks.archive_old_posts
  3. 返回序列化数据给 JobLog
"""
import logging
from datetime import timedelta

from django.utils import timezone

logger = logging.getLogger("apps.demo_blog.tasks")


def archive_old_posts(days: int = 180) -> str:
    """
    归档超过 N 天未更新的草稿文章。
    可被 APScheduler 定时调用。

    配置示例 (通过 ScheduleJob API):
      handler: apps.demo_blog.tasks.archive_old_posts
      trigger_type: cron
      trigger_config: {"day_of_week": "mon", "hour": 3, "minute": 0}
      args: [180]
    """
    from .models import Post

    cutoff = timezone.now() - timedelta(days=days)
    qs = Post.objects.filter(status=Post.STATUS_DRAFT, updated_at__lt=cutoff)
    count = qs.count()
    qs.update(status=Post.STATUS_ARCHIVED)
    logger.info(f"Archived {count} old draft posts")
    return f"archived {count} posts"


def post_statistics_summary() -> dict:
    """
    生成文章统计摘要（供 WSDL 或定时任务调用）。
    """
    from .models import Category, Post

    return {
        "total_posts": Post.objects.count(),
        "total_categories": Category.objects.count(),
        "published": Post.objects.filter(status=Post.STATUS_PUBLISHED).count(),
        "draft": Post.objects.filter(status=Post.STATUS_DRAFT).count(),
    }