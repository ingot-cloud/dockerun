# 部署指南

本文档详细说明了如何在不同环境中部署DockeRun。

## 目录

- [生产环境部署](#生产环境部署)
- [使用Nginx反向代理](#使用nginx反向代理)
- [系统服务配置](#系统服务配置)
- [离线部署](#离线部署)
- [集群部署](#集群部署)
- [监控和维护](#监控和维护)

## 生产环境部署

### 1. 准备工作

```bash
# 创建项目目录
mkdir -p /opt/dockerun
cd /opt/dockerun

# 下载项目文件
git clone <repository-url> .

# 创建数据目录
mkdir -p data docker-packages
```

### 2. 配置环境变量

创建`.env`文件：

```bash
cat > .env << EOF
# 服务配置
GIN_MODE=release
SERVER_ADDR=:8080

# 安全配置
JWT_SECRET=$(openssl rand -base64 32)

# 数据库配置
DATABASE_PATH=/app/data/dockerun.db
EOF
```

### 3. 使用Docker Compose部署

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 查看服务状态
docker-compose ps
```

### 4. 验证部署

```bash
# 检查容器运行状态
docker ps | grep dockerun

# 测试API
curl http://localhost:8080/api/auth/check-setup

# 查看日志
docker logs dockerun
```

## 使用Nginx反向代理

### 1. 安装Nginx

```bash
# Ubuntu/Debian
apt-get update
apt-get install nginx

# CentOS/RHEL
yum install nginx
```

### 2. 配置Nginx

创建配置文件`/etc/nginx/sites-available/dockerun`：

```nginx
server {
    listen 80;
    server_name dockerun.example.com;

    # 强制重定向到HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name dockerun.example.com;

    # SSL证书配置
    ssl_certificate /etc/nginx/ssl/dockerun.crt;
    ssl_certificate_key /etc/nginx/ssl/dockerun.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # 客户端上传大小限制
    client_max_body_size 100M;

    # 日志配置
    access_log /var/log/nginx/dockerun-access.log;
    error_log /var/log/nginx/dockerun-error.log;

    # 代理配置
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket支持
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
```

### 3. 启用配置

```bash
# 创建符号链接
ln -s /etc/nginx/sites-available/dockerun /etc/nginx/sites-enabled/

# 测试配置
nginx -t

# 重启Nginx
systemctl restart nginx

# 设置开机自启
systemctl enable nginx
```

### 4. 配置SSL证书（Let's Encrypt）

```bash
# 安装certbot
apt-get install certbot python3-certbot-nginx

# 获取证书
certbot --nginx -d dockerun.example.com

# 自动续期
certbot renew --dry-run
```

## 系统服务配置

### 使用Systemd管理服务

创建服务文件`/etc/systemd/system/dockerun.service`：

```ini
[Unit]
Description=DockeRun - Docker Management Platform
After=docker.service
Requires=docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/dockerun
ExecStart=/usr/local/bin/docker-compose up
ExecStop=/usr/local/bin/docker-compose down
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启用服务：

```bash
# 重载systemd配置
systemctl daemon-reload

# 启动服务
systemctl start dockerun

# 设置开机自启
systemctl enable dockerun

# 查看服务状态
systemctl status dockerun

# 查看日志
journalctl -u dockerun -f
```

## 离线部署

### 1. 准备离线安装包

在有网络的环境中：

```bash
# 导出Docker镜像
docker save dockerun:latest -o dockerun-image.tar

# 下载Docker安装包
# x86_64
wget https://download.docker.com/linux/static/stable/x86_64/docker-24.0.7.tgz

# aarch64
wget https://download.docker.com/linux/static/stable/aarch64/docker-24.0.7.tgz

# 下载docker-compose
wget https://github.com/docker/compose/releases/download/v2.23.0/docker-compose-linux-x86_64
chmod +x docker-compose-linux-x86_64

# 注意：如果需要本地构建前端，请确保安装了pnpm
npm install -g pnpm
```

### 2. 传输到离线环境

```bash
# 打包所有文件
tar czf dockerun-offline.tar.gz \
    dockerun-image.tar \
    docker-24.0.7.tgz \
    docker-compose-linux-x86_64 \
    docker-compose.yml \
    .env

# 传输到目标服务器
scp dockerun-offline.tar.gz user@target-server:/opt/
```

### 3. 离线环境安装

```bash
# 解压
cd /opt
tar xzf dockerun-offline.tar.gz

# 安装Docker（如果未安装）
tar xzf docker-24.0.7.tgz
cp docker/* /usr/bin/

# 创建Docker服务（参考系统服务配置）

# 安装docker-compose
mv docker-compose-linux-x86_64 /usr/local/bin/docker-compose

# 加载镜像
docker load -i dockerun-image.tar

# 启动服务
docker-compose up -d
```

## 集群部署

### 使用Docker Swarm

```bash
# 初始化Swarm
docker swarm init

# 部署服务
docker stack deploy -c docker-compose.yml dockerun

# 查看服务
docker service ls

# 扩展服务
docker service scale dockerun_dockerun=3

# 更新服务
docker service update --image dockerun:latest dockerun_dockerun
```

### Docker Stack配置

创建`docker-stack.yml`：

```yaml
version: '3.8'

services:
  dockerun:
    image: dockerun:latest
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dockerun-data:/app/data
    environment:
      - GIN_MODE=release
      - JWT_SECRET=${JWT_SECRET}
    networks:
      - dockerun-net
    deploy:
      replicas: 3
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
      update_config:
        parallelism: 1
        delay: 10s
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M

networks:
  dockerun-net:
    driver: overlay

volumes:
  dockerun-data:
```

## 监控和维护

### 1. 日志管理

```bash
# 查看容器日志
docker logs -f dockerun

# 限制日志大小
docker-compose down
# 修改docker-compose.yml，添加：
# logging:
#   driver: "json-file"
#   options:
#     max-size: "10m"
#     max-file: "3"
docker-compose up -d

# 使用logrotate管理日志
cat > /etc/logrotate.d/dockerun << EOF
/var/log/dockerun/*.log {
    daily
    rotate 7
    compress
    delaycompress
    notifempty
    create 0644 root root
    sharedscripts
    postrotate
        docker kill -s USR1 dockerun
    endscript
}
EOF
```

### 2. 备份策略

```bash
# 创建备份脚本
cat > /opt/dockerun/backup.sh << 'EOF'
#!/bin/bash

BACKUP_DIR="/backup/dockerun"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p ${BACKUP_DIR}

# 备份数据库
cp /opt/dockerun/data/dockerun.db ${BACKUP_DIR}/dockerun-${DATE}.db

# 压缩备份
tar czf ${BACKUP_DIR}/dockerun-backup-${DATE}.tar.gz \
    /opt/dockerun/data \
    /opt/dockerun/.env \
    /opt/dockerun/docker-compose.yml

# 删除30天前的备份
find ${BACKUP_DIR} -name "dockerun-backup-*.tar.gz" -mtime +30 -delete

echo "Backup completed: ${BACKUP_DIR}/dockerun-backup-${DATE}.tar.gz"
EOF

chmod +x /opt/dockerun/backup.sh

# 添加到crontab（每天凌晨2点备份）
echo "0 2 * * * /opt/dockerun/backup.sh" | crontab -
```

### 3. 恢复数据

```bash
# 停止服务
docker-compose down

# 恢复数据
tar xzf /backup/dockerun/dockerun-backup-20231201.tar.gz -C /

# 启动服务
docker-compose up -d
```

### 4. 性能监控

```bash
# 查看容器资源使用
docker stats dockerun

# 使用Prometheus监控（可选）
# 添加监控配置到docker-compose.yml
```

### 5. 更新升级

```bash
# 拉取最新代码
cd /opt/dockerun
git pull

# 备份当前数据
./backup.sh

# 重新构建镜像
docker-compose build

# 重启服务
docker-compose down
docker-compose up -d

# 验证更新
docker logs dockerun
```

## 安全加固

### 1. 防火墙配置

```bash
# Ubuntu/Debian (ufw)
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable

# CentOS/RHEL (firewalld)
firewall-cmd --permanent --add-service=ssh
firewall-cmd --permanent --add-service=http
firewall-cmd --permanent --add-service=https
firewall-cmd --reload
```

### 2. 限制访问

```nginx
# 在Nginx配置中添加IP白名单
location / {
    allow 192.168.1.0/24;
    deny all;
    
    proxy_pass http://127.0.0.1:8080;
    # ... 其他配置
}
```

### 3. 定期安全检查

```bash
# 检查容器安全
docker scan dockerun:latest

# 更新系统
apt-get update && apt-get upgrade -y

# 检查服务状态
systemctl status dockerun nginx
```

## 故障排查

### 常见问题

1. **服务无法启动**
   - 检查端口占用：`netstat -tlnp | grep 8080`
   - 检查Docker服务：`systemctl status docker`
   - 查看错误日志：`docker logs dockerun`

2. **无法连接Docker**
   - 确认Docker socket挂载正确
   - 检查容器权限配置

3. **前端无法访问**
   - 检查Nginx配置
   - 检查防火墙规则
   - 验证SSL证书

## 性能优化

### 1. 数据库优化

```bash
# 定期优化SQLite数据库
sqlite3 /opt/dockerun/data/dockerun.db "VACUUM;"
```

### 2. Nginx缓存配置

```nginx
# 添加静态资源缓存
location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
    expires 1y;
    add_header Cache-Control "public, immutable";
}
```

### 3. 容器资源限制

在`docker-compose.yml`中添加：

```yaml
services:
  dockerun:
    # ... 其他配置
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
        reservations:
          cpus: '1'
          memory: 512M
```

## 支持

如有部署问题，请参考[README.md](README.md)或提交Issue。

