"""
独立进程运行 APScheduler

用法:
  python manage.py run_scheduler          # 前台运行 (开发)
  python manage.py run_scheduler --daemon  # 守护进程 (生产, 配合 supervisor)
"""
import logging
import signal
import sys
import time

from django.core.management.base import BaseCommand

logger = logging.getLogger("djangoadminx.scheduler")


class Command(BaseCommand):
    help = "启动 APScheduler 调度器独立进程"

    def add_arguments(self, parser):
        parser.add_argument("--daemon", action="store_true", help="后台守护进程模式")

    def handle(self, *args, **options):
        self.stdout.write(self.style.SUCCESS("[Scheduler] Starting APScheduler..."))

        from djangoadminx.common.scheduler import scheduler_manager

        # 注册优雅退出
        def shutdown_handler(signum, frame):
            self.stdout.write(self.style.WARNING("[Scheduler] Shutting down..."))
            scheduler_manager.shutdown()
            sys.exit(0)

        signal.signal(signal.SIGINT, shutdown_handler)
        signal.signal(signal.SIGTERM, shutdown_handler)

        scheduler_manager.start()

        if options["daemon"]:
            import os
            if os.fork():
                sys.exit(0)

        pid = scheduler_manager.scheduler._pid if hasattr(scheduler_manager.scheduler, '_pid') else 'N/A'
        self.stdout.write(self.style.SUCCESS(f"[Scheduler] Running (PID: {pid})"))

        # 保持进程存活，同时定期写入心跳 + 处理通知
        scheduler_manager.write_heartbeat()
        tick = 0
        while True:
            time.sleep(1)
            tick += 1
            if tick >= 10:  # 每 10 秒一次综合操作
                scheduler_manager.write_heartbeat()
                scheduler_manager.process_notifications()
                tick = 0