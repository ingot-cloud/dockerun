#!/usr/bin/env bash
set -e

echo "Deploying MySQL..."

# Create data directory
mkdir -p ${WORK_DIR}/data

# Stop and remove existing container if exists
docker stop mysql-db 2>/dev/null || true
docker rm mysql-db 2>/dev/null || true

# Run MySQL container
CONTAINER_ID=$(docker run -d \
  --name mysql-db \
  -p ${MYSQL_PORT:-3306}:3306 \
  -e MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD} \
  -e MYSQL_DATABASE=${MYSQL_DATABASE:-mydb} \
  -e MYSQL_USER=${MYSQL_USER:-user} \
  -e MYSQL_PASSWORD=${MYSQL_PASSWORD} \
  -v ${WORK_DIR}/data:/var/lib/mysql \
  --restart unless-stopped \
  mysql:${MYSQL_VERSION:-8.0})

echo "MySQL deployed successfully!"
echo "Container ID: $CONTAINER_ID"
echo "Root Password: ${MYSQL_ROOT_PASSWORD}"
echo "Database: ${MYSQL_DATABASE:-mydb}"

