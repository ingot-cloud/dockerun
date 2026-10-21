#!/usr/bin/env bash
set -euo pipefail
platforms="${PLATFORMS:-linux/amd64,linux/arm64}"
image="${IMAGE:-dockerun:latest}"
cache_args=()
if [[ "${NO_CACHE:-0}" == 1 ]]; then cache_args+=(--no-cache); fi
if [[ "${PUSH:-0}" == 1 ]]; then
  docker buildx build --platform "$platforms" --tag "$image" "${cache_args[@]}" --push .
else
  IFS=',' read -r -a targets <<< "$platforms"
  for platform in "${targets[@]}"; do
    case "$platform" in linux/amd64|linux/arm64) ;; *) echo "Unsupported platform: $platform" >&2; exit 1;; esac
    arch="${platform#linux/}"
    tag="${image}-${arch}"
    docker buildx build --platform "$platform" --tag "$tag" "${cache_args[@]}" --load .
    docker save --output "${OUTPUT_PREFIX:-dockerun}-${arch}.tar" "$tag"
  done
fi
