# 🗄️ 私有 Docker Registry 管理指南

## 概述

Dockerun 提供了完整的私有 Docker Registry 管理功能，可以：
- ✅ 添加和管理多个私有 Registry
- ✅ 浏览私有 Registry 中的镜像和标签
- ✅ 搜索和筛选镜像
- ✅ 删除私有 Registry 中的镜像标签
- ✅ 支持认证和不安全连接

---

## 功能特性

### 1. Registry 管理

#### **添加 Registry**

支持添加私有 Docker Registry，配置项包括：

| 配置项 | 说明 | 必填 |
|--------|------|------|
| **名称** | Registry 的显示名称 | ✅ |
| **URL** | Registry 的完整 URL 地址 | ✅ |
| **用户名** | 认证用户名 | ❌ |
| **密码** | 认证密码 | ❌ |
| **邮箱** | 邮箱地址（可选） | ❌ |
| **设为默认** | 是否设为默认 Registry | ❌ |
| **不安全连接** | 允许不安全的 HTTPS 连接 | ❌ |

#### **编辑 Registry**

可以修改 Registry 的所有配置项，包括认证信息。

#### **删除 Registry**

删除 Registry 配置（不会影响 Registry 服务器上的数据）。

---

### 2. 镜像管理

#### **浏览镜像**

- 📋 查看 Registry 中所有镜像
- 🔍 搜索镜像名称（实时搜索）
- 📊 显示每个镜像的标签数量
- 📦 展开查看镜像的所有标签

#### **镜像标签操作**

- 📋 查看镜像的所有标签
- 🗑️ 删除指定标签（支持确认提示）
- 📋 复制完整镜像名称（包含 Registry 地址）

#### **搜索和筛选**

实时搜索镜像名称，支持模糊匹配。

---

## 使用指南

### 添加私有 Registry

#### **示例 1: Docker Hub 私有仓库**

```json
{
  "name": "Docker Hub Private",
  "url": "https://index.docker.io/v1/",
  "username": "your-username",
  "password": "your-password",
  "is_default": true,
  "insecure": false
}
```

#### **示例 2: Harbor 私有仓库**

```json
{
  "name": "Harbor Registry",
  "url": "https://harbor.example.com",
  "username": "admin",
  "password": "Harbor12345",
  "is_default": false,
  "insecure": false
}
```

#### **示例 3: 不安全的私有仓库**

```json
{
  "name": "Local Registry",
  "url": "http://192.168.1.100:5000",
  "username": "",
  "password": "",
  "is_default": false,
  "insecure": true
}
```

---

### 浏览和管理镜像

#### **1. 进入镜像列表**

在 Registry 管理页面，点击 "查看镜像" 按钮。

#### **2. 搜索镜像**

在搜索框中输入关键词，系统会自动筛选匹配的镜像。

```
搜索: "nginx"
结果: nginx, nginx-proxy, my-nginx-app
```

#### **3. 查看镜像标签**

点击表格行展开，查看该镜像的所有标签：

```
镜像: my-app/backend
标签: latest, v1.0.0, v1.0.1, dev
```

#### **4. 删除镜像标签**

点击标签右侧的 ❌ 按钮，确认后删除。

**⚠️ 注意事项：**
- 删除操作不可恢复
- 需要 Registry 启用 `DELETE` 方法
- 可能需要运行垃圾回收释放磁盘空间

#### **5. 复制镜像名称**

点击 "复制" 按钮，将完整镜像名称复制到剪贴板：

```
harbor.example.com/my-project/my-app:latest
```

可以直接用于 `docker pull` 或部署脚本。

---

## Docker Registry API 支持

### 支持的 API 版本

- ✅ Docker Registry HTTP API V2

### 实现的功能

| API 端点 | 功能 | 状态 |
|----------|------|------|
| `/v2/_catalog` | 获取镜像列表 | ✅ |
| `/v2/<name>/tags/list` | 获取镜像标签 | ✅ |
| `/v2/<name>/manifests/<tag>` | 获取 manifest | ✅ |
| `DELETE /v2/<name>/manifests/<digest>` | 删除镜像 | ✅ |

---

## 配置 Registry 服务器

### Harbor

Harbor 默认支持 Docker Registry API V2，无需额外配置。

```yaml
# harbor.yml
hostname: harbor.example.com
https:
  port: 443
  certificate: /path/to/cert.crt
  private_key: /path/to/cert.key
```

**启用删除功能：**

在 Harbor UI 中：
1. 进入项目设置
2. 启用 "允许内容删除"
3. 配置垃圾回收策略

### Docker Registry (官方)

#### **1. 启用删除功能**

```yaml
# config.yml
version: 0.1
storage:
  filesystem:
    rootdirectory: /var/lib/registry
  delete:
    enabled: true  # 启用删除功能
http:
  addr: :5000
```

#### **2. 启用认证**

```bash
# 创建认证文件
htpasswd -Bbn username password > /path/to/htpasswd

# Docker Compose 配置
services:
  registry:
    image: registry:2
    ports:
      - "5000:5000"
    volumes:
      - ./data:/var/lib/registry
      - ./auth:/auth
      - ./config.yml:/etc/docker/registry/config.yml
    environment:
      - REGISTRY_AUTH=htpasswd
      - REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd
      - REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm
      - REGISTRY_STORAGE_DELETE_ENABLED=true
```

#### **3. 运行垃圾回收**

删除镜像后，需要运行垃圾回收释放磁盘空间：

```bash
docker exec -it registry bin/registry garbage-collect /etc/docker/registry/config.yml
```

### Nexus Repository

Nexus 支持 Docker Registry API，需要：

1. 创建 Docker (hosted) Repository
2. 配置 HTTP/HTTPS 端口
3. 启用 Docker Bearer Token Realm

```
URL: https://nexus.example.com:8082
```

---

## 最佳实践

### 1. 命名规范

```
✅ 推荐:
- Production Harbor
- Dev Registry
- Local Test Registry

❌ 避免:
- registry1
- test
- 192.168.1.100
```

### 2. 安全配置

#### **使用 HTTPS**

```json
{
  "url": "https://registry.example.com",
  "insecure": false  // 使用安全连接
}
```

#### **配置认证**

始终为生产环境的 Registry 配置认证：

```json
{
  "username": "deploy-user",
  "password": "strong-password-here"
}
```

#### **最小权限原则**

为 Dockerun 创建专用的只读或有限权限账号：

```bash
# Harbor: 创建项目成员，分配 "Developer" 或 "Guest" 角色
# Docker Registry: 使用 ACL 限制访问
```

### 3. 镜像管理

#### **定期清理**

定期删除不再使用的镜像标签：

```
策略:
- 保留 latest 和最近 5 个版本
- 删除超过 90 天的开发版本
- 保留所有生产版本
```

#### **标签命名规范**

```
✅ 推荐:
- v1.0.0, v1.0.1 (语义化版本)
- 2024-12-10 (日期标签)
- latest, stable (环境标签)

❌ 避免:
- test1, test2
- final, final2
- 无意义的标签
```

### 4. 网络配置

#### **内网 Registry**

```json
{
  "url": "http://192.168.1.100:5000",
  "insecure": true
}
```

#### **通过代理访问**

确保后端服务器可以访问 Registry：

```bash
# 测试连接
curl -u username:password https://registry.example.com/v2/_catalog
```

---

## 故障排查

### 问题 1: 无法连接到 Registry

**错误信息**: "请求失败: dial tcp ..."

**解决方案**:
1. 检查 URL 是否正确
2. 确认网络连通性
3. 检查防火墙规则
4. 验证 HTTPS 证书（或启用 `insecure`）

```bash
# 测试连接
curl -v https://registry.example.com/v2/
```

### 问题 2: 认证失败

**错误信息**: "API 返回错误状态码: 401"

**解决方案**:
1. 验证用户名和密码
2. 检查账号权限
3. 确认 Registry 认证配置

```bash
# 测试认证
curl -u username:password https://registry.example.com/v2/_catalog
```

### 问题 3: 无法删除镜像

**错误信息**: "删除失败，状态码: 405"

**解决方案**:
1. 确认 Registry 启用了删除功能
2. 检查用户权限
3. 验证 API 版本支持

```yaml
# Docker Registry 配置
storage:
  delete:
    enabled: true
```

### 问题 4: 镜像列表为空

**可能原因**:
1. Registry 中确实没有镜像
2. 权限不足，无法列出镜像
3. URL 配置错误

**解决方案**:

```bash
# 手动测试
curl -u username:password https://registry.example.com/v2/_catalog

# 推送测试镜像
docker tag busybox registry.example.com/test:latest
docker push registry.example.com/test:latest
```

### 问题 5: 不安全的 HTTPS 连接

**错误信息**: "x509: certificate signed by unknown authority"

**解决方案**:

**方式 1**: 启用 `insecure` 选项

```json
{
  "insecure": true
}
```

**方式 2**: 添加 CA 证书到系统信任列表

```bash
# Ubuntu/Debian
sudo cp ca.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates

# CentOS/RHEL
sudo cp ca.crt /etc/pki/ca-trust/source/anchors/
sudo update-ca-trust
```

---

## 示例场景

### 场景 1: 公司内网 Harbor

```
配置:
  名称: Company Harbor
  URL: https://harbor.company.com
  用户名: dockerun-service
  密码: ********
  默认: 是

操作:
1. 添加 Registry
2. 浏览镜像列表
3. 搜索项目镜像
4. 清理旧版本标签
```

### 场景 2: 开发环境本地 Registry

```
配置:
  名称: Local Dev Registry
  URL: http://localhost:5000
  用户名: (空)
  密码: (空)
  不安全连接: 是

操作:
1. 启动本地 Registry
2. 添加到 Dockerun
3. 推送开发镜像
4. 测试部署
```

### 场景 3: 多环境管理

```
生产环境:
  名称: Prod Harbor
  URL: https://harbor-prod.company.com
  默认: 是

测试环境:
  名称: Test Registry
  URL: https://harbor-test.company.com
  默认: 否

开发环境:
  名称: Dev Registry
  URL: http://dev-registry:5000
  不安全连接: 是
```

---

## API 参考

### 后端 API

#### **1. 获取 Registry 列表**

```http
GET /api/registries
Authorization: Bearer <token>
```

响应：

```json
[
  {
    "id": 1,
    "name": "Harbor",
    "url": "https://harbor.example.com",
    "username": "admin",
    "is_default": true,
    "insecure": false,
    "created_at": "2024-12-10T10:00:00Z"
  }
]
```

#### **2. 添加 Registry**

```http
POST /api/registries
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "My Registry",
  "url": "https://registry.example.com",
  "username": "user",
  "password": "pass",
  "is_default": false,
  "insecure": false
}
```

#### **3. 获取镜像列表**

```http
GET /api/registries/1/images?search=nginx
Authorization: Bearer <token>
```

响应：

```json
[
  {
    "name": "nginx",
    "tags": ["latest", "1.21", "alpine"]
  },
  {
    "name": "my-app/nginx-proxy",
    "tags": ["v1.0.0", "latest"]
  }
]
```

#### **4. 删除镜像标签**

```http
DELETE /api/registries/1/images/my-app?tag=v1.0.0
Authorization: Bearer <token>
```

---

## 扩展功能

### 计划中的功能

- 🔄 镜像推送功能
- 📊 镜像大小统计
- 🏷️ 批量删除标签
- 📈 使用情况报表
- 🔐 Registry 连接测试
- 📋 镜像漏洞扫描（集成 Trivy）

---

## 参考资源

- [Docker Registry HTTP API V2](https://docs.docker.com/registry/spec/api/)
- [Harbor 文档](https://goharbor.io/docs/)
- [Docker Registry 官方文档](https://docs.docker.com/registry/)
- [Nexus Repository Manager](https://help.sonatype.com/repomanager3)

---

更多信息请参考：
- [CONFIGURATION.md](CONFIGURATION.md) - 配置说明
- [README.md](README.md) - 项目概述

