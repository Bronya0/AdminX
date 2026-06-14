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

        scheduler_manager.start()  # 仅初始化 scheduler 对象，不加载 job

        # daemon 模式：先 fork，再抢锁 —— 避免子进程继承已启动的 scheduler / Redis 连接
        if options["daemon"]:
            if platform.system() == "Windows":
                self.stderr.write(self.style.ERROR("Windows 不支持守护进程模式"))
                return
            if os.fork():
                sys.exit(0)

        # 抢锁成功才会加载 job + 启动 scheduler；失败则保持 standby
        scheduler_manager.try_become_leader()

        pid = os.getpid()
        role = "leader" if scheduler_manager.leader.is_leader else "standby"
        self.stdout.write(self.style.SUCCESS(
            f"[Scheduler] Running (PID: {pid}, role: {role})"
        ))

        # 主循环：每 10s 续租/抢占 + 写心跳 + 处理重载通知
        # - leader: 续租成功→写心跳+处理通知；续租失败→step_down 转 standby
        # - standby: 尝试抢占，成功则升级为 leader
        tick = 0
        while True:
            time.sleep(1)
            tick += 1
            if tick < 10:
                continue
            tick = 0

            if scheduler_manager.leader.is_leader:
                if scheduler_manager.leader.renew():
                    scheduler_manager.write_heartbeat()
                    scheduler_manager.process_notifications()
                else:
                    # 锁丢失 —— 停止调度器，转 standby 等待重新抢占
                    scheduler_manager.step_down()
            else:
                scheduler_manager.try_become_leader()
