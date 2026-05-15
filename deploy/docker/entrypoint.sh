#!/bin/bash
set -e

echo ">>> 等待 PostgreSQL..."
until pg_isready -h "$(echo $DATABASE_URL | sed 's/.*@//' | sed 's/:.*//')" -U djangoadminx 2>/dev/null; do
  sleep 1
done
echo ">>> PostgreSQL 就绪"

echo ">>> 等待 Redis..."
until redis-cli -u "$REDIS_URL" ping 2>/dev/null | grep -q PONG; do
  sleep 1
done
echo ">>> Redis 就绪"

echo ">>> 执行数据库迁移..."
python manage.py migrate --noinput

echo ">>> 初始化基础数据..."
python manage.py init_data

echo ">>> 收集静态文件..."
python manage.py collectstatic --noinput --clear

exec "$@"
