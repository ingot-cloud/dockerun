# Scripts 目录说明

本目录包含各种实用脚本和工具。

## 📂 目录结构

```
scripts/
├── README.md                          # 本文件
├── docker-run-template.sh            # 通用 Docker 容器运行模板 ⭐
├── DOCKER_RUN_TEMPLATE_GUIDE.md      # 模板脚本使用指南
├── setup-insecure-registry.sh        # 配置 Docker insecure registry
├── setup-registry-https.sh           # 配置 Registry HTTPS 证书
├── backup.sh                          # 数据备份脚本
├── restore.sh                         # 数据恢复脚本
├── update.sh                          # 系统更新脚本
├── install-docker.sh                  # Docker 安装脚本
└── examples/                          # 配置文件示例
    ├── nginx.conf                     # Nginx 配置示例
    ├── mysql.conf                     # MySQL 配置示例
    ├── redis.conf                     # Redis 配置示例
    ├── postgresql.conf                # PostgreSQL 配置示例
    └── mongodb.conf                   # MongoDB 配置示例
```

---

## 🌟 Docker 运行模板脚本（推荐）

### 特性

- ✅ **通用性强**：适用于任何 Docker 容器
- ✅ **灵活配置**：支持配置文件、环境变量、交互式三种方式
- ✅ **功能完整**：启动、停止、重启、删除、日志、进入容器
- ✅ **易于使用**：简单明了的命令行界面
- ✅ **生产就绪**：支持健康检查、资源限制、日志管理

### 快速开始

```bash
# 交互式配置（最简单）
./docker-run-template.sh interactive

# 生成示例配置
./docker-run-template.sh generate-config

# 使用配置文件
./docker-run-template.sh -c nginx.conf start

# 查看帮助
./docker-run-template.sh help
```

### 详细文档

查看 [DOCKER_RUN_TEMPLATE_GUIDE.md](./DOCKER_RUN_TEMPLATE_GUIDE.md) 获取完整使用指南。

---

## 📋 配置文件示例

在 `examples/` 目录下提供了常用服务的配置示例：

### 1. Nginx Web 服务器

```bash
./docker-run-template.sh -c examples/nginx.conf start
```

功能：
- HTTP/HTTPS 服务
- 静态文件服务
- 反向代理

### 2. MySQL 数据库

```bash
./docker-run-template.sh -c examples/mysql.conf start
```

功能：
- 关系型数据库
- 数据持久化
- 健康检查

### 3. Redis 缓存

```bash
./docker-run-template.sh -c examples/redis.conf start
```

功能：
- 内存缓存
- 数据持久化
- 自定义配置

### 4. PostgreSQL 数据库

```bash
./docker-run-template.sh -c examples/postgresql.conf start
```

功能：
- 高级关系型数据库
- 完整 SQL 支持
- 数据持久化

### 5. MongoDB 数据库

```bash
./docker-run-template.sh -c examples/mongodb.conf start
```

功能：
- NoSQL 文档数据库
- 数据持久化
- 认证配置

---

## 🔧 其他实用脚本

### Registry 配置脚本

#### setup-insecure-registry.sh

配置 Docker daemon 允许不安全的 HTTP Registry。

```bash
sudo ./setup-insecure-registry.sh 192.168.0.170:5000
```

#### setup-registry-https.sh

生成自签名 HTTPS 证书并配置 Registry。

```bash
sudo ./setup-registry-https.sh 192.168.0.170 5000
```

### 备份和恢复

#### backup.sh

备份 Dockerun 数据。

```bash
./backup.sh
```

#### restore.sh

恢复 Dockerun 数据。

```bash
./restore.sh backup-file.tar.gz
```

### 系统维护

#### update.sh

更新 Dockerun 到最新版本。

```bash
./update.sh
```

#### install-docker.sh

在服务器上安装 Docker。

```bash
./install-docker.sh
```

---

## 💡 使用技巧

### 1. 创建自己的配置

```bash
# 复制示例配置
cp examples/nginx.conf myapp.conf

# 编辑配置
vi myapp.conf

# 启动容器
./docker-run-template.sh -c myapp.conf start
```

### 2. 批量管理容器

```bash
# 创建管理脚本
cat > manage-all.sh << 'EOF'
#!/bin/bash
for conf in configs/*.conf; do
    ./docker-run-template.sh -c "$conf" "$1"
done
EOF

chmod +x manage-all.sh

# 批量启动
./manage-all.sh start

# 批量停止
./manage-all.sh stop
```

### 3. 集成到 CI/CD

```yaml
# .github/workflows/deploy.yml
steps:
  - name: Deploy Container
    run: |
      ./docker-run-template.sh -c production.conf start
```

### 4. 环境变量覆盖

```bash
# 快速修改配置
PORTS="8081:80" \
MEMORY_LIMIT="1g" \
./docker-run-template.sh -c nginx.conf start
```

---

## 📚 相关文档

- [Docker 官方文档](https://docs.docker.com/)
- [INSECURE_REGISTRY.md](../INSECURE_REGISTRY.md) - Registry HTTP 配置
- [SELF_SIGNED_CERTIFICATE.md](../SELF_SIGNED_CERTIFICATE.md) - 自签名证书指南
- [MYSQL_SETUP.md](../MYSQL_SETUP.md) - MySQL 数据库配置

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

如果你有好的配置示例，欢迎分享到 `examples/` 目录。

---

## 📄 许可证

MIT License

