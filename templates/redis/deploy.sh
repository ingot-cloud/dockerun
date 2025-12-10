#!/usr/bin/env bash
set -e

echo "Deploying Redis..."

# Create data directory
mkdir -p ${WORK_DIR}/data

# Stop and remove existing container if exists
docker stop redis-cache 2>/dev/null || true
docker rm redis-cache 2>/dev/null || true

# Run Redis container
CONTAINER_ID=$(docker run -d \
  --name redis-cache \
  -p ${REDIS_PORT:-6379}:6379 \
  -v ${WORK_DIR}/data:/data \
  --restart unless-stopped \
  redis:${REDIS_VERSION:-latest} \
  redis-server --appendonly yes --requirepass "${REDIS_PASSWORD}")

echo "Redis deployed successfully!"
echo "Container ID: $CONTAINER_ID"
echo "Password: ${REDIS_PASSWORD}"

