#!/usr/bin/env bash
# Dockerun 镜像构建、离线打包与仓库推送；配置见 build.env.example。
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
config_file="${BUILD_ENV_FILE:-$root_dir/build.env}"
pending_archive=''

info() { printf '[信息] %s\n' "$*"; }
fail() { printf '[错误] %s\n' "$*" >&2; exit 1; }

show_help() {
  cat <<'HELP'
Dockerun 镜像构建与交付

用法：./build.sh [--dry-run|--help]
可选读取脚本旁的 build.env；BUILD_ENV_FILE 可指定其他文件。
已导出的同名环境变量优先于配置文件。

镜像：REGISTRY、IMAGE_NAME、IMAGE_TAG；IMAGE 可直接覆盖完整镜像地址。
输出：EXPORT=1 导出离线包；PUSH=1 推送仓库，可同时开启。
其他：PLATFORMS、EXTRA_TAGS、OUTPUT_PREFIX、BUILDER、DOCKERFILE、
      BUILD_CONTEXT、NO_CACHE、PULL、PROGRESS、DRY_RUN。

示例：
  ./build.sh --dry-run
  PLATFORMS=linux/amd64 ./build.sh
  REGISTRY=registry.example.com IMAGE_NAME=team/dockerun PUSH=1 EXPORT=1 ./build.sh

推送前先使用 docker login 登录目标仓库；脚本不保存账号或密码。
HELP
}

is_config_key() {
  case "$1" in
    REGISTRY|IMAGE_NAME|IMAGE_TAG|IMAGE|PLATFORMS|EXTRA_TAGS|EXPORT|PUSH|OUTPUT_PREFIX|BUILDER|DOCKERFILE|BUILD_CONTEXT|NO_CACHE|PULL|PROGRESS|DRY_RUN) return 0 ;;
    *) return 1 ;;
  esac
}

# 与 deploy.env 一样，仅读取单行字面量，不执行 source 或变量插值。
load_config() {
  if [[ ! -f "$config_file" ]]; then
    [[ -z "${BUILD_ENV_FILE:-}" ]] || fail "构建配置文件不存在：$config_file"
    return
  fi
  local line key value first seen=' ' number=0
  while IFS= read -r line || [[ -n "$line" ]]; do
    number=$((number + 1))
    line="${line%$'\r'}"
    [[ "$line" =~ ^[[:space:]]*(#|$) ]] && continue
    [[ "$line" =~ ^[[:space:]]*([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*=(.*)$ ]] || fail "构建配置第 $number 行格式错误，请使用 KEY=value。"
    key="${BASH_REMATCH[1]}"
    value="${BASH_REMATCH[2]}"
    is_config_key "$key" || fail "构建配置包含未知变量：$key"
    [[ "$seen" != *" $key "* ]] || fail "构建配置变量重复：$key"
    seen="$seen$key "
    value="${value#"${value%%[![:space:]]*}"}"
    value="${value%"${value##*[![:space:]]}"}"
    first="${value:0:1}"
    if [[ "$first" == "'" || "$first" == '"' ]]; then
      [[ ${#value} -ge 2 && "${value: -1}" == "$first" ]] || fail "构建配置第 $number 行的引号未闭合。"
      value="${value:1:${#value}-2}"
    fi
    if ! declare -p "$key" >/dev/null 2>&1; then
      printf -v "$key" '%s' "$value"
      export "$key"
    fi
  done < "$config_file"
}

validate_tag() {
  [[ "$1" =~ ^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127}$ ]] || fail "镜像标签格式错误：$1"
}

configure() {
  local version key platform tag last_component path_part host
  version="$(cat "$root_dir/VERSION")"
  REGISTRY="${REGISTRY:-}"
  IMAGE_NAME="${IMAGE_NAME:-dockerun}"
  IMAGE_TAG="${IMAGE_TAG:-$version}"
  PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
  EXTRA_TAGS="${EXTRA_TAGS:-}"
  PUSH="${PUSH:-0}"
  # 保留旧的 PUSH=1 仅推送行为；显式 EXPORT=1 可同时打包和推送。
  if [[ -z "${EXPORT:-}" ]]; then
    EXPORT=1
    [[ "$PUSH" != 1 ]] || EXPORT=0
  fi
  NO_CACHE="${NO_CACHE:-0}"
  PULL="${PULL:-0}"
  DRY_RUN="${DRY_RUN:-0}"
  PROGRESS="${PROGRESS:-auto}"
  BUILDER="${BUILDER:-}"
  for key in PUSH EXPORT NO_CACHE PULL DRY_RUN; do
    [[ "${!key}" == 0 || "${!key}" == 1 ]] || fail "$key 仅支持 0 或 1。"
  done
  [[ "$EXPORT" == 1 || "$PUSH" == 1 ]] || fail 'EXPORT 和 PUSH 至少开启一个。'
  case "$PROGRESS" in auto|plain|tty|quiet|rawjson|none) ;; *) fail 'PROGRESS 格式错误。' ;; esac

  if [[ -n "${IMAGE:-}" ]]; then
    repository="$IMAGE"
    last_component="${IMAGE##*/}"
    if [[ "$last_component" == *:* ]]; then
      IMAGE_TAG="${IMAGE##*:}"
      repository="${IMAGE%:*}"
    fi
  else
    [[ -z "$REGISTRY" || "$REGISTRY" =~ ^[a-zA-Z0-9][a-zA-Z0-9.-]*(:[0-9]+)?$ ]] || fail 'REGISTRY 只能包含仓库主机名及可选端口，不带协议或路径。'
    [[ -z "$REGISTRY" || "$REGISTRY" == *.* || "$REGISTRY" == *:* || "$REGISTRY" == localhost ]] || fail 'REGISTRY 请使用完整域名、IP、localhost 或主机名:端口，避免被 Docker 识别为 Docker Hub 命名空间。'
    repository="${REGISTRY:+$REGISTRY/}$IMAGE_NAME"
  fi
  path_part="$repository"
  if [[ "$repository" == */* ]]; then
    host="${repository%%/*}"
    if [[ "$host" == *.* || "$host" == *:* || "$host" == localhost ]]; then
      [[ "$host" =~ ^[a-zA-Z0-9][a-zA-Z0-9.-]*(:[0-9]+)?$ ]] || fail '镜像仓库主机名格式错误。'
      path_part="${repository#*/}"
    fi
  fi
  local path_pattern='^[a-z0-9]+(([._]|__|-+)[a-z0-9]+)*(/[a-z0-9]+(([._]|__|-+)[a-z0-9]+)*)*$'
  [[ "$path_part" =~ $path_pattern && ${#repository} -le 255 ]] || fail '镜像名称必须是有效的小写仓库路径，不含协议、标签或摘要。'
  validate_tag "$IMAGE_TAG"
  image="$repository:$IMAGE_TAG"
  image_refs=("$image")
  if [[ -n "$EXTRA_TAGS" ]]; then
    [[ "$EXTRA_TAGS" != ,* && "$EXTRA_TAGS" != *, && "$EXTRA_TAGS" != *,,* ]] || fail 'EXTRA_TAGS 请使用逗号分隔的非空标签。'
    local extra_tags
    local seen_tags=" $IMAGE_TAG "
    IFS=',' read -r -a extra_tags <<< "$EXTRA_TAGS"
    for tag in "${extra_tags[@]}"; do
      validate_tag "$tag"
      [[ "$seen_tags" != *" $tag "* ]] || fail "镜像标签重复：$tag"
      seen_tags="$seen_tags$tag "
      image_refs+=("$repository:$tag")
    done
  fi

  [[ "$PLATFORMS" != ,* && "$PLATFORMS" != *, && "$PLATFORMS" != *,,* ]] || fail 'PLATFORMS 请使用逗号分隔的非空平台。'
  IFS=',' read -r -a targets <<< "$PLATFORMS"
  local seen_platforms=' '
  for platform in "${targets[@]}"; do
    case "$platform" in linux/amd64|linux/arm64) ;; *) fail "不支持的平台：$platform" ;; esac
    [[ "$seen_platforms" != *" $platform "* ]] || fail "目标平台重复：$platform"
    seen_platforms="$seen_platforms$platform "
    if [[ "$EXPORT" == 1 && "$PUSH" == 1 && ${#targets[@]} -gt 1 ]]; then
      tag="$IMAGE_TAG-${platform#linux/}"
      validate_tag "$tag"
      [[ ",$EXTRA_TAGS," != *",$tag,"* ]] || fail "附加标签与架构标签冲突：$tag"
    fi
  done

  build_context="${BUILD_CONTEXT:-$root_dir}"
  [[ "$build_context" == /* ]] || build_context="$root_dir/$build_context"
  [[ -d "$build_context" ]] || fail "构建目录不存在：$build_context"
  dockerfile="${DOCKERFILE:-Dockerfile}"
  [[ "$dockerfile" == /* ]] || dockerfile="$build_context/$dockerfile"
  [[ -f "$dockerfile" ]] || fail "Dockerfile 不存在：$dockerfile"
  output_prefix="${OUTPUT_PREFIX:-docker-packages/${path_part##*/}-$IMAGE_TAG}"
  [[ "$output_prefix" == /* ]] || output_prefix="$root_dir/$output_prefix"

  build_args=(--file "$dockerfile" --progress "$PROGRESS")
  [[ -z "$BUILDER" ]] || build_args+=(--builder "$BUILDER")
  [[ "$NO_CACHE" != 1 ]] || build_args+=(--no-cache)
  [[ "$PULL" != 1 ]] || build_args+=(--pull)
  for tag in "${image_refs[@]}"; do
    build_args+=(--tag "$tag")
  done
}

run() {
  if [[ "$DRY_RUN" == 1 ]]; then
    printf '[预览] '
    printf '%q ' "$@"
    printf '\n'
  else
    "$@"
  fi
}

cleanup() {
  [[ -z "$pending_archive" ]] || rm -f "$pending_archive"
}

save_archive() {
  local archive="$1"
  if [[ "$DRY_RUN" == 1 ]]; then
    run docker save --output "$archive" "${image_refs[@]}"
    return
  fi
  pending_archive="$(mktemp "${archive}.partial.XXXXXX")"
  docker save --output "$pending_archive" "${image_refs[@]}" || fail "离线包导出失败，原文件保留：$archive"
  mv -f "$pending_archive" "$archive"
  pending_archive=''
  info "离线包已生成：$archive"
}

main() {
  [[ $# -le 1 ]] || fail '只接受 --dry-run 或 --help；构建参数通过环境变量配置。'
  local force_preview=0
  case "${1:-}" in
    --help|-h|help) show_help; return ;;
    --dry-run) force_preview=1 ;;
    '') ;;
    *) fail "未知参数：$1；使用 --help 查看帮助。" ;;
  esac
  load_config
  [[ "$force_preview" != 1 ]] || DRY_RUN=1
  configure
  if [[ "$DRY_RUN" != 1 ]]; then
    command -v docker >/dev/null 2>&1 || fail '未安装 Docker。'
    docker info >/dev/null 2>&1 || fail '无法连接 Docker。'
    docker buildx version >/dev/null 2>&1 || fail '未安装 Docker Buildx。'
  fi
  trap cleanup EXIT
  info "镜像：${image}；平台：${PLATFORMS}；离线打包：${EXPORT}；推送：${PUSH}"
  [[ "$PUSH" != 1 ]] || info '推送使用当前 Docker 登录凭据，请先登录目标仓库。'

  local platform arch ref archive
  if [[ "$EXPORT" == 1 ]]; then
    run mkdir -p "$(dirname "$output_prefix")"
    local platform_refs=()
    for platform in "${targets[@]}"; do
      arch="${platform#linux/}"
      run docker buildx build --platform "$platform" "${build_args[@]}" --load "$build_context"
      archive="$output_prefix-$arch.tar"
      save_archive "$archive"
      if [[ "$PUSH" == 1 ]]; then
        if [[ ${#targets[@]} -eq 1 ]]; then
          for ref in "${image_refs[@]}"; do
            run docker push "$ref"
          done
        else
          ref="$repository:$IMAGE_TAG-$arch"
          run docker tag "$image" "$ref"
          run docker push "$ref"
          platform_refs+=("$ref")
        fi
      fi
    done
    if [[ "$PUSH" == 1 && ${#targets[@]} -gt 1 ]]; then
      local manifest_args=(buildx imagetools create)
      [[ -z "$BUILDER" ]] || manifest_args+=(--builder "$BUILDER")
      for ref in "${image_refs[@]}"; do
        manifest_args+=(--tag "$ref")
      done
      run docker "${manifest_args[@]}" "${platform_refs[@]}"
    fi
  else
    run docker buildx build --platform "$PLATFORMS" "${build_args[@]}" --push "$build_context"
  fi
  if [[ "$DRY_RUN" == 1 ]]; then
    info '以上仅为命令预览，未构建、导出或推送镜像。'
  else
    info '构建与交付已完成。'
  fi
}

main "$@"
