# Webhook 自动部署功能

> 适用范围：外部 Registry 回调及历史脚本集成。当前镜像自动发布按应用环境显式配置监听规则和触发授权，旧安装升级时不会直接沿用历史脚本自动执行，操作以[一体化操作指南](INTEGRATED_DEPLOYMENT.md#自动发布授权)为准。

DockeRun 的 Webhook 功能允许您在 Docker Registry 收到镜像推送事件时自动触发部署脚本，实现完整的 CI/CD 自动化流程。

---

## 📋 功能特性

### 核心特性
- ✅ **自动触发部署** - Registry 推送镜像时自动执行部署脚本
- ✅ **智能匹配** - 根据镜像名称和标签匹配触发规则
- ✅ **异步处理** - Webhook 立即响应，部署任务异步执行
- ✅ **自动清理** - 可选自动删除旧容器和旧镜像
- ✅ **签名验证** - 支持 HMAC-SHA256 签名验证
- ✅ **执行日志** - 完整记录每次 Webhook 触发和执行情况
- ✅ **支持通配符** - 镜像名称支持 `*` 通配符

### 支持的 Registry
- Docker Hub
- Harbor (私有仓库)
- GitLab Container Registry
- 其他兼容 Docker Registry v2 API 的仓库

---

## 🚀 快速开始

### 1. 创建部署脚本

首先在 **Deploy Scripts** 页面创建一个部署脚本，例如：

```bash
#!/bin/bash
set -e

# 停止旧容器
docker stop my-app || true
docker rm my-app || true

# 拉取最新镜像
docker pull registry.example.com/myapp:${IMAGE_TAG}

# 运行新容器
docker run -d \
  --name my-app \
  -p 8080:8080 \
  --restart unless-stopped \
  registry.example.com/myapp:${IMAGE_TAG}

echo "Deployment completed successfully!"
```

### 2. 创建 Webhook 配置

在 **Webhooks** 页面点击 "新建 Webhook"，配置以下信息：

| 字段 | 说明 | 示例 |
|------|------|------|
| **名称** | 配置名称 | `Auto Deploy MyApp` |
| **镜像名称** | 要监听的镜像 | `mycompany/myapp` |
| **镜像标签** | 要监听的标签（可选） | `latest` 或留空表示任意标签 |
| **部署脚本** | 关联的部署脚本 | 选择刚创建的脚本 |
| **Registry** | 镜像所在的 Registry | 选择对应的 Registry（如果是私有仓库） |
| **清理旧容器** | 是否自动停止删除旧容器 | ✅ 启用（推荐） |
| **清理旧镜像** | 是否自动删除旧镜像 | ✅ 启用（可选） |
| **Webhook 密钥** | 用于验证签名的密钥（可选） | `my-secret-key-123` |

### 3. 配置 Registry Webhook

点击配置旁边的 **URL** 按钮，复制 Webhook URL，然后在您的 Registry 中配置：

#### Docker Hub
1. 进入仓库页面 → **Webhooks** 标签
2. 点击 **Create Webhook**
3. 粘贴 URL，保存

#### Harbor
1. 进入项目 → **Webhooks** 标签
2. 点击 **NEW WEBHOOK**
3. 选择事件类型：`PUSH_ARTIFACT`
4. 粘贴 URL
5. 如果配置了密钥，填写到 **Auth Header** 字段

#### GitLab Container Registry
1. 进入项目 → **Settings** → **Webhooks**
2. 粘贴 URL
3. 勾选 **Push events**
4. 保存

### 4. 测试

推送一个新镜像到 Registry：

```bash
docker tag myapp:latest registry.example.com/myapp:latest
docker push registry.example.com/myapp:latest
```

几秒钟后，查看 DockeRun 的 **Webhook** 页面 → **日志** 按钮，确认部署是否成功。

---

## 🔧 配置详解

### 镜像名称匹配

支持以下匹配模式：

| 模式 | 匹配示例 |
|------|---------|
| `mycompany/myapp` | 完全匹配 `mycompany/myapp` |
| `mycompany/*` | 匹配 `mycompany/` 下的所有镜像 |
| `*/myapp` | 匹配所有命名为 `myapp` 的镜像 |
| `*myapp*` | 匹配包含 `myapp` 的所有镜像 |

### 镜像标签匹配

| 配置 | 行为 |
|------|------|
| **留空** | 匹配所有标签 |
| **`latest`** | 只匹配 `latest` 标签 |
| **`v1.*`** | 目前不支持通配符标签（计划中） |

### 自动清理策略

#### 清理旧容器
启用后，部署前会自动执行：
1. 查找使用该镜像的所有容器
2. 停止容器（超时 10 秒）
3. 强制删除容器

#### 清理旧镜像
启用后，部署前会删除旧镜像，释放磁盘空间。

**注意：** 如果其他容器正在使用该镜像，删除会失败，但不影响后续部署。

---

## 🔐 安全性

### Webhook 签名验证

为防止恶意请求，强烈建议配置 Webhook 密钥。

#### 配置流程
1. 在 Webhook 配置中设置一个强密钥（如 `my-secret-key-123`）
2. 在 Registry 的 Webhook 设置中配置相同的密钥
3. DockeRun 会验证每个请求的 HMAC-SHA256 签名

#### 支持的签名头
- `X-Hub-Signature` (Docker Hub 格式)
- `X-Registry-Signature` (Harbor 格式)

#### 签名格式
```
sha256=<hex_encoded_hmac>
```

### 最佳实践
- ✅ 使用强随机密钥（至少 32 字符）
- ✅ 定期轮换密钥
- ✅ 限制 Webhook 配置的创建/修改权限
- ✅ 启用 HTTPS（生产环境）

---

## 📊 执行流程

### Webhook 触发流程

```
┌─────────────┐
│   Registry  │
│  (推送镜像)  │
└──────┬──────┘
       │ 1. POST /webhooks/:id
       ▼
┌─────────────────────┐
│  DockeRun Webhook   │
│  (接收并快速响应)    │  ← 200 OK (立即返回)
└──────┬──────────────┘
       │ 2. 验证签名
       ▼
┌─────────────────────┐
│  检查配置是否启用    │
│  匹配镜像名称/标签   │
└──────┬──────────────┘
       │ 3. 创建日志记录
       ▼
┌─────────────────────┐
│  加入异步任务队列    │
└──────┬──────────────┘
       │ 4. 后台异步执行
       ▼
┌─────────────────────┐
│  1) 停止旧容器      │
│  2) 删除旧镜像      │
│  3) 拉取新镜像      │
│  4) 执行部署脚本    │
└──────┬──────────────┘
       │ 5. 更新日志状态
       ▼
   ✅ 部署完成
```

### 异步任务队列

- **工作线程数:** 3 个（可配置）
- **队列容量:** 100 个任务
- **超时处理:** 任务超时时，Registry 不会受影响
- **并发控制:** 多个 Webhook 同时触发时，按队列顺序处理

---

## 📈 监控和日志

### 查看执行日志

在 **Webhooks** 页面，点击配置旁边的 **日志** 按钮，可以查看：

| 字段 | 说明 |
|------|------|
| **状态** | `等待中` / `执行中` / `成功` / `失败` |
| **镜像** | 触发的镜像名称和标签 |
| **消息** | 执行详情或错误信息 |
| **耗时** | 执行时长（秒） |
| **触发时间** | Webhook 接收时间 |

### 日志状态说明

- 🔵 **等待中 (pending)** - 任务已创建，等待队列处理
- 🟡 **执行中 (processing)** - 正在执行部署
- 🟢 **成功 (success)** - 部署成功完成
- 🔴 **失败 (failed)** - 部署失败（查看消息了解原因）

---

## ❓ 常见问题

### Q: Webhook 没有触发？
**A:** 检查以下几点：
1. Webhook 配置是否已启用
2. 镜像名称和标签是否匹配
3. Registry 的 Webhook 是否配置正确
4. 查看 DockeRun 后端日志：`docker logs dockerun`

### Q: 签名验证失败？
**A:** 确保：
1. DockeRun 和 Registry 配置了相同的密钥
2. 密钥没有多余的空格或特殊字符
3. Registry 支持 HMAC-SHA256 签名

### Q: 部署脚本执行失败？
**A:** 
1. 查看 Webhook 日志的错误消息
2. 手动测试部署脚本是否可以成功执行
3. 检查脚本中的镜像名称、容器名称是否正确
4. 确保 DockeRun 容器有权限操作 Docker

### Q: 如何禁用某个 Webhook？
**A:** 在 Webhooks 页面，关闭配置左侧的开关即可。配置不会被删除，可以随时重新启用。

### Q: 多个 Webhook 同时触发会怎样？
**A:** 任务会按照接收顺序加入队列，由 3 个工作线程并发处理，不会相互阻塞。

---

## 🔧 高级配置

### 环境变量传递

Webhook 触发时，以下环境变量会自动注入到部署脚本：

| 变量 | 说明 | 示例 |
|------|------|------|
| `IMAGE_NAME` | 镜像名称 | `mycompany/myapp` |
| `IMAGE_TAG` | 镜像标签 | `latest` |
| `FULL_IMAGE` | 完整镜像名 | `mycompany/myapp:latest` |
| `DOCKER_NETWORK` | Docker 网络 | `dockerun-net` |

使用示例：

```bash
#!/bin/bash
echo "Deploying ${FULL_IMAGE}..."

docker run -d \
  --name my-app-${IMAGE_TAG} \
  --network ${DOCKER_NETWORK} \
  ${FULL_IMAGE}
```

### 多环境部署

可以创建多个 Webhook 配置，监听不同的标签：

| 配置名 | 镜像 | 标签 | 部署脚本 |
|--------|------|------|---------|
| `Deploy to Dev` | `myapp` | `dev` | `deploy-dev.sh` |
| `Deploy to Staging` | `myapp` | `staging` | `deploy-staging.sh` |
| `Deploy to Production` | `myapp` | `v*` | `deploy-prod.sh` |

---

## 📝 示例场景

### 场景 1: 简单的单容器应用

**需求：** 每次推送 `latest` 标签时，自动更新容器

**配置：**
- 镜像名称: `mycompany/webapp`
- 镜像标签: `latest`
- 清理旧容器: ✅
- 清理旧镜像: ✅

**部署脚本：**
```bash
#!/bin/bash
docker run -d \
  --name webapp \
  -p 80:80 \
  --restart unless-stopped \
  mycompany/webapp:latest
```

### 场景 2: 多容器应用（Docker Compose）

**需求：** 推送镜像后，重启整个 Compose 栈

**配置：**
- 镜像名称: `mycompany/api`
- 镜像标签: 留空（任意标签）
- 清理旧容器: ❌（由 Compose 管理）
- 清理旧镜像: ✅

**部署脚本：**
```bash
#!/bin/bash
cd /app/myproject

# 拉取最新镜像
docker-compose pull

# 重启服务
docker-compose up -d --force-recreate

# 清理未使用的镜像
docker image prune -f
```

### 场景 3: 蓝绿部署

**需求：** 先启动新容器，测试成功后停止旧容器

**部署脚本：**
```bash
#!/bin/bash
set -e

NEW_NAME="myapp-green"
OLD_NAME="myapp-blue"

# 启动新容器
docker run -d \
  --name ${NEW_NAME} \
  -p 8081:8080 \
  mycompany/myapp:${IMAGE_TAG}

# 健康检查
sleep 5
if curl -f http://localhost:8081/health; then
  echo "New container is healthy"
  
  # 切换流量（假设使用 Nginx 代理）
  # ...
  
  # 停止旧容器
  docker stop ${OLD_NAME} || true
  docker rm ${OLD_NAME} || true
  
  # 重命名新容器
  docker rename ${NEW_NAME} ${OLD_NAME}
else
  echo "Health check failed, rolling back"
  docker stop ${NEW_NAME}
  docker rm ${NEW_NAME}
  exit 1
fi
```

---

## 🛠️ 故障排查

### 查看后端日志

```bash
docker logs -f dockerun
```

关键日志前缀：
- `[Webhook]` - Webhook 相关日志
- `[Webhook] Worker N` - 异步任务执行日志

### 手动触发测试

使用 `curl` 模拟 Webhook 请求：

```bash
curl -X POST http://localhost:8080/webhooks/1 \
  -H "Content-Type: application/json" \
  -d '{
    "events": [{
      "action": "push",
      "target": {
        "repository": "mycompany/myapp",
        "tag": "latest"
      }
    }]
  }'
```

---

## 📚 参考资料

- [Docker Registry Webhook 规范](https://docs.docker.com/docker-hub/webhooks/)
- [Harbor Webhook 文档](https://goharbor.io/docs/latest/working-with-projects/project-configuration/configure-webhooks/)
- [GitLab Webhooks](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html)

---

**🎉 现在您可以享受全自动的 CI/CD 部署流程了！**

