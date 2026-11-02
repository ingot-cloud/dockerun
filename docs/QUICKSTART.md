# 快速开始

本指南帮助你在受控网络内启动 Dockerun，并完成一次 Compose 应用发布。正式环境的 HTTPS、离线安装和备份方案见[部署指南](DEPLOYMENT.md)。

## 准备环境

准备 Linux Docker Engine（API 1.44+，Docker 25+）和 Docker Compose v2，确认以下命令可用：

```bash
docker version
docker compose version
docker info --format '{{.OSType}}/{{.Architecture}}'
```

目标架构为 `linux/amd64` 或 `linux/arm64`。以下从源码构建镜像，需要联网；宿主机无需安装 Go、Node 或 pnpm。

## 启动并初始化

```bash
git clone https://github.com/ingot-cloud/dockerun.git
cd dockerun
cp .env.example .env
```

编辑 `.env`，例如：

```dotenv
PUBLIC_URL=http://服务器IP:8080
DOCKERUN_IMAGE=dockerun:2.0.0
DATABASE_TYPE=sqlite
```

替换实际 IP；仅在服务器本机访问时可用 `http://localhost:8080`。`PUBLIC_URL` 也是内置仓库认证令牌的入口，必须能从客户端和 Docker daemon 所在环境访问。使用域名和 HTTPS 时，先按[部署指南](DEPLOYMENT.md#https-反向代理)配置代理。

```bash
docker compose config --quiet
docker compose build
docker compose up -d --no-build --pull never
docker compose ps
curl --fail http://127.0.0.1:8080/health
```

默认使用 `dockerun-data` 命名卷，无需提前创建宿主目录；容器内部目录固定为 `/var/lib/dockerun`。已有目录绑定安装请先按[持久化说明](DEPLOYMENT.md#持久化与权限)保留原挂载配置。

浏览器打开配置的入口，首次为 `admin` 设置密码并登录，没有默认密码。进入系统页面确认 Docker 连接正常；HTTP 健康接口成功不代表 Docker 资源及业务应用健康。

## 发布一个 Compose 应用

以下使用公开镜像地址，需要目标执行节点能够拉取镜像。选择未被占用的 `18080` 端口，避免占用 Dockerun 的 `8080`。

1. 在发布工作区创建名为 `demo-nginx` 的应用，添加 `test` 环境，目标选择「本机」。
2. 点击「发布应用」，选择 Compose 部署和按镜像地址拉取，填写以下内容：

   ```yaml
   services:
     web:
       image: nginx:1.28-alpine
       ports:
         - "18080:80"
       restart: unless-stopped
       healthcheck:
         test: ["CMD-SHELL", "wget -q -O /dev/null http://127.0.0.1/"]
         interval: 10s
         timeout: 3s
         retries: 5
   ```

3. 点击「预览发布」，检查镜像、端口与目标，再「确认发布」。保存配置与预览不会启动容器。
4. 在发布任务中等待完成，检查日志、运行状态和健康结果。
5. 打开 `http://服务器IP:18080`，确认 Nginx 页面；如有防火墙，只开放实际需要的访问范围。

按镜像地址拉取会在执行时确定镜像身份；需要发布前锁定摘要时，将镜像准备到内置或外部仓库并选择对应仓库来源。如何推送内置仓库见[仓库指南](REGISTRY_GUIDE.md)。HTTP 仓库需要额外配置 Docker daemon 的 `insecure-registries`，这与浏览器访问网页是两件事。

也可以从「应用模板」创建 MySQL、Redis、Nacos 等服务。模板默认使用内置仓库，需先准备镜像；填写参数后仍需预览并确认发布。业务数据保留在环境对应的数据卷中。

## 后续操作

- 更新应用：编辑部署配置或准备新镜像，重新预览和发布；在任务中确认结果。
- 多环境：新增环境并填写各自参数，可将已有版本部署或晋升到目标环境。
- 离线发布：先导入镜像归档或完整发布包，再创建版本和任务；导入本身不替换服务。
- 备份：从「仓库与维护」创建完整备份并下载，另行备份业务数据卷。
- 停止 Dockerun：执行 `docker compose stop dockerun`；已部署业务容器不会随控制台停止。

完整操作、权限、自动发布及 Agent 见[一体化操作指南](INTEGRATED_DEPLOYMENT.md)。启动、认证或拉取失败时见[故障排查](TROUBLESHOOTING.md)。
