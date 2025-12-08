#!/bin/bash
# User data script for Worker instance

set -e

# Update system
yum update -y

# Install Docker and netcat (for health checks)
yum install -y docker nc
systemctl start docker
systemctl enable docker
usermod -a -G docker ec2-user

# Wait for Redis to be ready (retry logic)
RETRY_COUNT=0
MAX_RETRIES=30
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
  if nc -z ${redis_addr} 6379; then
    echo "Redis is ready"
    break
  fi
  echo "Waiting for Redis... ($RETRY_COUNT/$MAX_RETRIES)"
  sleep 10
  RETRY_COUNT=$((RETRY_COUNT + 1))
done

# Pull worker image
docker pull ${worker_image}

# Run worker container
docker run -d \
  --name worker \
  --restart unless-stopped \
  -e REDIS_ADDR="${redis_addr}:6379" \
  ${worker_image} \
  ./worker -queue=${queue_type} -mode=${mode}

# Log completion
echo "Worker container started successfully" >> /var/log/user-data.log

