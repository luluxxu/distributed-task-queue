#!/bin/bash

set -xe

# 安装 Docker
yum update -y
yum install -y docker
systemctl enable docker
systemctl start docker

# 拉取并启动 API 容器（Docker Hub 公共镜像）
docker pull luluxxu/dtq-api:latest

# 停止并删除旧容器（如果存在）
docker stop dtq-api 2>/dev/null || true
docker rm dtq-api 2>/dev/null || true

docker run -d --restart always \
  --name dtq-api \
  -p 8080:8080 \
  -e REDIS_ADDR="${redis_private_ip}:6379" \
  luluxxu/dtq-api:latest

