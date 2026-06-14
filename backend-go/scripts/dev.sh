#!/usr/bin/env bash
# 开发模式启动：直接 go run，前台运行便于看日志
# 用法: bash scripts/dev.sh [--scheduler]
set -euo pipefail
cd "$(dirname "$0")/.."

echo ">>> 检查配置文件..."
if [ ! -f configs/config.yaml ]; then
    echo "[!] configs/config.yaml 不存在，从 example 复制"
    cp configs/config.example.yaml configs/config.yaml
    echo "[!] 请编辑 configs/config.yaml 填入实际数据库/Redis 配置后重新运行"
    exit 1
fi

echo ">>> 启动 Go 后端（开发模式）..."
echo "    配置: configs/config.yaml"
echo "    前台运行，Ctrl+C 退出"
echo ""
exec go run ./cmd/server "$@"
