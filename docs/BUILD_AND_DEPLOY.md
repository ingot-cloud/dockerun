# 构建与交付

本文区分本地二进制构建、Dockerun 安装镜像和业务应用发布包。服务安装步骤见[部署指南](DEPLOYMENT.md)，命名卷和目录绑定见[数据存储指南](DATA_STORAGE.md)，业务发布流程见[一体化操作指南](INTEGRATED_DEPLOYMENT.md)。所有命令默认从仓库根目录执行。

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

未自定义构建配置时，`make docker-build`、`build.sh`、Compose 和 `.env.example` 使用相同标签。已有 `.env` 中的 `DOCKERUN_IMAGE` 会覆盖 Compose 默认值，升级时将其改为实际标签。自定义镜像名可使用 `make docker-build IMAGE=仓库地址/团队/dockerun:2.0.0`，同时对齐安装配置。

## 使用 build.sh 交付多平台镜像

`build.sh` 支持独立构建配置文件及环境变量，提供仅离线打包、仅推送、同时打包并推送三种方式。没有配置文件时，默认构建 `linux/amd64,linux/arm64`，镜像标签为 `dockerun:2.0.0`，只导出离线包。

### 配置仓库、名称与版本

```bash
cp build.env.example build.env
```

编辑 `build.env`，例如：

```dotenv
REGISTRY=registry.example.com
IMAGE_NAME=team/dockerun
IMAGE_TAG=2.0.0
PLATFORMS=linux/amd64,linux/arm64
EXPORT=1
PUSH=1
```

上述配置生成完整镜像名 `registry.example.com/team/dockerun:2.0.0`。`REGISTRY` 不包含协议或路径；命名空间放在 `IMAGE_NAME`。仓库地址使用完整域名、IP、`localhost` 或带端口的主机名，例如 `registry:5000`，避免被 Docker 识别成 Docker Hub 命名空间。`IMAGE_TAG` 留空时读取根目录 `VERSION`，不必随版本升级重复修改配置。

配置每行一个 `KEY=value`，可用引号包住整个值，按字面量读取，不执行变量插值、转义或命令替换；注释必须单独占一行。`build.env` 与 `build.*.env` 已从 Git 和镜像构建上下文排除。使用 `BUILD_ENV_FILE=/路径/配置.env` 可指定其他文件，显式指定的文件不存在时会报错；已导出的环境变量优先于文件配置。

| 参数 | 用途与默认值 |
| --- | --- |
| `REGISTRY` | 仓库主机名及可选端口，留空使用 Docker 默认仓库 |
| `IMAGE_NAME` | 镜像名称及可选命名空间，默认 `dockerun` |
| `IMAGE_TAG` | 版本标签，默认读取 `VERSION` |
| `IMAGE` | 完整镜像引用，优先于上述三项，兼容旧命令；未带标签时补上 `IMAGE_TAG` |
| `EXTRA_TAGS` | 附加标签，逗号分隔，例如 `latest,stable`；默认不附加标签 |
| `PLATFORMS` | 目标平台，默认 `linux/amd64,linux/arm64`，也可只选其中一个 |
| `EXPORT` | `1` 导出并加载本地镜像，`0` 不导出；示例文件默认为 `1` |
| `PUSH` | `1` 推送目标仓库，默认 `0` |
| `OUTPUT_PREFIX` | 离线文件名前缀，默认 `docker-packages/镜像短名称-标签`，自动追加架构和 `.tar` |
| `BUILDER` | 可选 Buildx builder，默认使用当前 builder，不自动创建或切换 |
| `BUILD_CONTEXT` | 构建目录，默认仓库根目录 |
| `DOCKERFILE` | Dockerfile 路径，默认构建目录下的 `Dockerfile` |
| `NO_CACHE` | `1` 禁用构建缓存，默认 `0` |
| `PULL` | `1` 刷新基础镜像，默认 `0` |
| `PROGRESS` | 构建输出方式，默认 `auto`，排查构建过程可设为 `plain` |
| `DRY_RUN` | `1` 仅显示命令，也可使用 `./build.sh --dry-run` |

`OUTPUT_PREFIX` 和相对构建目录均以仓库根目录为基准，`DOCKERFILE` 的相对路径以构建目录为基准。从其他目录调用脚本也使用相同路径。没有配置文件且未显式设置 `EXPORT` 时，兼容旧行为：`PUSH=1` 仅推送，`PUSH=0` 仅导出；同时交付请明确设置 `EXPORT=1 PUSH=1`。

### 离线打包

```bash
# 构建两个平台，生成 docker-packages/dockerun-2.0.0-amd64.tar 和 dockerun-2.0.0-arm64.tar
EXPORT=1 PUSH=0 ./build.sh
# 仅交付 AMD64，生成 docker-packages/dockerun-2.0.0-amd64.tar
PLATFORMS=linux/amd64 EXPORT=1 PUSH=0 ./build.sh
# 仅交付 ARM64，并自定义归档前缀
PLATFORMS=linux/arm64 OUTPUT_PREFIX=dist/dockerun-2.0.0 EXPORT=1 PUSH=0 ./build.sh
# 显式禁用缓存
PLATFORMS=linux/amd64 NO_CACHE=1 EXPORT=1 PUSH=0 ./build.sh
```

脚本逐个平台构建、加载并导出归档，归档包含配置的完整镜像引用及附加标签，镜像版本不自动追加架构后缀，仅文件名区分架构。一次构建两个平台时，本机同名标签最终指向最后构建的平台；两个离线包各自保留对应平台镜像，目标机只加载匹配架构的包。归档先写临时文件，导出成功后替换目标文件，失败时保留已有归档。

自定义 `REGISTRY`、`IMAGE_NAME` 或 `IMAGE` 后，归档内不再是默认的 `dockerun:2.0.0`，安装时将 Compose 的 `DOCKERUN_IMAGE` 或 `deploy.env` 的 `IMAGE_NAME` 改为实际完整镜像名。

### 推送镜像

先登录自行维护的仓库，登录凭据沿用 Docker 配置，脚本不保存账号密码：

```bash
docker login registry.example.com
# 仅推送：一次多平台 Buildx 构建，不导出离线包
EXPORT=0 PUSH=1 REGISTRY=registry.example.com IMAGE_NAME=team/dockerun ./build.sh
# 同时打包并推送相同构建产物
EXPORT=1 PUSH=1 REGISTRY=registry.example.com IMAGE_NAME=team/dockerun ./build.sh
# 显式增加 latest 标签
EXPORT=0 PUSH=1 EXTRA_TAGS=latest IMAGE=registry.example.com/team/dockerun:2.0.0 ./build.sh
```

同时打包与推送时，单平台直接推送已导出的本地镜像；多平台逐个构建、导出并推送 `2.0.0-amd64`、`2.0.0-arm64` 等架构标签，最后将其合并到统一的 `2.0.0` 多架构标签及附加标签。每个平台只构建一次，即使 `NO_CACHE=1` 也不会为了推送重新构建，因此离线包和仓库中的对应平台镜像来自同一构建产物。架构标签会保留在本地和仓库，离线包内仍使用不带架构后缀的主标签。

多平台构建需要 builder 支持目标平台；镜像仓库应支持 manifest list。同时交付模式通过宿主 Docker 推送本地镜像，Docker daemon 也须能访问并信任目标仓库。默认复用构建缓存，不会自动给版本增加 `latest`。推送或合并失败会以非零状态退出，已导出的离线包保留；部分架构标签可能已推送，不应把它当作统一版本已完成。避免同时向同一个版本标签执行多个构建任务。

构建与推送参数遵循 [Buildx build](https://docs.docker.com/reference/cli/docker/buildx/build/)，多平台标签合并使用 [Buildx imagetools create](https://docs.docker.com/reference/cli/docker/buildx/imagetools/create/)。

### 预览与验证

```bash
./build.sh --help
./build.sh --dry-run
PUSH=1 EXPORT=1 ./build.sh --dry-run
```

预览会校验配置和源文件路径，显示将执行的命令，不连接 Docker、不创建归档，也不推送镜像。脚本回归测试使用模拟 Docker：

```bash
bash -n build.sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s scripts/tests -p 'test_build.py' -v
```

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
