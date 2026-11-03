# 本地开发指南

本地开发运行 Go 后端和 Vite 前端，容器安装见[部署指南](DEPLOYMENT.md)。以下命令均从仓库根目录执行，除非明确标注其他目录。

## 工具链与依赖

项目使用 Go 1.26.0、Node 22.17.0、pnpm 10.12.4，与 Dockerfile 保持一致。Go 可通过支持自动工具链下载的版本引导，`GOTOOLCHAIN` 不要设为 `local`。首次安装需联网下载工具链及依赖。先启动 Docker Desktop 或本机 Docker Engine；应用发布还需要 Docker CLI、Compose 和 Bash，构建推送需要 Buildx。

```bash
node --version
pnpm --version
go version
docker version
docker compose version
make install-deps
```

Make 使用锁文件安装前端依赖，不要通过删除 `pnpm-lock.yaml` 解决安装问题。可使用已有的 Corepack 安装对应 pnpm：

```bash
corepack enable
corepack prepare pnpm@10.12.4 --activate
```

## 启动开发服务

分别在两个终端进入仓库根目录：

```bash
# 终端一：后端 API、内置 Registry 和发布执行器
make dev-backend
```

```bash
# 终端二：Vite 前端，支持热更新
make dev-frontend
```

访问 [http://localhost:5999](http://localhost:5999)，首次设置 `admin` 密码。后端默认监听 `127.0.0.1:8080`；前端代理 `/api`、`/v2`、`/registry/token` 和 `/health` 到 `http://127.0.0.1:8080`，包括 WebSocket。前端端口固定，冲突时会报错，不会静默改端口。

后端通过 `go run .` 运行，修改 Go 代码后需手动重启，当前 Make 命令没有配置后端自动热重载。在对应终端按 `Ctrl+C` 停止进程。`make -C backend dev` 委托到同一根目录配置。

开发数据默认在仓库 `data/local`，不会自动导入旧的 `backend/data`。开发命令不加载部署用 `.env`，需要通过进程环境或 Make 参数传入配置。不要将开发进程指向业务安装的数据目录。

## 本地 HTTP 仓库调试

Make 默认 `PUBLIC_URL=http://host.docker.internal:8080`，适合 Docker Desktop：daemon 运行在虚拟机中，需要通过此地址访问宿主 Go 服务。浏览器仍访问 `localhost:5999`，前端代理仍连接 `127.0.0.1:8080`。

在 Docker Desktop 的 Settings → Docker Engine 中，把 `host.docker.internal:8080` 合并进现有 `insecure-registries` 列表，保留已有条目及其他设置，再 Apply & Restart。以下仅表示需要加入的配置：

```json
{
  "insecure-registries": ["host.docker.internal:8080"]
}
```

初始化管理员后执行：

```bash
docker login -u admin host.docker.internal:8080
```

`PUBLIC_URL` 决定 `/v2/` 认证挑战中的 `/registry/token` 地址，该地址也必须对 daemon 可达。仅修改 `docker login` 地址或增加 `http://` 不能替代服务配置。独立的 Buildx builder 可能还需要自身的 HTTP / CA 配置。

Linux Docker Engine 与后端同机时，可覆盖入口：

```bash
make dev-backend PUBLIC_URL=http://localhost:8080
docker login -u admin localhost:8080
```

Linux HTTP 仓库需将相同主机和端口合并到 `/etc/docker/daemon.json` 的 `insecure-registries` 并重启 Docker。重启会影响该 daemon 管理的容器，应在开发环境安排操作。远程测试改用 daemon 可达的服务器 IP 或域名，并相应调整后端监听地址。正式环境使用 HTTPS。

## 自定义端口、数据库与下载代理

```bash
# 先在终端一启动后端
make dev-backend SERVER_ADDR=127.0.0.1:8081 PUBLIC_URL=http://host.docker.internal:8081
# 再在终端二启动前端
make dev-frontend BACKEND_URL=http://127.0.0.1:8081 FRONTEND_PORT=6000
```

改端口后同步修改 HTTP 仓库的 daemon 信任地址。直接运行 Vite 时，后端代理配置名为 `VITE_BACKEND_URL`。

Make 默认 `GOPROXY=https://goproxy.cn,direct`，`GOSUMDB=sum.golang.org https://goproxy.cn/sumdb/sum.golang.org`。仅下载传输走镜像，校验仍开启，不改写全局 `go env`。可按当前网络覆盖：

```bash
make install-deps GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org
```

出现校验数据库连接中断时，可先重试安装，不要设置 `GOSUMDB=off`。使用 MySQL 时，先通过当前进程环境提供 `MYSQL_DSN`，再执行 `make dev-backend DATABASE_TYPE=mysql`；不要把真实密码写进 Makefile。详见[配置参考](CONFIGURATION.md)和[MySQL 指南](MYSQL_SETUP.md)。

## 构建与本地预览

```bash
make build
make run
```

`make build` 执行 Vue 类型检查和前端构建，复制静态资源，然后生成：

- `bin/dockerun-server`：提供网页、API、Registry 及本机执行器。
- `bin/dockerun`：发布、打包、导入、晋升、状态、恢复及 Agent CLI。

`make run` 从仓库根目录运行服务，访问 [http://localhost:8080](http://localhost:8080)，不需要另起 Vite。构建和源码变更后需重新运行对应进程。单独使用 `pnpm preview` 仅预览静态资源，不等同于已启动完整后端。

## 检查与验证

```bash
make test-backend
# 在 backend 目录运行静态检查
cd backend
go vet ./...
cd ../frontend
pnpm run type-check
pnpm run build
# 按修改范围运行对应前端测试，例如
pnpm run test:releases
pnpm run test:containers
```

可用前端测试脚本以 `frontend/package.json` 为准，没有通用 `pnpm test` 命令。真实 Docker 与 MySQL 集成测试需要显式开启，并使用隔离资源；复现命令和已有结果见[验证记录](VALIDATION.md)。文档改动通常只需要链接、示例和配置检查，不必执行会替换容器的集成测试。

## 目录与开发约定

| 路径 | 内容 |
| --- | --- |
| `backend/main.go` | 服务启动与退出流程 |
| `backend/cmd/dockerun/` | CLI 入口 |
| `backend/internal/` | 认证、路由、Docker、发布、离线包、Agent、备份等实现 |
| `frontend/src/` | Vue 页面、公共组件、接口及状态管理 |
| `frontend/tests/` | 按功能划分的前端测试 |
| `templates/` | 文件形式的内置应用模板；格式见 [模板说明](../templates/README.md) |
| `data/local/` | 本地开发运行数据，不提交 |
| `docs/` | 当前操作指南、专题和历史设计文档 |

前端变更先遵循仓库 `AGENTS.md` 和 [Dockerun 前端规范](../.agents/skills/dockerun-frontend/SKILL.md)，新增项目说明及用户可见文案使用简体中文。Go 使用 `gofmt`；按改动范围检查和构建。问题排查见[故障排查](TROUBLESHOOTING.md)，镜像及交付方式见[构建与交付](BUILD_AND_DEPLOY.md)。
