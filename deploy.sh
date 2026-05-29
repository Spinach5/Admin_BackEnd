#!/bin/bash
set -e

SSH_ALIAS="server"               
REMOTE_DIR="/home/www/project/go" # 服务器上项目目录
SERVICE_NAME="go-app"             # systemd 服务名
ENV_FILE=".env.prod"              # 本地环境变量文件

BINARY_LOCAL="./dist/backend"     # 本地编译产物路径
BUILD_CMD="CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $BINARY_LOCAL ./cmd/server"

echo "编译 Go 项目..."
eval $BUILD_CMD

rsync -avz --delete "$BINARY_LOCAL" --rsync-path='sudo rsync' "$SSH_ALIAS:$REMOTE_DIR/backend"
rsync -avz --delete "$ENV_FILE" --rsync-path='sudo rsync' "$SSH_ALIAS:$REMOTE_DIR/.env"

echo "重启服务 $SERVICE_NAME ..."
ssh "$SSH_ALIAS" "sudo systemctl restart $SERVICE_NAME"

echo "部署完成"