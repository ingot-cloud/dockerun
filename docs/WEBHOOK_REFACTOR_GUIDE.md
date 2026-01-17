# Webhook 自动部署功能重构指南

> **团队**: Ingot Cloud  
> **作者**: JY  
> **日期**: 2026-01-17  
> **版本**: v2.0

## 📋 目录

- [概述](#概述)
- [核心设计](#核心设计)
- [工作流程](#工作流程)
- [配置说明](#配置说明)
- [使用示例](#使用示例)
- [API 接口](#api-接口)
- [安全性](#安全性)
- [故障排查](#故障排查)

## 概述

### 设计理念

Webhook 自动部署功能允许 Docker Registry 在镜像推送时自动触发部署脚本。系统采用**一对多**的关联模式：
- **一个 Webhook 配置** ←→ **一个 Docker Registry**
- **一个镜像推送事件** ←→ **多个部署脚本**（通过镜像名称和标签匹配）

### 核心特性

1. ✅ **Secret 验证**：使用 HMAC-SHA256 签名验证，确保 Webhook 请求来自可信的 Registry
2. ✅ **智能匹配**：根据镜像名称和标签自动匹配多个部署脚本
3. ✅ **异步执行**：快速响应 Registry，避免超时；后台队列异步处理部署任务
4. ✅ **通配符支持**：支持 `*` 通配符匹配镜像名称和标签
5. ✅ **多平台支持**：兼容 Docker Hub、Harbor、GitLab Registry、自建 Registry

## 核心设计

### 数据模型

#### WebhookConfig（Webhook 配置）

```go
type WebhookConfig struct {
    ID          uint      `gorm:"primarykey" json:"id"`
    Name        string    `json:"name"`              // 配置名称
    Description string    `json:"description"`       // 描述
    Enabled     bool      `json:"enabled"`           // 是否启用
    RegistryID  uint      `json:"registry_id"`       // 关联的 Registry
    Secret      string    `json:"secret"`            // HMAC 签名密钥
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

**关键点**：
- `Secret` 用于验证 Webhook 请求的签名
- `RegistryID` 关联到具体的 Docker Registry
- Webhook URL 格式：`http://your-domain/webhooks/{webhook_id}`

#### DeployScript（部署脚本）

```go
type DeployScript struct {
    ID         uint                 `gorm:"primarykey" json:"id"`
    Name       string               `json:"name"`
    ImageName  string               `json:"image_name"`   // 匹配的镜像名（支持通配符）
    ImageTag   string               `json:"image_tag"`    // 匹配的标签（支持通配符）
    AutoDeploy bool                 `json:"auto_deploy"`  // 是否启用自动部署
    MainScript string               `json:"main_script"`  // 主脚本内容
    Files      []DeployScriptFile   `gorm:"-" json:"files"`
    EnvVars    map[string]string    `gorm:"type:json" json:"env_vars"`
    WorkDir    string               `json:"work_dir"`
    // ... 其他字段
}
```

**关键点**：
- `ImageName` 和 `ImageTag` 用于匹配推送的镜像
- `AutoDeploy=true` 时，脚本会被 Webhook 自动触发
- 支持通配符：`indohit/*` 匹配所有 `indohit` 命名空间的镜像

### 匹配规则

镜像匹配支持以下几种模式：

1. **精确匹配**：
   ```
   Pattern: indohit/gateway
   Matches: indohit/gateway
   ```

2. **通配符匹配**：
   ```
   Pattern: indohit/*
   Matches: indohit/gateway, indohit/member, indohit/auth
   ```

3. **去除 Registry 地址匹配**：
   ```
   Pattern: 192.168.0.170:5000/indohit/gateway
   Webhook Payload: indohit/gateway
   结果: 匹配（自动去除 Registry 地址）
   ```

4. **标签匹配**：
   ```
   Pattern: v*
   Matches: v1.0.0, v2.3.4, v1.0.0-beta
   
   Pattern: latest
   Matches: latest
   
   Pattern: *
   Matches: 任意标签
   ```

## 工作流程

### 1. Webhook 接收

```
Registry Push Event → POST /webhooks/{webhook_id}
                    ↓
            快速响应 200 OK
                    ↓
            异步处理（goroutine）
```

### 2. 请求验证

```go
// 支持多种签名头
signature := headers.Get("X-Registry-Signature")  // Harbor
if signature == "" {
    signature = headers.Get("X-Hub-Signature-256") // Docker Hub
}
if signature == "" {
    signature = headers.Get("X-Gitlab-Token")      // GitLab
}

// HMAC-SHA256 验证
mac := hmac.New(sha256.New, []byte(secret))
mac.Write(body)
calculatedHash := hex.EncodeToString(mac.Sum(nil))
```

### 3. 脚本匹配与执行

```
解析 Webhook Payload
        ↓
提取镜像名称和标签
        ↓
查询启用自动部署的脚本
        ↓
匹配镜像名称和标签（支持通配符）
        ↓
为每个匹配的脚本创建部署任务
        ↓
任务加入队列（异步执行）
        ↓
Worker 处理任务：
  1. 拉取新镜像
  2. 删除旧容器（可选）
  3. 删除旧镜像（可选）
  4. 执行部署脚本
```

### 4. 日志记录

每个部署任务都会创建 `WebhookLog` 记录：

```go
type WebhookLog struct {
    ID              uint       `gorm:"primarykey" json:"id"`
    WebhookConfigID uint       `json:"webhook_config_id"` // 关联的 Webhook 配置
    ImageName       string     `json:"image_name"`
    ImageTag        string     `json:"image_tag"`
    Status          string     `json:"status"`            // pending, processing, success, failed
    Message         string     `json:"message"`
    StartedAt       *time.Time `json:"started_at"`
    CompletedAt     *time.Time `json:"completed_at"`
    ExecutionTime   int        `json:"execution_time"`    // 秒
    CreatedAt       time.Time  `json:"created_at"`
}
```

## 配置说明

### 步骤 1：创建 Registry

在 Dockerun 中添加 Docker Registry：

```json
{
  "name": "My Harbor Registry",
  "url": "https://harbor.example.com",
  "username": "admin",
  "password": "Harbor12345",
  "insecure": false
}
```

### 步骤 2：创建 Webhook 配置

```json
{
  "name": "Ingot 微服务自动部署",
  "description": "当 Ingot 微服务镜像推送时自动部署",
  "enabled": true,
  "registry_id": 1,
  "secret": "my-secret-key-123"
}
```

**注意**：
- `secret` 必须与 Registry 配置中的密钥一致
- 记录生成的 Webhook URL（如：`http://dockerun.example.com/webhooks/1`）

### 步骤 3：配置部署脚本

为每个需要自动部署的服务创建部署脚本：

```json
{
  "name": "Ingot Gateway 部署",
  "image_name": "indohit/gateway",
  "image_tag": "latest",
  "auto_deploy": true,
  "main_script": "#!/bin/bash\n...",
  "env_vars": {
    "CONTAINER_NAME": "ingot-gateway",
    "PORT": "8080"
  }
}
```

**通配符示例**：

```json
{
  "name": "所有 Ingot 服务",
  "image_name": "indohit/*",
  "image_tag": "*",
  "auto_deploy": true
}
```

### 步骤 4：配置 Registry Webhook

#### Harbor

1. 进入项目设置 → Webhooks
2. 点击 "New Webhook"
3. 配置：
   ```
   Endpoint URL: http://dockerun.example.com/webhooks/1
   Auth Header: (留空)
   Event Types: ☑ Artifact pushed
   ```
4. 如果配置了 Secret，需要在 "HTTP Headers" 中添加：
   ```
   X-Registry-Signature: your-secret-key
   ```

#### Docker Hub

1. 进入 Repository → Webhooks
2. 配置：
   ```
   Webhook URL: http://dockerun.example.com/webhooks/1
   ```

#### GitLab Registry

1. 进入项目设置 → Webhooks
2. 配置：
   ```
   URL: http://dockerun.example.com/webhooks/1
   Secret Token: your-secret-key
   Trigger: Registry events
   ```

#### 自建 Registry

编辑 Registry 的 `config.yml`：

```yaml
notifications:
  endpoints:
    - name: dockerun
      url: http://dockerun.example.com/webhooks/1
      headers:
        X-Registry-Signature: [your-secret-key]
      timeout: 5s
      threshold: 5
      backoff: 1s
```

## 使用示例

### 示例 1：单服务自动部署

**场景**：每次推送 `indohit/gateway:latest` 时自动部署。

**配置**：

1. 创建 Webhook：
   ```json
   {
     "name": "Gateway Auto Deploy",
     "registry_id": 1,
     "secret": "gateway-secret",
     "enabled": true
   }
   ```

2. 创建部署脚本：
   ```json
   {
     "name": "Gateway Deploy",
     "image_name": "indohit/gateway",
     "image_tag": "latest",
     "auto_deploy": true,
     "main_script": "#!/bin/bash\ndocker-compose up -d"
   }
   ```

3. 推送镜像：
   ```bash
   docker push indohit/gateway:latest
   ```

**结果**：部署脚本自动执行，容器重启。

### 示例 2：多服务批量部署

**场景**：所有 `indohit/*` 服务推送时都触发对应的部署脚本。

**配置**：

1. 创建 Webhook（同上）

2. 为每个服务创建部署脚本：
   ```json
   // Gateway
   {
     "name": "Gateway Deploy",
     "image_name": "indohit/gateway",
     "image_tag": "*",
     "auto_deploy": true
   }
   
   // Auth
   {
     "name": "Auth Deploy",
     "image_name": "indohit/auth",
     "image_tag": "*",
     "auto_deploy": true
   }
   
   // Member
   {
     "name": "Member Deploy",
     "image_name": "indohit/member",
     "image_tag": "*",
     "auto_deploy": true
   }
   ```

3. 推送任意服务：
   ```bash
   docker push indohit/gateway:v1.0.0
   docker push indohit/auth:v1.0.1
   ```

**结果**：对应的部署脚本自动执行。

### 示例 3：版本控制部署

**场景**：只有 `v*` 版本标签触发生产部署，`dev-*` 触发开发环境部署。

**配置**：

```json
// 生产环境
{
  "name": "Production Gateway",
  "image_name": "indohit/gateway",
  "image_tag": "v*",
  "auto_deploy": true,
  "env_vars": {
    "ENVIRONMENT": "production"
  }
}

// 开发环境
{
  "name": "Dev Gateway",
  "image_name": "indohit/gateway",
  "image_tag": "dev-*",
  "auto_deploy": true,
  "env_vars": {
    "ENVIRONMENT": "development"
  }
}
```

## API 接口

### Webhook 配置管理

```http
# 创建 Webhook 配置
POST /api/webhook-configs
Content-Type: application/json
{
  "name": "My Webhook",
  "registry_id": 1,
  "secret": "my-secret",
  "enabled": true
}

# 获取 Webhook URL
GET /api/webhook-configs/{id}/url
Response:
{
  "url": "http://dockerun.example.com/webhooks/1",
  "note": "Configure this URL in your Docker Registry webhook settings"
}

# 列出所有配置
GET /api/webhook-configs

# 更新配置
PUT /api/webhook-configs/{id}

# 删除配置
DELETE /api/webhook-configs/{id}
```

### Webhook 日志

```http
# 列出日志
GET /api/webhook-logs?config_id={webhook_config_id}

# 日志响应
[
  {
    "id": 1,
    "webhook_config_id": 1,
    "image_name": "indohit/gateway",
    "image_tag": "latest",
    "status": "success",
    "message": "Successfully deployed indohit/gateway:latest",
    "execution_time": 15,
    "created_at": "2026-01-17T10:00:00Z",
    "completed_at": "2026-01-17T10:00:15Z"
  }
]
```

### 接收 Webhook（公开接口）

```http
# Registry 调用此接口推送事件
POST /webhooks/{webhook_id}
Content-Type: application/json
X-Registry-Signature: sha256=abc123...

{
  "events": [
    {
      "action": "push",
      "target": {
        "repository": "indohit/gateway",
        "tag": "latest"
      }
    }
  ]
}
```

## 安全性

### 1. Secret 签名验证

所有 Webhook 请求都应配置 Secret，系统会验证签名：

```go
// 支持多种签名算法
if !verifySignature(body, webhookConfig.Secret, signature) {
    log.Printf("[Webhook] Signature verification failed")
    return
}
```

### 2. Registry 地址验证

系统会验证 Webhook 配置关联的 Registry 是否存在：

```go
var registry models.Registry
if err := db.First(&registry, webhookConfig.RegistryID).Error; err != nil {
    return
}
```

### 3. HTTPS 支持

生产环境强烈建议使用 HTTPS：

```
Webhook URL: https://dockerun.example.com/webhooks/1
```

### 4. IP 白名单（可选）

可以在反向代理（如 Nginx）中配置 IP 白名单：

```nginx
location /webhooks {
    allow 192.168.1.0/24;  # Registry 所在网段
    deny all;
    proxy_pass http://dockerun:8080;
}
```

## 故障排查

### 1. Webhook 未触发

**检查项**：
- ✅ Webhook 配置是否启用（`enabled: true`）
- ✅ Registry 配置是否正确
- ✅ Webhook URL 是否可访问
- ✅ Registry 能否访问 Dockerun 服务（网络连通性）

**查看日志**：
```bash
# 后端日志
docker logs dockerun | grep Webhook

# Registry 日志（Harbor）
docker logs harbor-core | grep webhook
```

### 2. 签名验证失败

**检查项**：
- ✅ Secret 是否一致（区分大小写）
- ✅ Registry 是否正确发送签名头
- ✅ 签名格式是否正确（`sha256=xxx` 或 `sha256:xxx`）

**调试**：
```go
log.Printf("[Webhook] Expected signature: %s", calculatedHash)
log.Printf("[Webhook] Received signature: %s", signature)
```

### 3. 部署脚本未匹配

**检查项**：
- ✅ 部署脚本的 `auto_deploy` 是否为 `true`
- ✅ 镜像名称和标签是否匹配
- ✅ 通配符是否正确使用

**查看匹配日志**：
```bash
docker logs dockerun | grep "Found.*matching scripts"
```

### 4. 任务队列已满

**检查项**：
- ✅ 队列缓冲是否足够（默认 100）
- ✅ Worker 数量是否足够（默认 3）

**调整配置**：
```go
// backend/main.go
webhook.InitQueue(5) // 增加 worker 数量
```

### 5. 镜像拉取失败

**检查项**：
- ✅ Registry 认证信息是否正确
- ✅ 镜像名称是否完整（包含 registry 地址）
- ✅ 网络连接是否正常
- ✅ 是否配置了 insecure registry

**查看错误信息**：
```bash
docker logs dockerun | grep "Failed to pull image"
```

## 总结

Webhook 自动部署功能的核心优势：

1. ✅ **简化配置**：一个 Webhook 配置关联一个 Registry，通过脚本匹配实现灵活部署
2. ✅ **安全可靠**：HMAC 签名验证，确保请求来源可信
3. ✅ **高性能**：异步任务队列，快速响应，避免超时
4. ✅ **灵活匹配**：支持通配符，一次配置，多服务自动部署
5. ✅ **完整日志**：记录每次部署的详细信息，便于追踪和调试

---

**Ingot Cloud - JY**  
*让 Docker 部署更简单*
