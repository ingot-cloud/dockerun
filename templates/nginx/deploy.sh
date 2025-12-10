#!/usr/bin/env bash
set -e

echo "Deploying Nginx..."

# Create necessary directories
mkdir -p ${WORK_DIR}/html ${WORK_DIR}/conf

# Stop and remove existing container if exists
docker stop nginx-web 2>/dev/null || true
docker rm nginx-web 2>/dev/null || true

# Run Nginx container
CONTAINER_ID=$(docker run -d \
  --name nginx-web \
  -p ${NGINX_PORT:-80}:80 \
  -v ${WORK_DIR}/html:/usr/share/nginx/html:ro \
  -v ${WORK_DIR}/conf/nginx.conf:/etc/nginx/nginx.conf:ro \
  --restart unless-stopped \
  nginx:${NGINX_VERSION:-latest})

echo "Nginx deployed successfully!"
echo "Container ID: $CONTAINER_ID"

