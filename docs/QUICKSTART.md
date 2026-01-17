# 快速开始指南

本指南将帮助你在5分钟内快速启动DockeRun。

## 前置要求

- Docker已安装并运行
- Docker Compose已安装（v2.0+）

### 本地开发环境
- Node.js 20+
- pnpm 8+
- Go 1.21+

## 快速部署

### 1. 克隆项目

```bash
git clone <repository-url>
cd dockerun
```

### 2. 配置环境变量（可选）

```bash
# 复制环境变量示例文件
cp .env.example .env

# 编辑.env文件，修改JWT密钥（推荐）
# JWT_SECRET=your-random-secret-key
```

### 3. 启动服务

```bash
# 使用Docker Compose启动
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 4. 访问应用

打开浏览器访问: `http://localhost:8080`

### 5. 初始设置

首次访问会自动跳转到初始设置页面：

1. 设置管理员密码（至少6位）
2. 点击"设置密码"
3. 使用 `admin` 和你设置的密码登录

## 常用命令

```bash
# 启动服务
docker-compose up -d

# 停止服务
docker-compose down

# 重启服务
docker-compose restart

# 查看日志
docker-compose logs -f

# 查看服务状态
docker-compose ps

# 进入容器
docker exec -it dockerun sh

# 备份数据
./scripts/backup.sh

# 更新服务
./scripts/update.sh
```

## 功能概览

### 1. 容器管理
- 查看所有容器
- 启动/停止/重启容器
- 查看容器日志
- 删除容器

### 2. 镜像管理
- 查看镜像列表
- 拉取新镜像
- 删除镜像
- 镜像打标签

### 3. 网络管理
- 查看网络列表
- 创建/删除网络
- 连接/断开容器

### 4. 卷管理
- 查看卷列表
- 创建/删除卷

### 5. 部署脚本
- 创建部署脚本
- 执行脚本部署容器
- 查看执行输出

### 6. 系统管理
- 查看系统信息
- 检测Docker状态
- Docker安装（支持在线/离线）

## 部署脚本示例

创建一个简单的Nginx部署脚本：

```bash
#!/usr/bin/env bash

SERVICE_NAME=my-nginx
IMAGE_NAME=nginx:latest

# 停止旧容器
docker ps -q --filter name="${SERVICE_NAME}" | xargs -r docker rm -f

# 启动新容器
docker run -d \
  --name ${SERVICE_NAME} \
  --restart always \
  -p 8080:80 \
  ${IMAGE_NAME}

# 输出容器ID
docker ps -q --filter name="${SERVICE_NAME}"
```

保存并执行此脚本即可部署Nginx容器。

## 故障排查

### 服务无法启动

```bash
# 检查Docker是否运行
systemctl status docker

# 检查端口占用
netstat -tlnp | grep 8080

# 查看详细日志
docker logs dockerun
```

### 无法连接到Docker

确保Docker socket已正确挂载：

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock
```

### 前端无法访问

1. 检查防火墙设置
2. 确认容器正常运行
3. 查看浏览器控制台错误

## 安全建议

1. **立即修改默认密码**
2. **更改JWT密钥**（生产环境必须）
3. **使用HTTPS**（配置Nginx反向代理）
4. **限制访问IP**（通过防火墙或Nginx）
5. **定期备份数据**

## 下一步

- 阅读完整[README.md](README.md)了解详细功能
- 查看[DEPLOYMENT.md](DEPLOYMENT.md)了解生产部署
- 配置Nginx反向代理启用HTTPS
- 设置定时备份任务

## 获取帮助

- 查看[常见问题](README.md#常见问题)
- 提交[Issue](https://github.com/your-repo/issues)
- 阅读[部署指南](DEPLOYMENT.md)

## 更新日志

查看[CHANGELOG.md](CHANGELOG.md)了解最新更新。

---

**享受使用DockeRun！** 🚀

