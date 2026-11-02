# 构建与交付

本文区分本地二进制构建、Dockerun 安装镜像和业务应用发布包。服务安装步骤见[部署指南](DEPLOYMENT.md)，业务发布流程见[一体化操作指南](INTEGRATED_DEPLOYMENT.md)。所有命令默认从仓库根目录执行。

## 本地构建二进制

源码工具链为 Go 1.26.0、Node 22.17.0、pnpm 10.12.4。首次需要下载依赖：

```bash
make install-deps
make build
```

构建产物为 `bin/dockerun-server`、`bin/dockerun` 和前端静态资源。`make run` 启动完整服务，默认使用仓库 `data/local`，访问 `http://localhost:8080`。

独立构建目标平台的 Go 二进制，例如 Linux AMD64：

```bash
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o ../bin/dockerun-server-linux-amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o ../bin/dockerun-linux-amd64 ./cmd/dockerun
```

当前 SQLite 驱动支持纯 Go，Dockerfile 使用 `CGO_ENABLED=0`。服务二进制不等于完整运行环境，网页静态资源、模板、Docker CLI、Compose、Bash 和备份工具仍需按用途准备。容器交付已包含这些材料，因此服务器安装优先使用容器。

## 构建 Dockerun 镜像

当前版本为 `2.0.0`，Git 标签为 `2.0.0`。根目录 `VERSION` 供 Makefile 和 `build.sh` 读取版本；默认镜像标签为 `dockerun:2.0.0`。后续发布时需同步前端包版本、Compose、`.env.example` 和文档中的版本。

先启动 Docker Desktop 或 Docker Engine。根目录 Dockerfile 已执行完整多阶段构建，不需要先在宿主机运行 `make build`：

```bash
make docker-build
# 等价于
docker build -t dockerun:2.0.0 .
# 或使用根目录 Compose，镜像名来自 .env 的 DOCKERUN_IMAGE
docker compose build
```

`Dockerfile.multistage`、`Dockerfile.multiarch` 当前与主 Dockerfile 使用相同的构建流程；通常使用主文件即可。固定基础工具链为 Node 22.17.0、Go 1.26.0 和 Docker CLI 29.1.3，pnpm 固定为 10.12.4。

构建单个目标平台并加载到本机镜像存储：

```bash
docker buildx build --platform linux/amd64 --tag dockerun:2.0.0 --load .
```

AMD64 适用于常见 Intel / AMD 服务器；ARM64 改用 `linux/arm64`，镜像标签仍为 `dockerun:2.0.0`。跨架构构建要求 builder 能运行相应平台的构建步骤。运行前确认目标 Engine 平台和镜像匹配：

```bash
docker image inspect dockerun:2.0.0 --format '{{.Os}}/{{.Architecture}}'
```

`make docker-build`、`build.sh`、Compose 和 `.env.example` 默认使用相同标签。已有 `.env` 中的 `DOCKERUN_IMAGE` 会覆盖 Compose 默认值，升级时将其改为 `dockerun:2.0.0`。自定义镜像名可使用 `make docker-build IMAGE=仓库地址/团队/dockerun:2.0.0`，同时对齐安装配置。

## 使用 build.sh 交付多平台镜像

`build.sh` 通过环境变量配置，不接收位置版本参数。默认目标为 `linux/amd64,linux/arm64`，镜像标签为 `dockerun:2.0.0`。

```bash
# 构建两个平台，生成 docker-packages/dockerun-2.0.0-amd64.tar 和 dockerun-2.0.0-arm64.tar
./build.sh
# 仅交付 AMD64，生成 docker-packages/dockerun-2.0.0-amd64.tar
PLATFORMS=linux/amd64 ./build.sh
# 仅交付 ARM64，并自定义归档前缀
PLATFORMS=linux/arm64 OUTPUT_PREFIX=dist/dockerun-2.0.0 ./build.sh
# 显式禁用缓存
PLATFORMS=linux/amd64 NO_CACHE=1 ./build.sh
```

未设置 `PUSH=1` 时，脚本逐个平台构建、加载并导出归档。每个归档内的镜像标签都是 `dockerun:2.0.0`，仅文件名区分架构。一次构建两个平台时，本机同名标签最终指向最后构建的平台；两个离线包各自保留对应平台镜像，目标机只加载匹配架构的包。自定义 `IMAGE` 可覆盖镜像名称及标签。

推送多平台镜像到自行维护的仓库时，先登录该仓库：

```bash
docker login registry.example.com
PUSH=1 IMAGE=registry.example.com/team/dockerun:2.0.0 ./build.sh
```

`PUSH=1` 使用一次多平台构建并直接推送指定标签，不生成本地离线归档。缓存默认开启。镜像仓库地址和示例标签需替换为实际值。

## Dockerun 离线安装材料

对单平台镜像执行：

```bash
docker save --output dockerun-2.0.0-amd64.tar dockerun:2.0.0
# 在目标机执行
docker load --input dockerun-2.0.0-amd64.tar
```

同时交付根目录 `docker-compose.yml`、`.env.example`，以及实际需要的代理配置和证书。目标 `.env` 对齐加载的标签，然后执行：

```bash
docker compose up -d --no-build --pull never
```

`run.sh` 当前只执行上述启动命令，没有 `start`、`stop`、`logs` 等子命令。停止和查看日志直接使用 `docker compose stop dockerun`、`docker compose logs -f dockerun`。完整离线安装流程见[部署指南](DEPLOYMENT.md#离线安装)。

## 业务应用发布包

业务发布包用于安装完成后的应用交付，不用于启动 Dockerun 本身。先准备全部目标平台镜像和 Compose、`configs/` 中的非敏感文件，再使用本地构建的 CLI：

```bash
./bin/dockerun package gateway --file compose.yaml --platform linux/amd64 --output gateway-release.tar.gz
```

每个完整包对应一个平台；敏感参数保留在目标环境中。在线构建发布使用 `deploy --build`，由本机 Compose / Buildx 构建并推送，服务端受管部署阶段不执行 `build`。

```bash
export DOCKERUN_URL=https://dockerun.example.com
# 通过当前进程环境提供 DOCKERUN_TOKEN，先在网页创建应用及环境
./bin/dockerun deploy gateway --env test --file compose.yaml --build --wait
./bin/dockerun import gateway-release.tar.gz --wait
./bin/dockerun deploy gateway --env prod --bundle gateway-release.tar.gz --wait
```

导入只准备材料；确认发布才会更新服务。Dockerun 的内置仓库、目标 Docker 本地镜像和完整包材料相互独立，打包与导入的详细约束见[一体化操作指南](INTEGRATED_DEPLOYMENT.md)。
