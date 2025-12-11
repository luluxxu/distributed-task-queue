#!/bin/bash
set -xe
yum update -y

yum install -y docker

systemctl enable docker

systemctl start docker
docker pull ${worker_image}

docker stop dtq-worker 2>/dev/null || true
docker rm dtq-worker 2>/dev/null || true

docker run -d --restart always \
  --name dtq-worker \
  -e REDIS_ADDR="${redis_private_ip}:6379" \
  ${worker_image} \
  ./worker -queue=${queue_type} -mode=${mode}