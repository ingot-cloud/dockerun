#!/usr/bin/env bash
set -euo pipefail
# Build online first, or load the Dockerun image before running this script offline.
docker compose up -d --no-build --pull never
