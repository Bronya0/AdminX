#!/usr/bin/env bash
# Linux 生产部署脚本 — gunicorn 模式
# 使用方式: bash start-linux.sh
set -euo pipefail

cd "$(dirname "$0")"

# 虚拟环境
VENV=".venv"
if [ ! -d "$VENV" ]; then
    echo "[ERROR] 虚拟环境不存在: $VENV"
    echo "请先创建: python3 -m venv $VENV && source $VENV/bin/activate && pip install -r requirements.txt"
    exit 1
fi
source "$VENV/bin/activate"

# 环境变量
export DJANGO_SETTINGS_MODULE="${DJANGO_SETTINGS_MODULE:-config.settings.prod}"

echo ">>> 确保日志目录存在..."
mkdir -p logs

echo ">>> 启动调度器进程（后台）..."
if nohup python manage.py run_scheduler > logs/scheduler.log 2>&1 &
then
    SCHEDULER_PID=$!
    echo "调度器已启动 (PID: $SCHEDULER_PID)"
else
    echo "[WARNING] 调度器启动失败，请检查 logs/scheduler.log"
fi

echo ">>> 执行数据库迁移..."
python manage.py migrate --noinput

echo ">>> 收集静态文件..."
python manage.py collectstatic --noinput --clear

echo ">>> 启动 Gunicorn (端口 9999, 4 workers, 后台)..."
gunicorn config.wsgi:application \
    --bind 0.0.0.0:9999 \
    --workers 4 \
    --timeout 120 \
    --daemon \
    --pid logs/gunicorn.pid \
    --access-logfile logs/gunicorn_access.log \
    --error-logfile logs/gunicorn_error.log \
    --log-level info
echo "Gunicorn 已启动 (PID: $(cat logs/gunicorn.pid))"

echo ">>> 启动反向代理 (端口 8888, 后台)..."
nohup python proxy.py --port 8888 --backend http://127.0.0.1:9999 \
    > logs/proxy.log 2>&1 &
echo $! > logs/proxy.pid
echo "代理已启动 (PID: $(cat logs/proxy.pid))"

echo ""
echo "所有服务已启动，停止方式:"
echo "  kill \$(cat logs/gunicorn.pid)   # 停 Gunicorn"
echo "  kill \$(cat logs/proxy.pid)      # 停代理"
echo "  kill \$(pgrep -f run_scheduler)  # 停调度器"