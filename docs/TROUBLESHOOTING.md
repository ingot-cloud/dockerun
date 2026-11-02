# 故障排查

先确认实际运行方式：容器安装使用安装目录下的 `docker compose`；源码开发使用 `make dev-backend` 和 `make dev-frontend`。开发命令不读取部署 `.env`。不要通过删除数据目录、数据库或锁文件来尝试修复。

## 先收集运行状态

```bash
docker version
docker compose version
docker compose ps
docker compose logs --tail=200 dockerun
curl -i http://127.0.0.1:8080/health
```

最后一条适用于默认 HTTP 后端或本机代理上游；直接 TLS 安装改用实际 HTTPS 地址。分享错误时隐藏密码、令牌和 DSN。`docker compose config --quiet` 可检查配置而不输出完整环境，避免把含凭据的配置复制到公开日志。

`/health` 只表示服务能响应 HTTP，不验证 Docker 连接、Registry 或业务健康；登录系统页面检查 Docker 状态，发布结果到具体任务核对。

## 服务无法启动或启动后没有旧数据

- 查看日志中的 `PUBLIC_URL` 校验、数据库连接、密钥、证书或文件权限错误。
- `PUBLIC_URL` 必须是完整 HTTP(S) 根地址，不能带路径或凭据；`TLS_CERT`、`TLS_KEY` 必须同时提供。
- 默认数据目录是 `/var/lib/dockerun`，Compose 将该路径原样绑定到宿主；旧的 `/app/data` 或 `backend/data` 不会自动迁移。
- 切换 `DATABASE_TYPE` 或 `MYSQL_DSN` 会连接另一份元数据，不会复制旧数据库。
- 检查宿主 `8080` 是否占用；改端口后同步修改 `PUBLIC_URL`。源码前端默认是 `5999`，不是 `5173`。
- 修改 `.env` 后执行 `docker compose up -d --no-build --pull never`，单纯重启不能加载新环境变量。

恢复旧数据前先停机并保留现有目录，核对完整备份及原密钥配置，按[部署指南](DEPLOYMENT.md#备份与恢复)处理。

## 镜像找不到、离线时仍尝试构建

`DOCKERUN_IMAGE` 必须与实际加载的标签一致。`build.sh`、`make docker-build` 和 Compose 默认使用 `dockerun:2.0.0`。`build.sh` 仅给归档文件名追加架构后缀，例如 `docker-packages/dockerun-2.0.0-amd64.tar`，归档内标签保持 `dockerun:2.0.0`。已有 `.env` 会覆盖 Compose 默认值，升级时也需更新。

```bash
docker image inspect dockerun:2.0.0 --format '{{.Os}}/{{.Architecture}}'
docker compose up -d --no-build --pull never
```

替换为安装实际标签。出现 `exec format error` 时核对目标架构；离线目标先 `docker load`，不运行 `docker compose build`。完整步骤见[构建与交付](BUILD_AND_DEPLOY.md)。

## Docker 连接失败或发布找不到 docker

```bash
# 宿主确认 daemon 可用
docker info
# 容器确认 CLI、Compose 与 socket 连接
docker compose exec dockerun docker version
docker compose exec dockerun docker compose version
```

确认挂载 `/var/run/docker.sock:/var/run/docker.sock`，服务身份具有访问权限。不要把 socket 改成全局可写。原生进程和远程 Agent 应检查自身运行环境的 Docker CLI、Compose、Bash、PATH 和依赖，不能只检查浏览器所在机器。

本机脚本在 Dockerun 进程或容器中运行，Agent 脚本在对应目标执行环境运行；执行器不加载交互式 Shell 配置。挂载或配置找不到时，同时核对宿主与执行器中相同的绝对路径。

## 网页能访问，docker login 却失败

初始化后执行以下检查，HTTP / HTTPS 地址按实际安装修改：

```bash
curl -i https://dockerun.example.com/v2/
```

未认证时预期为带 `WWW-Authenticate` 的 `401`。检查其中 realm 指向真实可达的 `/registry/token`。初始化前 `503` 提示先设置管理员；维护期间写请求也可能返回只读提示。

Docker Desktop 的 daemon 在虚拟机中，本地 Go 服务通常应使用 `PUBLIC_URL=http://host.docker.internal:8080`，并使用 `docker login -u admin host.docker.internal:8080`。宿主 `curl localhost` 成功不说明 daemon 能访问相同入口。HTTP 信任配置见[开发指南](DEVELOPMENT.md#本地-http-仓库调试)。

出现 `server gave HTTP response to HTTPS client` 时，核对实际协议及 daemon 的 `insecure-registries`；出现 `x509` 错误时，核对域名 SAN、证书链及各执行环境的 CA 信任。正式环境使用 HTTPS，不通过跳过校验长期规避。

`docker login` 地址不含协议，非默认端口需保留。网页和内置仓库共用账号，普通用户需要仓库权限，自动化令牌需要 pull / push 范围。外部仓库使用独立账号，其连接测试通过不会自动配置 Docker daemon 的证书或权限。

## 推送、上传被拒绝或日志不实时

通过反向代理访问时，检查整个 `/` 是否统一代理，包括 `/v2/`、`/registry/token`、`/api/`、`/agent/` 和 `/webhooks/`。推送认证失败时核对代理是否保留认证及 Location 响应头；大文件 `413` 或超时时核对网关上传上限与超时。

流式日志需要关闭代理响应缓冲，容器控制台需要 WebSocket 转发。可参考[部署指南的 Nginx 示例](DEPLOYMENT.md#https-反向代理)。独立 Buildx builder 需核对自己的网络与证书配置。

## 直接 TLS 后容器显示不健康

Dockerfile 默认探测 `http://localhost:8080/health`。设置 `TLS_CERT`、`TLS_KEY` 后服务只提供 HTTPS，需同时配置 HTTPS 健康检查，域名应匹配证书，私有 CA 应提供信任。完整示例见[直接启用 TLS](DEPLOYMENT.md#直接启用-tls)。

## 发布失败、健康未知或任务需要核对

先在任务详情查看失败阶段、目标主机、镜像与日志。镜像准备、平台、端口和挂载检查失败时，按对应原因修正后重新预览。导入成功只代表材料准备，不代表已经发布；控制台加载完成也不表示远程节点已加载完成。

没有健康检查时只能确认服务启动，命令退出成功也不能替代容器状态核对。进程中断、脚本后置失败或容器替换失败时，先检查目标实际状态，不盲目重复执行有副作用的脚本。命令部署不自动回滚，业务卷不随版本恢复。

Agent 离线时任务等待，不自动改派；需要的发布能力不匹配时同步升级控制台及 Agent。详细语义见[一体化操作指南](INTEGRATED_DEPLOYMENT.md)。

## 备份失败或凭据无法解密

完整备份从「仓库与维护」创建，查看后台任务错误。核对可用磁盘、目录权限、Registry 维护状态；MySQL 还需核对 `mysqldump`、数据库权限与兼容性。进程中断的备份不会作为完成文件提供下载。

恢复只接受完整格式并要求空目录；MySQL 解包后还需手动恢复 `metadata.sql`。恢复后仓库凭据解密失败，核对原 `keys/credentials.key` 或 `CREDENTIALS_SECRET`，以及历史 `AES_KEY` / `AES_IV`，不要生成新密钥覆盖旧文件。见[凭据说明](REGISTRY_PASSWORD_ENCRYPTION.md)。

## 本地依赖或前端连接问题

```bash
make install-deps
make dev-backend
# 另开终端
make dev-frontend
```

使用项目固定工具链，前端访问 `http://localhost:5999`。修改后端端口时同时提供 `BACKEND_URL` 与 `PUBLIC_URL`，见[开发指南](DEVELOPMENT.md#自定义端口数据库与下载代理)。Go 下载故障可按团队网络覆盖 `GOPROXY` 和 `GOSUMDB`，保留校验，不以删除锁文件或升级全部依赖作为首次排查步骤。
