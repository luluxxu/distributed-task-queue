#!/bin/bash
# User data script for Redis instance

set -e

# Update system
yum update -y

# Install Docker
yum install -y docker
systemctl start docker
systemctl enable docker
usermod -a -G docker ec2-user

# Pull and run Redis container
docker pull ${redis_image}

# Run Redis container
docker run -d \
  --name redis \
  --restart unless-stopped \
  -p 6379:6379 \
  ${redis_image}

# Wait for Redis to be ready
sleep 5
docker exec redis redis-cli ping || echo "Redis starting..."

# Log completion
echo "Redis container started successfully" >> /var/log/user-data.log

