#!/usr/bin/env bash
# Linux 生产部署脚本 — gunicorn 模式
# 使用方式: bash scripts/start-linux.sh
set -euo pipefail

cd "$(dirname "$0")/.."

# 虚拟环境
VENV=".venv"
if [ ! -d "$VENV" ]; then
    echo "[ERROR] 虚拟环境不存在: $VENV"
    echo "请先创建: python3 -m venv $VENV && source $VENV/bin/activate && pip install -r requirements/prod.txt"
    exit 1
fi
source "$VENV/bin/activate"

# 环境变量
export DJANGO_SETTINGS_MODULE="${DJANGO_SETTINGS_MODULE:-config.settings.prod}"

echo ">>> 执行数据库迁移..."
python manage.py migrate --noinput

echo ">>> 收集静态文件..."
python manage.py collectstatic --noinput --clear

echo ">>> 启动 Gunicorn (端口 8000, 4 workers)..."
exec gunicorn config.wsgi:application \
    --bind 0.0.0.0:8000 \
    --workers 4 \
    --timeout 120 \
    --access-logfile logs/gunicorn_access.log \
    --error-logfile logs/gunicorn_error.log \
    --log-level info