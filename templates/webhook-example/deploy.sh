#!/bin/bash
set -e

# ==============================================================================
# Webhook 自动部署示例脚本
# ==============================================================================
# 
# 此脚本演示如何通过 Webhook 自动部署应用
# 
# Webhook 触发时会自动注入以下环境变量：
# - IMAGE_NAME: 镜像名称（如：mycompany/myapp）
# - IMAGE_TAG: 镜像标签（如：latest, v1.0.0）
# - FULL_IMAGE: 完整镜像名（如：mycompany/myapp:latest）
# - DOCKER_NETWORK: Docker 网络（如：dockerun-net）
# 
# ==============================================================================

echo "======================================"
echo "  Webhook 自动部署"
echo "======================================"
echo "镜像: ${FULL_IMAGE}"
echo "时间: $(date)"
echo ""

# 容器名称（可以从环境变量设置）
CONTAINER_NAME="${CONTAINER_NAME:-myapp}"
CONTAINER_PORT="${CONTAINER_PORT:-8080}"
HOST_PORT="${HOST_PORT:-8080}"

echo "[1/5] 停止旧容器..."
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    docker stop ${CONTAINER_NAME} || true
    docker rm ${CONTAINER_NAME} || true
    echo "✓ 旧容器已删除"
else
    echo "✓ 没有旧容器"
fi

echo ""
echo "[2/5] 清理旧镜像..."
# DockeRun 已在 Webhook 配置中处理，这里可选
if [ "${CLEANUP_OLD_IMAGE}" = "true" ]; then
    docker rmi ${FULL_IMAGE} || true
    echo "✓ 旧镜像已删除"
else
    echo "✓ 跳过镜像清理（Webhook 配置已处理）"
fi

echo ""
echo "[3/5] 拉取最新镜像..."
# DockeRun 已在 Webhook 任务中处理，这里可选
echo "✓ 镜像已由 Webhook 拉取"

echo ""
echo "[4/5] 启动新容器..."
docker run -d \
    --name ${CONTAINER_NAME} \
    --network ${DOCKER_NETWORK:-bridge} \
    -p ${HOST_PORT}:${CONTAINER_PORT} \
    --restart unless-stopped \
    -e TZ=Asia/Shanghai \
    ${FULL_IMAGE}

echo "✓ 容器已启动: ${CONTAINER_NAME}"

echo ""
echo "[5/5] 健康检查..."
sleep 3

if docker ps --filter "name=${CONTAINER_NAME}" --filter "status=running" | grep -q ${CONTAINER_NAME}; then
    echo "✓ 容器运行正常"
    
    # 可选：执行健康检查（如果应用提供健康检查端点）
    # if curl -f http://localhost:${HOST_PORT}/health > /dev/null 2>&1; then
    #     echo "✓ 健康检查通过"
    # else
    #     echo "⚠ 健康检查失败，但容器正在运行"
    # fi
else
    echo "✗ 容器启动失败"
    echo "查看日志："
    docker logs ${CONTAINER_NAME}
    exit 1
fi

echo ""
echo "======================================"
echo "  ✅ 部署成功！"
echo "======================================"
echo "容器名称: ${CONTAINER_NAME}"
echo "镜像版本: ${FULL_IMAGE}"
echo "访问地址: http://localhost:${HOST_PORT}"
echo "======================================"

