"""
独立进程运行 APScheduler

用法:
  python manage.py run_scheduler          # 前台运行 (开发)
  python manage.py run_scheduler --daemon  # 守护进程 (Linux 生产)
"""
import logging
import os
import platform
import signal
import sys
import time

from django.core.management.base import BaseCommand

logger = logging.getLogger("djangoadminx.scheduler")


class Command(BaseCommand):
    help = "启动 APScheduler 调度器独立进程"

    def add_arguments(self, parser):
        parser.add_argument("--daemon", action="store_true", help="后台守护进程模式（仅 Linux）")

    def handle(self, *args, **options):
        self.stdout.write(self.style.SUCCESS("[Scheduler] Starting APScheduler..."))

        from djangoadminx.common.scheduler import scheduler_manager

        def shutdown_handler(signum, frame):
            self.stdout.write(self.style.WARNING("[Scheduler] Shutting down..."))
            scheduler_manager.shutdown()
            sys.exit(0)

        signal.signal(signal.SIGINT, shutdown_handler)
        if platform.system() != "Windows":
            signal.signal(signal.SIGTERM, shutdown_handler)

        scheduler_manager.start()

        if options["daemon"]:
            if platform.system() == "Windows":
                self.stderr.write(self.style.ERROR("Windows 不支持守护进程模式"))
                return
            if os.fork():
                sys.exit(0)

        pid = os.getpid()
        self.stdout.write(self.style.SUCCESS(f"[Scheduler] Running (PID: {pid})"))

        scheduler_manager.write_heartbeat()
        tick = 0
        while True:
            time.sleep(1)
            tick += 1
            if tick >= 10:
                scheduler_manager.write_heartbeat()
                scheduler_manager.process_notifications()
                tick = 0
