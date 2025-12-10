#!/bin/bash
set -xe
yum update -y

yum install -y docker

systemctl enable docker

systemctl start docker
docker pull luluxxu/dtq-worker:latest

# 停止并删除旧容器（如果存在）
docker stop dtq-worker 2>/dev/null || true
docker rm dtq-worker 2>/dev/null || true

docker run -d --restart always \
  --name dtq-worker \
  -e REDIS_ADDR="${redis_private_ip}:6379" \
  luluxxu/dtq-worker:latest \
  ./worker -queue=${queue_type} -mode=${mode}