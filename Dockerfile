# 使用轻量级基础镜像（AMD64 架构以获得最大兼容性）
FROM --platform=linux/amd64 alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata sqlite-libs bash curl

# 设置时区
ENV TZ=Asia/Shanghai

# 设置工作目录
WORKDIR /app

# 复制已编译的后端二进制文件
COPY backend/main .

# 复制前端构建产物
COPY frontend/dist ./static

# 复制模板目录
COPY templates ./templates

# 创建数据目录
RUN mkdir -p /app/data

# 设置可执行权限
RUN chmod +x /app/main

# 暴露端口
EXPOSE 8080

# 设置环境变量
ENV GIN_MODE=release
ENV SERVER_ADDR=:8080
ENV DATABASE_PATH=/app/data/dockerun.db
ENV TEMPLATE_DIR=/app/templates

# Docker Network 配置
ENV DOCKER_NETWORK_NAME=ingot-net
ENV DOCKER_NETWORK_DRIVER=bridge
ENV DOCKER_NETWORK_SUBNET=172.88.0.0/16
ENV DOCKER_NETWORK_GATEWAY=172.88.0.1

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# 运行应用
CMD ["./main"]
