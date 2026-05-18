#!/usr/bin/env bash
# 安全停止所有 DjangoAdminX 服务
set -uo pipefail

cd "$(dirname "$0")"

TIMEOUT=15  # 等待优雅退出的秒数

stop_pid_file() {
    local name="$1"
    local pidfile="$2"

    if [ ! -f "$pidfile" ]; then
        echo "[$name] PID 文件不存在，跳过"
        return 0
    fi

    local pid
    pid=$(cat "$pidfile")

    if ! kill -0 "$pid" 2>/dev/null; then
        echo "[$name] 进程 $pid 已不存在，清理 PID 文件"
        rm -f "$pidfile"
        return 0
    fi

    echo "[$name] 发送 SIGTERM 到进程 $pid..."
    kill -TERM "$pid"

    local i=0
    while kill -0 "$pid" 2>/dev/null; do
        if [ $i -ge $TIMEOUT ]; then
            echo "[$name] 超时，强制 SIGKILL..."
            kill -KILL "$pid" 2>/dev/null || true
            break
        fi
        sleep 1
        i=$((i + 1))
    done

    rm -f "$pidfile"
    echo "[$name] 已停止"
}

stop_by_pattern() {
    local name="$1"
    local pattern="$2"

    local pids
    pids=$(pgrep -f "$pattern" 2>/dev/null || true)

    if [ -z "$pids" ]; then
        echo "[$name] 未找到运行中的进程，跳过"
        return 0
    fi

    echo "[$name] 发送 SIGTERM 到进程 $pids..."
    echo "$pids" | xargs kill -TERM 2>/dev/null || true

    local i=0
    while pgrep -f "$pattern" > /dev/null 2>&1; do
        if [ $i -ge $TIMEOUT ]; then
            echo "[$name] 超时，强制 SIGKILL..."
            pgrep -f "$pattern" | xargs kill -KILL 2>/dev/null || true
            break
        fi
        sleep 1
        i=$((i + 1))
    done

    echo "[$name] 已停止"
}

echo ">>> 停止 Gunicorn..."
stop_pid_file "gunicorn" "logs/gunicorn.pid"

echo ">>> 停止反向代理..."
stop_pid_file "proxy" "logs/proxy.pid"

echo ">>> 停止调度器..."
stop_by_pattern "scheduler" "manage.py run_scheduler"

echo ""
echo "所有服务已停止"
