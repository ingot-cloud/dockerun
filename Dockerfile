# 多阶段构建
# 阶段1: 构建前端
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

# 安装pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# 复制依赖文件
COPY frontend/package.json frontend/pnpm-lock.yaml* ./

# 安装依赖
RUN pnpm install --frozen-lockfile

# 复制源代码
COPY frontend/ ./

# 构建
RUN pnpm run build

# 阶段2: 构建后端
FROM golang:1.21-alpine AS backend-builder

WORKDIR /backend

# 安装必要的依赖
RUN apk add --no-cache git

# 复制go.mod和go.sum
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# 复制后端源代码
COPY backend/ ./

# 构建后端
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .

# 阶段3: 最终镜像
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata sqlite-libs bash curl

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从构建阶段复制文件
COPY --from=backend-builder /backend/main .
COPY --from=frontend-builder /frontend/dist ./static

# 复制模板目录
COPY templates ./templates

# 创建数据目录
RUN mkdir -p /app/data

# 暴露端口
EXPOSE 8080

# 设置环境变量
ENV GIN_MODE=release
ENV SERVER_ADDR=:8080
ENV DATABASE_PATH=/app/data/dockerun.db
ENV TEMPLATE_DIR=/app/templates

# Docker Network 配置
ENV DOCKER_NETWORK_NAME=dockerun-net
ENV DOCKER_NETWORK_DRIVER=bridge
ENV DOCKER_NETWORK_SUBNET=172.88.0.0/16
ENV DOCKER_NETWORK_GATEWAY=172.88.0.1

# 运行应用
CMD ["./main"]

