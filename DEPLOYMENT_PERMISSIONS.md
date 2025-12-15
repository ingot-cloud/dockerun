# 部署权限配置说明

## 问题描述

在服务器上部署时，添加部署脚本后无法创建文件，这是因为：

1. **工作目录不存在** - 默认工作目录 `/ingot-data/dockerun` 在容器中不存在
2. **没有挂载卷** - 工作目录没有挂载到宿主机，数据无法持久化
3. **权限问题** - 容器内创建的文件权限可能与宿主机不匹配

## 解决方案

### 1. Dockerfile 修改

在 `Dockerfile.multiarch` 中添加工作目录创建：

```dockerfile
# 创建数据目录和部署工作目录
RUN mkdir -p /app/data \
    && mkdir -p /ingot-data/dockerun \
    && chmod 755 /ingot-data/dockerun
```

### 2. 运行脚本修改

在 `run.sh` 中添加卷挂载：

```bash
# 创建宿主机目录
mkdir -p /ingot-data/dockerun

# 挂载工作目录
docker run -d \
    --name dockerun \
    -p 19000:8080 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /ingot-data/dockerun:/ingot-data/dockerun \  # ← 新增
    --restart unless-stopped \
    docker-registry.ingotcloud.top/ingot/dockerun:0.1.0
```

### 3. 手动运行时的配置

如果手动运行容器，确保添加工作目录挂载：

```bash
# 1. 创建宿主机目录
sudo mkdir -p /ingot-data/dockerun
sudo chmod 755 /ingot-data/dockerun

# 2. 运行容器（添加卷挂载）
docker run -d \
    --name dockerun \
    -p 19000:8080 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /ingot-data/dockerun:/ingot-data/dockerun \
    --restart unless-stopped \
    docker-registry.ingotcloud.top/ingot/dockerun:0.1.0
```

## 目录结构

```
/ingot-data/dockerun/          # 工作目录根
├── script-1/                  # 脚本1的工作目录
│   ├── deploy.sh             # 部署脚本
│   ├── files/                # 关联文件
│   │   ├── config.yml
│   │   └── docker-compose.yml
│   └── logs/                 # 执行日志
├── script-2/                  # 脚本2的工作目录
│   └── ...
└── ...
```

## 权限说明

### 容器内权限

- **用户**: root (UID 0)
- **工作目录权限**: 755 (rwxr-xr-x)
- **创建的文件权限**: 644 (rw-r--r--)
- **创建的目录权限**: 755 (rwxr-xr-x)

### 宿主机权限

挂载后，宿主机上的文件权限：

```bash
# 查看权限
ls -la /ingot-data/dockerun

# 如果需要修改权限
sudo chown -R root:root /ingot-data/dockerun
sudo chmod -R 755 /ingot-data/dockerun
```

## 常见问题

### Q1: 文件创建失败，提示 "Permission denied"

**原因**: 工作目录不存在或没有写权限

**解决**:
```bash
# 检查目录是否存在
ls -ld /ingot-data/dockerun

# 创建目录并设置权限
sudo mkdir -p /ingot-data/dockerun
sudo chmod 755 /ingot-data/dockerun
```

### Q2: 已运行的容器如何添加卷挂载？

**解决**: 需要重新创建容器

```bash
# 1. 停止并删除容器
docker stop dockerun
docker rm dockerun

# 2. 创建工作目录
sudo mkdir -p /ingot-data/dockerun

# 3. 重新运行（使用 run.sh）
./run.sh start
```

### Q3: 如何修改默认工作目录？

**方法 1**: 通过系统配置页面修改

1. 登录 Dockerun
2. 进入 "System" 页面
3. 修改 "Deploy Work Dir" 配置
4. 确保新目录已创建并挂载

**方法 2**: 通过环境变量

```bash
docker run -d \
    --name dockerun \
    -p 19000:8080 \
    -e DEPLOY_WORK_DIR=/custom/path \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /custom/path:/custom/path \
    --restart unless-stopped \
    docker-registry.ingotcloud.top/ingot/dockerun:0.1.0
```

### Q4: SELinux 环境下的权限问题

如果服务器启用了 SELinux，可能需要添加 `:z` 或 `:Z` 标记：

```bash
docker run -d \
    --name dockerun \
    -p 19000:8080 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /ingot-data/dockerun:/ingot-data/dockerun:z \
    --restart unless-stopped \
    docker-registry.ingotcloud.top/ingot/dockerun:0.1.0
```

- `:z` - 共享卷标记（多个容器可以读写）
- `:Z` - 私有卷标记（仅当前容器可以读写）

### Q5: 如何在 Docker Compose 中配置？

```yaml
version: '3'
services:
  dockerun:
    image: docker-registry.ingotcloud.top/ingot/dockerun:0.1.0
    container_name: dockerun
    ports:
      - "19000:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /ingot-data/dockerun:/ingot-data/dockerun  # 工作目录挂载
    restart: unless-stopped
```

## 验证

运行容器后，验证权限配置：

```bash
# 1. 检查容器内目录
docker exec dockerun ls -la /ingot-data/dockerun

# 2. 在容器内创建测试文件
docker exec dockerun touch /ingot-data/dockerun/test.txt

# 3. 检查宿主机文件
ls -la /ingot-data/dockerun/test.txt

# 4. 清理测试文件
docker exec dockerun rm /ingot-data/dockerun/test.txt
```

## 迁移现有部署

如果已经有运行中的容器，迁移步骤：

```bash
# 1. 导出现有数据（如果需要）
docker exec dockerun tar czf /tmp/backup.tar.gz /app/data
docker cp dockerun:/tmp/backup.tar.gz ./backup.tar.gz

# 2. 停止并删除旧容器
docker stop dockerun
docker rm dockerun

# 3. 创建工作目录
sudo mkdir -p /ingot-data/dockerun

# 4. 恢复数据（如果需要）
# ...

# 5. 使用新配置启动
./run.sh start
```

## 推荐配置

### 开发环境

```bash
# 使用本地目录，方便调试
docker run -d \
    --name dockerun \
    -p 19000:8080 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v $(pwd)/data:/ingot-data/dockerun \
    docker-registry.ingotcloud.top/ingot/dockerun:0.1.0
```

### 生产环境

```bash
# 使用系统目录，数据持久化
docker run -d \
    --name dockerun \
    -p 19000:8080 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /ingot-data/dockerun:/ingot-data/dockerun \
    --restart unless-stopped \
    --log-opt max-size=10m \
    --log-opt max-file=3 \
    docker-registry.ingotcloud.top/ingot/dockerun:0.1.0
```

## 安全建议

1. **最小权限原则** - 只挂载必要的目录
2. **定期备份** - 备份 `/ingot-data/dockerun` 目录
3. **监控日志** - 监控容器日志，及时发现权限问题
4. **网络隔离** - 使用 Docker 网络隔离
5. **定期更新** - 保持镜像和系统更新

## 相关文档

- [Docker Volume 文档](https://docs.docker.com/storage/volumes/)
- [Docker 权限管理](https://docs.docker.com/engine/security/)
- [SELinux 和 Docker](https://docs.docker.com/storage/bind-mounts/#configure-the-selinux-label)

