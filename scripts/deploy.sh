#!/usr/bin/env bash
# DjangoAdminX 一键部署脚本 — Ubuntu/Debian
# 用法: bash scripts/deploy.sh
set -euo pipefail

APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$APP_DIR"

echo "=========================================="
echo " DjangoAdminX 一键部署"
echo "=========================================="

# ── 1. 系统依赖 ──
echo ""
echo ">>> 安装系统依赖..."
sudo apt-get update -qq
sudo apt-get install -y -qq \
    python3 python3-venv python3-pip \
    redis-server \
    postgresql postgresql-client libpq-dev \
    nginx \
    supervisor

# ── 2. Redis ──
echo ""
echo ">>> 配置 Redis..."
sudo systemctl enable redis-server
sudo systemctl start redis-server
redis-cli ping && echo "Redis 就绪"

# ── 3. PostgreSQL ──
echo ""
echo ">>> 配置 PostgreSQL..."
sudo systemctl enable postgresql
sudo systemctl start postgresql
sudo -u postgres psql -c "CREATE USER djangoadminx WITH PASSWORD 'djangoadminx';" 2>/dev/null || true
sudo -u postgres psql -c "CREATE DATABASE djangoadminx OWNER djangoadminx;" 2>/dev/null || true

# ── 4. Python 虚拟环境 ──
echo ""
echo ">>> 创建虚拟环境..."
python3 -m venv .venv
source .venv/bin/activate
pip install --quiet --upgrade pip
pip install --quiet -r requirements.txt

# ── 5. 环境变量 ──
echo ""
echo ">>> 配置环境变量..."
if [ ! -f .env ]; then
    cat > .env <<EOF
SECRET_KEY=$(python3 -c "from django.core.management.utils import get_random_secret_key; print(get_random_secret_key())")
DEBUG=False
DATABASE_URL=postgres://djangoadminx:djangoadminx@127.0.0.1:5432/djangoadminx
REDIS_URL=redis://127.0.0.1:6379/0
JWT_ACCESS_EXPIRE=30
JWT_REFRESH_EXPIRE=7
THROTTLE_ANON=30/minute
THROTTLE_USER=200/minute
ALLOWED_HOSTS=*
CORS_ALLOWED_ORIGINS=http://localhost:8080
EOF
    echo ".env 已生成"
fi
export DJANGO_SETTINGS_MODULE=config.settings.prod

# ── 6. 初始化 ──
echo ""
echo ">>> 数据库迁移..."
python manage.py migrate --noinput

echo ""
echo ">>> 初始化基础数据..."
python manage.py init_data

echo ""
echo ">>> 收集静态文件..."
python manage.py collectstatic --noinput --clear

# ── 7. Supervisor ──
echo ""
echo ">>> 配置 Supervisor..."
sudo tee /etc/supervisor/conf.d/djangoadminx.conf > /dev/null <<EOF
[program:djangoadminx-gunicorn]
command=$APP_DIR/.venv/bin/gunicorn config.wsgi:application --bind 0.0.0.0:8000 --workers 4 --timeout 120
directory=$APP_DIR
user=$(whoami)
autostart=true
autorestart=true
redirect_stderr=true
stdout_logfile=$APP_DIR/logs/gunicorn.log

[program:djangoadminx-scheduler]
command=$APP_DIR/.venv/bin/python manage.py run_scheduler
directory=$APP_DIR
user=$(whoami)
autostart=true
autorestart=true
redirect_stderr=true
stdout_logfile=$APP_DIR/logs/scheduler.log
EOF

sudo supervisorctl reread
sudo supervisorctl update
sudo supervisorctl start all

# ── 8. Nginx ──
echo ""
echo ">>> 配置 Nginx..."
sudo tee /etc/nginx/sites-available/djangoadminx > /dev/null <<NGINX
upstream django_backend {
    server 127.0.0.1:8000;
}

limit_req_zone \$binary_remote_addr zone=djangoadminx_api:10m rate=10r/s;

server {
    listen 80;
    server_name _;
    client_max_body_size 100M;

    location = /djangoadminx {
        return 301 /djangoadminx/;
    }

    location ^~ /djangoadminx/assets/ {
        alias $APP_DIR/adminx-ui/dist/assets/;
        expires 7d;
        add_header Cache-Control "public";
        try_files \$uri =404;
    }

    location = /djangoadminx/favicon.ico {
        alias $APP_DIR/adminx-ui/dist/favicon.ico;
        expires 7d;
        add_header Cache-Control "public";
    }

    location ^~ /djangoadminx/static/ {
        alias $APP_DIR/staticfiles/;
        expires 7d;
    }

    location ^~ /djangoadminx/media/ {
        alias $APP_DIR/media/;
        expires 30d;
    }

    location ^~ /djangoadminx/api/ {
        limit_req zone=djangoadminx_api burst=20 nodelay;
        limit_req_status 429;
        proxy_pass http://django_backend/api/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location ^~ /djangoadminx/admin/ {
        proxy_pass http://django_backend/admin/;
        proxy_redirect /admin/ /djangoadminx/admin/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location ^~ /djangoadminx/ws/ {
        proxy_pass http://django_backend/ws/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location = /djangoadminx/health/ {
        proxy_pass http://django_backend/api/v1/common/health/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location ^~ /djangoadminx/ {
        alias $APP_DIR/adminx-ui/dist/;
        try_files \$uri \$uri/ /djangoadminx/index.html;
    }
}
NGINX

sudo ln -sf /etc/nginx/sites-available/djangoadminx /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t && sudo systemctl reload nginx

echo ""
echo "=========================================="
echo " 部署完成!"
echo ""
echo " 访问: http://$(curl -s ifconfig.me 2>/dev/null || echo 'localhost')"
echo " Redis: $(redis-cli ping)"
echo " PostgreSQL: 已配置"
echo " Gunicorn: supervisorctl status djangoadminx-gunicorn"
echo " Scheduler: supervisorctl status djangoadminx-scheduler"
echo "=========================================="
