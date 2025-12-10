#!/usr/bin/env bash
set -e

echo "Deploying Docker Compose application..."

cd ${WORK_DIR}

# Stop existing services
docker-compose -f docker-compose.yml down 2>/dev/null || true

# Start services
docker-compose -f docker-compose.yml up -d

# Get container IDs
CONTAINER_ID=$(docker-compose -f docker-compose.yml ps -q | head -n 1)

echo "Docker Compose application deployed successfully!"
echo "Container ID: $CONTAINER_ID"
docker-compose -f docker-compose.yml ps

