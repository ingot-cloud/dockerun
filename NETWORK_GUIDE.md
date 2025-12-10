# 🌐 Docker 网络自动创建指南

## 概述

Dockerun 在启动时会自动创建一个 Docker 网络，所有通过 Dockerun 部署的容器都可以连接到这个网络，实现容器间通信。

## 功能特性

- ✅ 启动时自动检查网络是否存在
- ✅ 如果不存在则自动创建
- ✅ 如果已存在则跳过创建
- ✅ 支持自定义网络参数
- ✅ 部署脚本自动注入网络名称

## 配置参数

### 环境变量

| 变量名 | 说明 | 默认值 | 示例 |
|--------|------|--------|------|
| `DOCKER_NETWORK_NAME` | 网络名称 | `dockerun-net` | `my-app-net` |
| `DOCKER_NETWORK_DRIVER` | 网络驱动 | `bridge` | `bridge`, `overlay` |
| `DOCKER_NETWORK_SUBNET` | 子网 CIDR | `172.20.0.0/16` | `10.10.0.0/16` |
| `DOCKER_NETWORK_GATEWAY` | 网关地址 | `172.20.0.1` | `10.10.0.1` |

### 默认配置

```bash
DOCKER_NETWORK_NAME=dockerun-net
DOCKER_NETWORK_DRIVER=bridge
DOCKER_NETWORK_SUBNET=172.20.0.0/16
DOCKER_NETWORK_GATEWAY=172.20.0.1
```

## 启动流程

```
应用启动
    ↓
初始化 Docker 客户端
    ↓
检查网络是否存在？
    ├─ 是 → 跳过创建
    └─ 否 → 创建网络
         ↓
    [成功] Network 'dockerun-net' created successfully
```

### 日志输出

**网络已存在：**
```
Loading templates from: ../templates
Network 'dockerun-net' already exists, skipping creation
Server starting on :8080
```

**创建新网络：**
```
Loading templates from: ../templates
Creating network 'dockerun-net'...
Network 'dockerun-net' created successfully (ID: abc123...)
Server starting on :8080
```

## 在部署脚本中使用

### 自动注入环境变量

执行部署脚本时，系统会自动注入以下环境变量：

| 变量名 | 说明 | 示例值 |
|--------|------|--------|
| `WORK_DIR` | 工作目录 | `/opt/my-app` |
| `DOCKER_NETWORK` | Docker 网络名称 | `dockerun-net` |

### 示例脚本

```bash
#!/usr/bin/env bash
set -e

echo "Deploying application..."

# 使用自动注入的网络名称
docker run -d \
  --name my-app \
  --network ${DOCKER_NETWORK:-dockerun-net} \
  -p 8080:80 \
  nginx:latest

echo "Container ID: $(docker ps -qf name=my-app)"
```

### 容器间通信

部署到同一网络的容器可以通过容器名称互相访问：

```bash
# 部署 MySQL
docker run -d \
  --name mysql-db \
  --network dockerun-net \
  -e MYSQL_ROOT_PASSWORD=secret \
  mysql:8.0

# 部署应用，连接到 MySQL
docker run -d \
  --name web-app \
  --network dockerun-net \
  -e DB_HOST=mysql-db \
  -e DB_PORT=3306 \
  my-web-app:latest
```

在应用中：
```javascript
// 直接使用容器名称作为主机名
const connection = mysql.createConnection({
  host: 'mysql-db',  // ✅ 容器名称
  port: 3306,
  user: 'root',
  password: 'secret'
})
```

## 自定义配置

### 方式1: 修改环境变量

**本地开发：**
```bash
# backend/Makefile
dev:
	TEMPLATE_DIR=../templates \
	DOCKER_NETWORK_NAME=my-custom-net \
	DOCKER_NETWORK_SUBNET=10.10.0.0/16 \
	DOCKER_NETWORK_GATEWAY=10.10.0.1 \
	go run main.go
```

**Docker Compose：**
```yaml
# docker-compose.yml
services:
  dockerun:
    environment:
      - DOCKER_NETWORK_NAME=my-custom-net
      - DOCKER_NETWORK_SUBNET=10.10.0.0/16
      - DOCKER_NETWORK_GATEWAY=10.10.0.1
```

### 方式2: 使用现有网络

如果您已经有一个 Docker 网络，可以指定使用它：

```bash
# 使用现有网络
DOCKER_NETWORK_NAME=existing-network go run main.go
```

应用会检查该网络是否存在：
- 如果存在：跳过创建
- 如果不存在：创建新网络

## 网络信息查看

### 查看创建的网络

```bash
# 列出所有网络
docker network ls

# 查看网络详情
docker network inspect dockerun-net

# 查看连接到网络的容器
docker network inspect dockerun-net -f '{{range .Containers}}{{.Name}} {{end}}'
```

### 输出示例

```json
{
  "Name": "dockerun-net",
  "Driver": "bridge",
  "IPAM": {
    "Config": [
      {
        "Subnet": "172.20.0.0/16",
        "Gateway": "172.20.0.1"
      }
    ]
  },
  "Containers": {
    "abc123...": {
      "Name": "mysql-db",
      "IPv4Address": "172.20.0.2/16"
    },
    "def456...": {
      "Name": "nginx-web",
      "IPv4Address": "172.20.0.3/16"
    }
  }
}
```

## 高级用法

### 使用 Overlay 网络（Swarm）

如果您使用 Docker Swarm，可以配置为 overlay 网络：

```bash
DOCKER_NETWORK_NAME=dockerun-overlay
DOCKER_NETWORK_DRIVER=overlay
DOCKER_NETWORK_SUBNET=10.11.0.0/16
DOCKER_NETWORK_GATEWAY=10.11.0.1
```

### 自定义子网

```bash
# 使用不同的 IP 段
DOCKER_NETWORK_SUBNET=192.168.100.0/24
DOCKER_NETWORK_GATEWAY=192.168.100.1
```

### 连接外部网络

如果需要连接到外部网络：

```bash
# 在部署脚本中连接多个网络
docker run -d \
  --name my-app \
  --network dockerun-net \
  --network external-net \
  my-app:latest
```

## 最佳实践

### 1. 网络命名
```
✅ 推荐: app-name-net, project-net
❌ 避免: net, network, default
```

### 2. 子网规划
```
开发环境: 172.20.0.0/16  (65534 个地址)
生产环境: 10.10.0.0/16   (65534 个地址)
测试环境: 192.168.100.0/24 (254 个地址)
```

### 3. 容器命名
使用有意义的名称，方便容器间引用：
```bash
--name mysql-db      # ✅ 清晰明确
--name db            # ❌ 太简单
--name container1    # ❌ 无意义
```

### 4. 安全考虑
- 使用独立网络隔离应用
- 不要将所有容器放在同一网络
- 敏感服务不要暴露端口

## 故障排查

### 网络创建失败

**问题**: 看到 `failed to create network` 错误

**可能原因**:
1. 子网冲突（与其他网络重叠）
2. 权限不足
3. Docker daemon 未运行

**解决方案**:
```bash
# 检查现有网络
docker network ls

# 检查子网使用情况
docker network inspect bridge

# 删除冲突的网络
docker network rm conflicting-network

# 使用不同的子网
DOCKER_NETWORK_SUBNET=172.21.0.0/16
```

### 容器无法加入网络

**问题**: 容器启动时提示网络不存在

**解决方案**:
```bash
# 确认网络已创建
docker network ls | grep dockerun-net

# 手动创建网络
docker network create \
  --driver bridge \
  --subnet 172.20.0.0/16 \
  --gateway 172.20.0.1 \
  dockerun-net
```

### 容器间无法通信

**检查清单**:
1. ✅ 容器在同一网络中
2. ✅ 使用正确的容器名称
3. ✅ 端口配置正确
4. ✅ 防火墙规则允许

```bash
# 测试网络连通性
docker exec container1 ping container2

# 检查 DNS 解析
docker exec container1 nslookup container2
```

## 示例场景

### 场景1: Web + 数据库

```bash
# 1. Dockerun 自动创建 dockerun-net

# 2. 部署 MySQL
docker run -d \
  --name mysql-db \
  --network dockerun-net \
  -e MYSQL_ROOT_PASSWORD=secret \
  mysql:8.0

# 3. 部署 Web 应用
docker run -d \
  --name web-app \
  --network dockerun-net \
  -p 8080:80 \
  -e DB_HOST=mysql-db \
  web-app:latest

# 4. Web 应用可以通过 mysql-db:3306 访问数据库
```

### 场景2: 微服务架构

```bash
# API Gateway
docker run -d --name api-gateway --network dockerun-net -p 80:80 gateway:latest

# 用户服务
docker run -d --name user-service --network dockerun-net user-svc:latest

# 订单服务
docker run -d --name order-service --network dockerun-net order-svc:latest

# Redis 缓存
docker run -d --name redis --network dockerun-net redis:latest

# 服务间可以直接通过名称访问
# http://user-service:8080
# http://order-service:8080
# redis:6379
```

## 管理命令

```bash
# 查看网络
docker network ls

# 查看网络详情
docker network inspect dockerun-net

# 连接容器到网络
docker network connect dockerun-net my-container

# 断开容器连接
docker network disconnect dockerun-net my-container

# 删除网络（谨慎！）
docker network rm dockerun-net
```

---

更多信息请参考：
- [Docker 网络文档](https://docs.docker.com/network/)
- [CONFIGURATION.md](CONFIGURATION.md)

