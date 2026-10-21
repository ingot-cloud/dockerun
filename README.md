# Dockerun

集成镜像仓库、应用发布和运行管理的轻量部署工具。一个 Dockerun 容器提供管理网页、内嵌 CNCF Distribution v3.1.2、用户认证和部署执行器，保留外部 Registry 连接。

- 网页与 `docker login` 共用账号；普通用户按无权限、只读、读写授权。
- 选择仓库、本机镜像或完整离线包发布，先准备镜像和预检查，再更新本应用的 Compose project。
- 发布版本锁定 digest / Image ID；任务、配置快照、日志、健康及恢复结果持久化。
- 发布包包含镜像、Compose 和非敏感配置；支持分块续传及服务器目录导入。
- Portainer 风格暗色工作台，提供简单容器、Compose、环境、用户、仓库、任务和主机页面。
- CLI 支持构建发布、打包、导入、晋升、状态和恢复；Agent 主动连接，固定主机组顺序发布。

## 启动

交付容器使用 Docker CLI 29；需要兼容的 Linux Docker Engine（API 1.44+，Docker 25+）与 Compose。支持 `linux/amd64`、`linux/arm64`；每个离线包对应一个平台。

```bash
cp .env.example .env
# 修改 PUBLIC_URL，配置可用的 HTTPS 证书或现有反向代理。
docker compose build
docker compose up -d --no-build --pull never
```

首次访问网页初始化管理员。数据统一持久化在 `/var/lib/dockerun`；该路径与宿主保持一致，以便 Docker 访问 Compose 配置及挂载文件。管理界面、`/api/`、`/v2/` 和 `/registry/token` 使用同一入口。

离线安装先在联网机器准备同平台的 Dockerun 镜像，使用 `docker save` 交付，在目标机器 `docker load` 后运行上述启动命令。运行时不安装依赖、不读取外部 CDN。目标机器需预先安装 Docker Engine。

## 本地开发启动

先启动 Docker Desktop 或本机 Docker Engine。准备 Node 22.17.0、pnpm 10.12.4，以及支持自动工具链下载的 Go（Go 1.21+，`GOTOOLCHAIN` 不要设为 `local`）；项目实际使用 Go 1.26.0。首次启动需要联网下载工具链和依赖。

在项目根目录先执行一次：

```bash
make install-deps
```

然后打开两个终端，均切换到项目根目录：

```bash
# 终端一：后端 API、内置 Registry 和部署执行器，默认 8080
make dev-backend
```

```bash
# 终端二：前端热更新，默认 5173
make dev-frontend
```

访问 [http://localhost:5173](http://localhost:5173)。首次设置 `admin` 密码。API 与 Registry 由前端开发代理转发到后端；`docker login/push/pull` 使用 `localhost:8080`。在对应终端按 `Ctrl+C` 停止进程。

开发模式默认 SQLite，统一使用项目内 `data/local`，与本地体验数据一致，不自动迁移旧的 `backend/data`。`make -C backend dev` 使用相同配置。前端端口被占用时明确报错，不静默切换端口。

Make 命令默认通过 `goproxy.cn` 下载工具链及模块，并通过它的镜像访问官方 `sum.golang.org` 校验数据库；校验保持开启，不改写全局 `go env`。如果之前遇到 `sum.golang.org ... connection reset by peer`，直接重试 `make install-deps`。可以通过环境变量或 Make 参数覆盖，例如：

```bash
# 自定义网络或团队代理
make dev-backend GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org
# 使用其他端口时，同时指定服务地址、Registry 对外地址和前端代理
make dev-backend SERVER_ADDR=127.0.0.1:8081 PUBLIC_URL=http://localhost:8081
make dev-frontend BACKEND_URL=http://127.0.0.1:8081 FRONTEND_PORT=5174
```

如需 MySQL，显式指定 `DATABASE_TYPE=mysql` 和 `MYSQL_DSN`，不要将密码写入 Makefile。开发命令不自动加载容器部署的 `.env`；通过进程环境或 Make 参数提供配置。若只体验构建后的界面，可执行 `make build`、`make run`，直接访问 [http://localhost:8080](http://localhost:8080)，无需另开前端开发服务器。

## 本地交付

```bash
export DOCKERUN_URL=https://dockerun.example.com
# DOCKERUN_TOKEN 使用个人账号页面创建的受限令牌；不要写入仓库。
dockerun deploy gateway --env test --build --wait
dockerun package gateway --file compose.yaml --platform linux/amd64 --output gateway-release.tar.gz
dockerun deploy gateway --env prod --bundle gateway-release.tar.gz --wait
dockerun promote gateway --release 12 --to prod --wait
```

先在网页创建应用和环境。构建默认使用缓存，按目标平台构建；多架构需具有相应能力的 Buildx builder。普通用户的令牌不能部署应用。完整包导入不会自动替换服务。

## 文档与验证

- [一体化操作、权限、迁移及接口](docs/INTEGRATED_DEPLOYMENT.md)
- [分阶段实施与验收](ROADMAP.md)
- [UI 规范](docs/UI_DESIGN.md)
- [可重复验证步骤](docs/VALIDATION.md)

工具链固定 Go 1.26.0、Node 22.17.0、pnpm 10.12.4。开发时使用 `make build`，产生 `bin/dockerun-server` 和 `bin/dockerun`。

Registry 已运行官方 OCI 基础镜像交付 profile（SHA-256 镜像、索引、标签、上传和读取），不能据此声称 OCI 1.1 全量一致。维护删除、Referrers、SHA-512 与外部层的边界见验收文档。普通容器替换可能短暂停机；数据库业务数据回滚、跨机迁移和零停机发布不在此版本范围。

MIT License · Ingot Cloud
