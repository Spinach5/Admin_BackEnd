#!/bin/bash
# ============================================================
# 脚本名称: local-run.sh
# 功能: 本地编译并运行 Go 项目，加载指定环境变量文件
# 用法: ./local-run.sh [dev|prod]
#       dev  - 使用 .env.dev (默认)
#       prod - 使用 .env.prod
# ============================================================

set -e

# ---------- 配置 ----------
ENV_TYPE="${1:-dev}"   # 默认 dev，可选 prod
PROJECT_DIR="$(pwd)"
BIN_NAME="backend-local"
DIST_DIR="./dist"
BUILD_CMD="go build -o $DIST_DIR/$BIN_NAME ./cmd/server"

# 选择环境文件
if [ "$ENV_TYPE" = "prod" ]; then
    ENV_FILE=".env.prod"
    echo "🏭 使用生产环境配置: $ENV_FILE"
elif [ "$ENV_TYPE" = "dev" ]; then
    ENV_FILE=".env.dev"
    echo "🛠️ 使用开发环境配置: $ENV_FILE"
else
    echo "❌ 无效参数: $ENV_TYPE，请使用 dev 或 prod"
    exit 1
fi

# 检查环境文件是否存在
if [ ! -f "$ENV_FILE" ]; then
    echo "❌ 环境文件 $ENV_FILE 不存在，请先创建"
    exit 1
fi

# ---------- 编译 ----------
echo "🔨 编译 Go 项目..."
mkdir -p "$DIST_DIR"
eval "$BUILD_CMD"
echo "✅ 编译完成: $DIST_DIR/$BIN_NAME"

# ---------- 停止旧进程（可选） ----------
OLD_PID=$(pgrep -f "$DIST_DIR/$BIN_NAME" || true)
if [ -n "$OLD_PID" ]; then
    echo "🛑 发现旧进程 PID: $OLD_PID，正在停止..."
    kill "$OLD_PID"
    sleep 2
fi

# ---------- 加载环境变量并运行 ----------
echo "🚀 启动服务 (Ctrl+C 停止)..."
echo "📋 日志将输出到终端，也可以重定向到文件"

# 从 .env 文件导出环境变量，然后前台运行
# 使用 set -a 确保后续命令自动导出所有变量
set -a
source "$ENV_FILE"
set +a

# 前台运行，按 Ctrl+C 即可终止
exec "$DIST_DIR/$BIN_NAME"