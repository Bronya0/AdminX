#!/usr/bin/env bash
# 构建生产二进制到 bin/
# 用法: bash scripts/build.sh [--ldflags "version info"]
# 产物: bin/server (主服务) + bin/init-data (初始化工具)
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-dev}"
LDFLAGS="${LDFLAGS:--X main.version=$VERSION}"

echo ">>> 构建版本: $VERSION"
echo ">>> 编译主服务..."
CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o bin/server ./cmd/server

echo ">>> 编译初始化工具..."
CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o bin/init-data ./cmd/init-data

echo ""
echo "构建完成:"
ls -lh bin/server bin/init-data
echo ""
echo "运行:"
echo "  ./bin/init-data --username admin --password admin123  # 初始化数据"
echo "  ./bin/server                                              # 启动服务"
echo "  ./bin/server --scheduler                                  # 启动 + 内置调度器"
