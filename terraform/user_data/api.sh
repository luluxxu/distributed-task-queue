#!/bin/bash

set -xe

yum update -y
yum install -y docker
systemctl enable docker
systemctl start docker


docker pull ${api_image}

docker stop dtq-api 2>/dev/null || true
docker rm dtq-api 2>/dev/null || true

docker run -d --restart always \
  --name dtq-api \
  -p 8080:8080 \
  -e REDIS_ADDR="${redis_private_ip}:6379" \
  ${api_image}

