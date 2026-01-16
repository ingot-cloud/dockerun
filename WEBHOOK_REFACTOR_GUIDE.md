# Webhook 多脚本关联重构指南

## 概述

本次重构将 Webhook 从"一对一"模式（一个 Webhook 配置对应一个部署脚本）升级为"一对多"模式（一个 Registry 的 Webhook 可以触发多个匹配的部署脚本）。

## 核心改进

### 1. 数据模型变化

#### DeployScript 新增字段
```typescript
interface DeployScript {
  // ... 现有字段 ...
  
  // 新增：Webhook 自动部署配置
  auto_deploy?: boolean   // 是否启用自动部署
  image_name?: string     // 镜像名称（如：namespace/myapp）
  image_tag?: string      // 镜像标签（如：latest，留空=任意标签）
}
```

#### WebhookConfig 字段变化
```typescript
interface WebhookConfig {
  // 保留字段
  registry_id: number
  enabled: boolean
  secret?: string
  
  // 已废弃字段（保留兼容）
  image_name?: string        // 不再使用
  image_tag?: string         // 不再使用
  deploy_script_id?: number  // 不再使用
}
```

### 2. 工作流程变化

#### 旧模式
```
Registry Push → Webhook URL(/webhooks/{configId}) 
  → 匹配 WebhookConfig 
  → 执行单个部署脚本
```

#### 新模式
```
Registry Push → Webhook URL(/webhooks/registry/{registryId})
  → 查找所有匹配的 DeployScript (auto_deploy=true AND image_name匹配 AND image_tag匹配)
  → 并发执行多个部署脚本
```

### 3. 镜像匹配逻辑

```typescript
// 精确匹配
image_name = "namespace/myapp" AND image_tag = "latest"

// 通配标签
image_name = "namespace/myapp" AND (image_tag = "" OR image_tag = "*")
→ 匹配任意标签

// 示例
DeployScript {
  name: "Deploy MyApp Production",
  image_name: "namespace/myapp",
  image_tag: "latest",
  auto_deploy: true
}

DeployScript {
  name: "Deploy MyApp Staging",
  image_name: "namespace/myapp",
  image_tag: "staging",
  auto_deploy: true
}

// 当推送 namespace/myapp:latest 时，只触发第一个脚本
// 当推送 namespace/myapp:staging 时，只触发第二个脚本
```

## 使用指南

### 配置步骤

#### 1. 配置部署脚本的自动部署

在 "Deploy Scripts" 页面创建或编辑脚本：

1. 填写脚本基本信息
2. 展开 "🔔 Webhook Auto Deploy" 部分
3. 启用 "Enable Auto Deploy"
4. 填写 "Image Name"：如 `docker.io/library/nginx` 或 `myregistry.com/myapp`
5. 填写 "Image Tag"（可选）：
   - 精确匹配：`latest`, `v1.0.0`
   - 通配匹配：留空或填 `*`
6. 保存

#### 2. 配置 Registry 的 Webhook

在 "Webhooks" 页面：

1. 点击 "Add Webhook"
2. 选择 Registry
3. 设置 Secret（可选，用于验证）
4. 保存后，复制 Webhook URL
5. 在 Registry 设置中配置此 Webhook URL

#### 3. Webhook URL 格式

```
新模式（推荐）：
http://your-server:19000/webhooks/registry/{registryId}

旧模式（兼容）：
http://your-server:19000/webhooks/{webhookConfigId}
```

### Registry 配置示例

#### Docker Registry (config.yml)
```yaml
notifications:
  endpoints:
    - name: dockerun
      url: http://your-server:19000/webhooks/registry/1
      headers:
        X-Registry-Signature: [your-secret]
      timeout: 5s
      threshold: 5
      backoff: 1s
```

#### Harbor
1. 进入项目 → Webhooks
2. 添加 Webhook
3. Endpoint URL: `http://your-server:19000/webhooks/registry/1`
4. Auth Header: 留空（或配置 Secret）
5. 触发事件：选择 "Push Artifact"

#### GitLab Container Registry
1. 项目 → Settings → Webhooks
2. URL: `http://your-server:19000/webhooks/registry/1`
3. Secret Token: 输入你的 Secret
4. Trigger: 选择 "Push events"

## API 变化

### 新增 Webhook Endpoint

```
POST /webhooks/registry/:registryId
Body: Registry Webhook Payload (JSON)
Response: 200 OK { "status": "received" }
```

### DeployScript API 更新

创建/更新脚本时，支持新字段：

```typescript
POST /api/deploy-scripts
PUT /api/deploy-scripts/:id

Body: {
  name: string,
  main_script: string,
  // ... 其他字段 ...
  
  // 新增字段
  auto_deploy?: boolean,
  image_name?: string,
  image_tag?: string
}
```

## 前端变化

### DeployScripts.vue

添加了 "Webhook Auto Deploy" 折叠面板：

```vue
<el-collapse>
  <el-collapse-item title="🔔 Webhook Auto Deploy">
    <el-switch v-model="form.auto_deploy" />
    <el-input v-model="form.image_name" />
    <el-input v-model="form.image_tag" />
  </el-collapse-item>
</el-collapse>
```

### WebhookConfigs.vue（建议简化）

**简化前**：
- 需要选择镜像名称
- 需要选择镜像标签
- 需要选择部署脚本

**简化后**：
- 只需选择 Registry
- 自动显示关联的脚本（从 DeployScript 查询）

## 数据库迁移

新字段会自动迁移（GORM AutoMigrate）：

```sql
-- deploy_scripts 表新增字段
ALTER TABLE deploy_scripts ADD COLUMN auto_deploy BOOLEAN DEFAULT FALSE;
ALTER TABLE deploy_scripts ADD COLUMN image_name VARCHAR(500);
ALTER TABLE deploy_scripts ADD COLUMN image_tag VARCHAR(100);

-- webhook_configs 表字段变更（标记为可选）
-- 旧字段保留，但不再强制要求
```

## 兼容性

### 向后兼容

- 旧的 `/webhooks/:id` 路由仍然可用
- 旧的 WebhookConfig 配置仍然有效
- 现有 Webhook 不受影响

### 迁移建议

1. **新项目**：直接使用新模式
2. **现有项目**：
   - 保持现有 Webhook 配置不变
   - 新增 Webhook 使用新模式
   - 逐步迁移旧配置到新模式

### 迁移步骤

```bash
# 1. 在 DeployScript 中配置镜像信息
- 编辑部署脚本
- 启用 Auto Deploy
- 填写 image_name 和 image_tag

# 2. 更新 Registry Webhook URL
- 从 /webhooks/{oldId}
- 改为 /webhooks/registry/{registryId}

# 3. 删除旧的 WebhookConfig（可选）
- 旧配置不再需要
```

## 测试

### 测试场景

1. **单脚本触发**
   - 推送 `myapp:latest`
   - 验证匹配的脚本执行

2. **多脚本触发**
   - 配置多个脚本监听同一镜像
   - 推送镜像
   - 验证所有匹配的脚本并发执行

3. **通配标签**
   - 配置 `image_tag = ""`
   - 推送任意标签
   - 验证脚本触发

4. **精确匹配**
   - 配置 `image_tag = "v1.0.0"`
   - 推送 `v1.0.0` → 触发
   - 推送 `v2.0.0` → 不触发

### 测试脚本

```bash
# 1. 配置测试脚本
curl -X POST http://localhost:19000/api/deploy-scripts \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Test Auto Deploy",
    "main_script": "echo 'Triggered!'",
    "auto_deploy": true,
    "image_name": "test/myapp",
    "image_tag": "latest"
  }'

# 2. 模拟 Webhook 推送
curl -X POST http://localhost:19000/webhooks/registry/1 \
  -H "Content-Type: application/json" \
  -d '{
    "events": [{
      "action": "push",
      "target": {
        "repository": "test/myapp",
        "tag": "latest"
      }
    }]
  }'

# 3. 查看日志
docker logs -f dockerun
```

## 常见问题

### Q1: 一个镜像推送会触发多少个脚本？

A: 所有满足以下条件的脚本都会被触发：
- `auto_deploy = true`
- `image_name` 精确匹配
- `image_tag` 匹配（精确或通配）

### Q2: 如何避免重复触发？

A: 确保每个脚本的 `image_name` 和 `image_tag` 组合是唯一的。

### Q3: 旧的 Webhook 配置还能用吗？

A: 可以，旧的 `/webhooks/:id` 路由仍然支持。

### Q4: 如何查看哪些脚本会被触发？

A: 在部署脚本列表中，可以添加一个 "Auto Deploy" 标识列，显示启用了自动部署的脚本。

### Q5: Webhook 验证（Secret）如何工作？

A: 
- 新模式：暂不支持 Registry 级别的验证（TODO）
- 旧模式：每个 WebhookConfig 可以配置独立的 Secret

## 后续优化

- [ ] 添加 Webhook 执行历史查询（按 Registry 或脚本过滤）
- [ ] 支持镜像名称通配符（如 `namespace/*`）
- [ ] Registry 级别的 Secret 验证
- [ ] Webhook 触发频率限制
- [ ] 脚本执行优先级
- [ ] 条件触发（如：仅工作时间触发）

## 参考资料

- [Docker Registry Webhook Spec](https://docs.docker.com/registry/notifications/)
- [Harbor Webhook Guide](https://goharbor.io/docs/latest/working-with-projects/project-configuration/configure-webhooks/)
- [GitLab Container Registry](https://docs.gitlab.com/ee/user/packages/container_registry/)
