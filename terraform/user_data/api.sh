#!/bin/bash
# User data script for API instance

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

# Pull API image
docker pull ${api_image}

# Run API container
docker run -d \
  --name api \
  --restart unless-stopped \
  -p 8080:8080 \
  -e REDIS_ADDR="${redis_addr}:6379" \
  ${api_image}

# Wait for API to start
sleep 5

# Health check
for i in {1..10}; do
  if curl -f http://localhost:8080/queue/status > /dev/null 2>&1; then
    echo "API is ready"
    break
  fi
  echo "Waiting for API... ($i/10)"
  sleep 5
done

# Log completion
echo "API container started successfully" >> /var/log/user-data.log

