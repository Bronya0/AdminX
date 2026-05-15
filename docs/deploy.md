# 部署指南

## 架构

```
                         Nginx (负载均衡)
                       /        |        \
                节点 1       节点 2       节点 3
              (gunicorn)   (gunicorn)   (gunicorn)
                    \         |          /
                    Redis + PostgreSQL
                          |
                   APScheduler (独立进程，只跑一份)
```

### 设计原则

- **无状态应用节点**: JWT 认证，节点间不需要共享 session，水平扩展只需加机器
- **Redis 共享缓存**: 配置缓存、限流计数、APScheduler JobStore、WebSocket 通道层
- **PostgreSQL 统一存储**: 所有节点连同一数据库，ORM 层统一
- **独立调度器**: APScheduler 以独立进程运行，不受 Gunicorn 多 worker 影响，也不会多节点重复执行
- **文件存储抽象**: 支持 Local / MinIO，多节点部署时切到 MinIO 即可共享文件

---

## Docker Compose 部署（推荐）

### 前置条件

- Docker 24+
- Docker Compose v2+

### 启动

```bash
# 克隆
git clone https://github.com/Bronya0/DjangoAdminX.git
cd DjangoAdminX

# 一键启动全部服务
docker compose up -d

# 查看状态
docker compose ps

# 查看日志
docker compose logs -f
```

启动后访问 `http://localhost`。

### 服务组成

| 服务 | 说明 | 外部端口 |
|------|------|---------|
| `nginx` | 反向代理 + 静态文件服务 | 80 |
| `django` | Gunicorn (4 workers) | 8000 |
| `scheduler` | APScheduler 独立进程 | - |
| `redis` | 缓存 + JobStore + ChannelLayer | 6379 |
| `postgres` | 数据库 | 5432 |

### 环境变量

通过 `.env` 或 `docker compose run -e KEY=VAL` 传入:

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SECRET_KEY` | `change-me-in-production` | Django secret key |
| `ALLOWED_HOSTS` | `*` | 允许的主机名 |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:8080` | 前端域名 |

### 多节点扩展

```bash
# 在另一台机器上
docker compose up -d django

# 然后把这台机器的 IP 加到 Nginx upstream
```

---

## 裸机部署 (Ubuntu/Debian)

### 一键脚本

```bash
bash scripts/deploy.sh
```

脚本自动安装: Python 虚拟环境、Redis、PostgreSQL、Nginx、Supervisor，执行迁移和初始化。

### 分步安装

```bash
# 1. 系统依赖
sudo apt-get install -y python3-venv redis-server postgresql nginx supervisor

# 2. 虚拟环境
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

# 3. 配置 .env
cp .env.example .env
# 编辑 .env，修改 SECRET_KEY、DATABASE_URL、REDIS_URL

# 4. 初始化
python manage.py migrate
python manage.py init_data
python manage.py collectstatic --noinput

# 5. 启动服务 (Supervisor)
# 参考 deploy/docker/supervisor.conf 配置 gunicorn + scheduler
```

---

## 集群部署

### 节点拓扑

```
Nginx (主) ─── 节点1 (gunicorn)
    │          ─── 节点2 (gunicorn)
    │          ─── 节点3 (gunicorn)
    │
    └── Redis (主从)
    └── PostgreSQL (主从)
    └── MinIO (文件存储)
    └── Scheduler (只在节点1运行)
```

### Nginx 负载均衡配置

编辑 `deploy/nginx/nginx.conf`，在 upstream 中列出所有节点:

```nginx
upstream django_backend {
    # 负载均衡方式: least_conn / ip_hash / 默认轮询
    least_conn;

    server 10.0.1.10:8000 weight=3;
    server 10.0.1.11:8000 weight=3;
    server 10.0.1.12:8000 weight=3;

    # 健康检查
    keepalive 32;
}
```

### 节点初始化

```bash
# 每台节点:
git clone https://github.com/Bronya0/DjangoAdminX.git
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt

# 迁移和静态文件只在主节点执行一次
# python manage.py migrate
# python manage.py collectstatic

# 启动 gunicorn
gunicorn config.wsgi:application \
    --bind 0.0.0.0:8000 \
    --workers $(nproc) \
    --timeout 120 \
    --access-logfile logs/gunicorn_access.log
```

### Redis 配置

```bash
# /etc/redis/redis.conf
bind 0.0.0.0
requirepass your-redis-password
```

`.env` 中配置:

```
REDIS_URL=redis://:your-redis-password@redis-master:6379/0
```

### 调度器

APScheduler 以独立进程运行，**只在主节点启动一份**:

```bash
# 主节点:
python manage.py run_scheduler

# 通过 Supervisor 管理:
# [program:djangoadminx-scheduler]
# command=/path/to/.venv/bin/python manage.py run_scheduler
```

### 文件存储

多节点部署时使用 MinIO 共享文件:

```
# .env
FILE_STORAGE_BACKEND=minio
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=your-access-key
MINIO_SECRET_KEY=your-secret-key
MINIO_BUCKET=djangoadminx
```

---

## 环境变量参考

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SECRET_KEY` | (自动生成) | Django secret key |
| `DEBUG` | `False` | 调试模式 |
| `DATABASE_URL` | `sqlite:///db.sqlite3` | 数据库连接串 |
| `REDIS_URL` | `redis://127.0.0.1:6379/0` | Redis 连接串 |
| `JWT_ACCESS_EXPIRE` | `30` | Access token 过期分钟 |
| `JWT_REFRESH_EXPIRE` | `7` | Refresh token 过期天数 |
| `THROTTLE_ANON` | `30/minute` | 匿名用户限流 |
| `THROTTLE_USER` | `200/minute` | 认证用户限流 |
| `ALLOWED_HOSTS` | `*` | 允许的主机名 |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:8080` | 跨域来源 |
| `FERNET_KEY` | `None` | 配置加密密钥 |

---

## Supervisor 管理

```bash
# 查看状态
supervisorctl status

# 重启全部
supervisorctl restart all

# 只重启调度器
supervisorctl restart djangoadminx-scheduler

# 只重启后端
supervisorctl restart djangoadminx-gunicorn
```

---

## Nginx 限流配置

```nginx
# 在 http 块
limit_req_zone $binary_remote_addr zone=api:10m rate=30r/s;

# 在 server 块
location /api/ {
    limit_req zone=api burst=20 nodelay;
    proxy_pass http://django_backend;
}
```
