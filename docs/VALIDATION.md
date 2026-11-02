# 验证记录与复现

日期：2026-10-21。测试使用独立数据目录、端口、Docker project 和数据库，不修改本地 Portainer 或业务容器。

## 开发检查

```bash
cd backend
go test ./...
go vet ./...
go test -race ./internal/router ./internal/release ./internal/bundle ./internal/backup
cd ../frontend
pnpm run type-check
pnpm run build
cd ..
docker build -t dockerun:validation .
```

已执行后端测试、核心 race 检查、Vue 类型检查及生产构建、单镜像构建。Go 源码同时交叉编译 Linux amd64；arm64 镜像及部署已实际运行。编辑器从启动入口移到按需加载，所有静态资源在构建产物内。

## 真实 Docker

预先准备同平台 `nginx:1.28-alpine`，只运行明确开启的测试：

```bash
cd backend
DOCKERUN_DOCKER_TESTS=1 go test ./internal/release -run TestDockerLocalAndOfflineDeployment -count=1 -v
```

测试创建 `dockerun-a987654-*` 专属 project，无宿主端口和业务数据卷，并在结束时清理。覆盖镜像身份、平台、预检查、Compose 更新、健康、缺少材料保留旧服务、打包、哈希校验、镜像加载和严格本机 ID 部署。

另外已通过专用单容器验证实例实际执行：

- 原生 Docker login、单平台 push / pull。
- Buildx docker-container builder 推送 `linux/amd64,linux/arm64` OCI index；builder 单独命名，未切换用户默认 builder。
- 从内置仓库解析 digest、准备镜像、创建持久化任务、更新 Compose 并确认健康。
- 故意让新版本健康失败，确认任务失败、恢复成功、当前版本仍为原成功版本。
- 本机 Agent 主动连接隔离控制台；CLI 创建任务并等待远端健康结果及实际 Image ID。
- CLI 生成 25 MiB 完整发布包、按 8 MiB 分块上传、服务端校验、Agent 下载并加载后用本地 ID 部署。未变化服务保持原容器。

以上测试运行在有 Docker daemon 的开发环境，不等同于“物理断外网、双架构真实 Linux 主机”的完整交付验收。该项保留为正式发布门槛。

## MySQL

使用专用空测试数据库，不对业务库运行以下测试：

```bash
DOCKERUN_MYSQL_TESTS=1 \
MYSQL_DSN='test_user:test_password@tcp(127.0.0.1:3306)/dockerun_test?charset=utf8mb4&parseTime=True&loc=Local' \
go test ./internal/database -run TestMySQL -count=1 -v
```

MySQL 8.4 重复迁移、管理员初值及中文 JSON 参数读写已通过。SQLite 完整备份恢复测试覆盖数据、Registry 材料、离线材料、密钥及 AES-GCM 凭据解密，并拒绝覆盖已有目录。MySQL 完整备份恢复的专用环境演练仍保留。

## 官方 OCI profile

测试来源：`opencontainers/distribution-spec`，commit `97274622c11112caa21efb8c52acca3c6b8fa7f1`。先构建该仓库 `conformance/` 下的官方工具，再针对**专用测试 Registry**运行，不能对业务仓库执行。

```bash
export OCI_REGISTRY=registry-test.example.com
export OCI_TLS=enabled
export OCI_USERNAME=test_admin
# 密码通过当前进程环境提供，不写入工程或报告。
export OCI_REPO1=conformance/repo1 OCI_REPO2=conformance/repo2
export OCI_VERSION=1.0
export OCI_API_BLOBS_DELETE=false OCI_API_TAGS_DELETE=false OCI_API_MANIFESTS_DELETE=false
export OCI_API_REFERRER=false
export OCI_DATA_SHA512=false OCI_DATA_SUBJECT=false OCI_DATA_SUBJECT_MISSING=false
export OCI_DATA_SUBJECT_LIST=false OCI_DATA_NONDISTRIBUTABLE=false
export OCI_RESULTS_DIR=./results
./conformance
```

该 profile 覆盖当前镜像交付范围：309 Pass、0 Fail、104 Disabled、2 Skip。包括分块上传、挂载、流式读写、HEAD、Range、manifest 按 digest / tag、标签列表、普通镜像、artifact、空层、索引与嵌套索引。删除由 Dockerun 的保护策略和单独权限测试验证，因此没有作为无条件 DELETE 能力宣称。

也运行了 OCI 1.1 默认全量配置（删除关闭），结果为 421 Pass、91 Fail、163 Disabled、16 Skip。失败涉及 Referrers / OCI-Subject、SHA-512 digest 行为和外部层等能力。**不能把基础 profile 通过表述为 OCI 1.1 全量一致**；这些全量结果保留为兼容范围及后续评估依据，未修改成熟引擎以临时冒充完整协议支持。

## 权限与故障逻辑

自动测试覆盖标准 GET / OAuth POST 令牌交换、交集权限、三档角色、网页 / Registry 令牌不可混用、旧令牌撤销、跨仓库范围、普通用户直接调用部署 / 导入 / 资源接口、最后管理员、原始受保护删除、未知路径和归档类型、缺少参数、Docker JSON 错误帧、事件去重、允许推送人撤权、主机组顺序与失败暂停。

正式发布前仍需专门环境回归：完整浏览器尺寸和角色矩阵、TLS / 私有 CA、大文件长时间中断与并发、各执行阶段强制重启、磁盘不足注入、多台 Agent 重连及完全离线安装更新。不得通过填满业务宿主磁盘或中断业务 Docker daemon 来完成故障测试。

## 本地镜像页优化（2026-10-21）

- 前端 `pnpm test:images`：10 项测试通过，覆盖 ID / 标签 / 摘要搜索、排序、文件校验、归档命名、Docker JSON Lines / SSE 分块与失败判定、强制删除参数和导出请求。
- Vue 类型检查与生产构建通过。构建仍有已有的大体积资源块提示。
- 后端 `go test ./internal/handler ./internal/docker ./internal/router` 与编译通过。新增导出测试使用内存 HTTP Transport 模拟 Docker，验证多选合并、标签保留、未标记镜像、去重、非法请求与失败响应，不修改本机镜像。
- 浏览器验证 1280×720、1440×900、1920×1080、1000×720：页面不产生整体纵向滚动，操作区、固定表头与分页可见，列表主体独立滚动。
- 浏览器验证当前页选择 25 项、换页追加为 26 项，搜索后清除选择；完整 ID 搜索、无匹配状态、大小排序、普通及强制删除确认、取消、本页导入弹窗均正常。没有执行实际删除、拉取或导入。
- 已实际下载单镜像 tar 并检查 manifest，确认旧后端按 ID 导出会丢失标签；修复通过模拟 Docker 请求测试。批量导出及标签保留修复需重启后端后再做实际联调；用户选择自行重启，当前开发后端未由代理重启。
- Chrome 扩展未允许本地文件 URL，自动文件选择未能完成；文件类型、空文件及导入流失败处理已通过单元测试，实际上传导入仍待人工验证。

## 镜像使用状态与详情（2026-10-21）

- 前端镜像测试扩展到 13 项：增加按 ID 判断运行及停止容器引用、未知状态不误标、环境变量含等号、构建指令格式化、返回列表的查询状态及详情 / 历史 / 标签接口请求测试。类型检查与生产构建通过。
- 后端历史接口使用请求上下文，测试覆盖正常历史、空历史、镜像不存在、Docker 异常，以及普通用户禁止读取历史。`go test ./internal/handler ./internal/docker ./internal/router` 和编译通过。
- 浏览器实际核对列表未使用标记及在用镜像不显示该标记；进入在用镜像和未标记镜像详情，核对标签、完整 ID、大小、创建时间、平台、Labels、Dockerfile 和使用状态；验证无效标签输入及删除确认后取消，没有修改标签或删除镜像。
- 浏览器验证返回列表仍保持第 2 页及原首项，搜索 nginx 后进入详情再返回仍为过滤结果；详情在 1280×720、1440×900、1920×1080 和 1000×720 下没有整体或横向溢出。
- 后端重启前验证了历史接口未生效时的独立错误状态；重启后浏览器已显示 Python 镜像的 9 条真实构建历史。序号列在桌面端与窄屏下分别加宽至 96px 和 80px，表头禁止换行，浏览器确认序号文字与排序图标保持同一行。用户确认测试通过。


## 命名卷与目录绑定（2026-11-02）

- 后端全量测试、`go vet ./...` 及 `go test -race ./internal/docker ./internal/release` 通过。新增启动校验覆盖无映射原生进程、旧版同路径绑定、映射配置冲突、命名卷缺失及无挂载点；路径测试覆盖边界、符号链接、JSON 整数精度、流式响应和 Docker exec 双向连接。
- 部署脚本 28 项模拟测试通过，覆盖默认命名卷、创建与复用卷、旧配置和原挂载识别、显式配置优先级、固定容器路径及失败时保留原容器。构建脚本原有 14 项测试通过。Compose 配置实际合并后，两种模式都只有一个 `/var/lib/dockerun` 数据挂载，目录绑定覆盖文件会清空 `DATA_VOLUME` 并设置 `HOST_DATA_DIR`。
- 在 Docker Desktop Linux ARM64 上，使用已有镜像内的 Docker CLI/Compose 和交叉编译的最新测试程序，分别挂载独立命名卷及临时宿主目录，运行 `TestDockerStoragePathDeployment`。验证相对 `configs.file`、旧宿主绝对绑定源、本地卷驱动 bind、脚本 `docker run -v`/`--mount` 及 `docker exec` 均读取到正确内容；重复 Compose up 保持容器 ID。测试资源已清理，没有构建或推送镜像。
- 临时路径代理 socket 存放在容器 `/tmp`，避免 Docker Desktop 共享目录不支持 socket chmod；持久化目录不保存代理 socket。真实双架构 Linux 主机及已有业务数据的生产迁移仍需按停机备份流程核对。

存储集成测试默认跳过。复现时先将测试二进制交叉编译为 Docker daemon 的 Linux 架构，并在专用辅助容器内执行，不能直接将生产目录作为测试数据：

```bash
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go test -c ./internal/release -o /tmp/dockerun-storage.test
# 数据卷使用本次验证专用且未使用的名称，镜像需已存在本地。
docker volume create dockerun-storage-validation
docker run --rm --pull never --network none --entrypoint /storage-test \
  --mount type=bind,src=/tmp/dockerun-storage.test,dst=/storage-test,readonly \
  --mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
  --mount type=volume,src=dockerun-storage-validation,dst=/var/lib/dockerun \
  --env DATA_DIR=/var/lib/dockerun --env DATA_VOLUME=dockerun-storage-validation \
  --env DOCKERUN_STORAGE_TESTS=1 \
  dockerun:2.0.0 -test.run '^TestDockerStoragePathDeployment$' -test.v -test.timeout 150s
docker volume rm dockerun-storage-validation
```

预先准备同架构的 `nginx:1.28-alpine`，测试不会主动拉取镜像。AMD64 将编译架构改为 `amd64`。目录绑定验证将数据卷挂载替换为测试专用宿主空目录，去掉 `DATA_VOLUME`，设置 `HOST_DATA_DIR` 为该目录的真实绝对路径；容器内 `DATA_DIR` 不变。业务测试项目使用 `dockerun-a987655-*` 名称，不发布宿主端口，结束时移除项目容器、网络和测试卷。
