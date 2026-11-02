#!/usr/bin/env bash
set -euo pipefail
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
version="$(cat "$root_dir/VERSION")"
platforms="${PLATFORMS:-linux/amd64,linux/arm64}"
image="${IMAGE:-dockerun:$version}"
output_prefix="${OUTPUT_PREFIX:-docker-packages/dockerun-$version}"
build_args=(--tag "$image")
if [[ "${NO_CACHE:-0}" == 1 ]]; then build_args+=(--no-cache); fi
if [[ "${PUSH:-0}" == 1 ]]; then
  docker buildx build --platform "$platforms" "${build_args[@]}" --push "$root_dir"
else
  IFS=',' read -r -a targets <<< "$platforms"
  mkdir -p "$(dirname "$output_prefix")"
  for platform in "${targets[@]}"; do
    case "$platform" in linux/amd64|linux/arm64) ;; *) echo "不支持的平台：$platform" >&2; exit 1;; esac
    arch="${platform#linux/}"
    docker buildx build --platform "$platform" "${build_args[@]}" --load "$root_dir"
    docker save --output "${output_prefix}-${arch}.tar" "$image"
  done
fi
