# Dockerun 一体化部署操作与接口

## 安装与持久化

统一镜像内含 Go 服务、网页、Distribution v3.1.2、Docker CLI、Compose 和数据库备份工具。运行时不启动额外 Registry 容器。入口为 `/`、`/api/`、`/v2/`、`/registry/token`（GET 与 BuildKit OAuth POST）。

设置 `PUBLIC_URL` 为客户端及 Docker daemon 可达的根地址；不能带路径、账号或查询参数。默认禁止匿名访问，初始化管理员前仓库不可用。直接 HTTPS 使用 `TLS_CERT`、`TLS_KEY`；已有反向代理需保留 Authorization、WWW-Authenticate、Location 等协议头，并允许大文件及流式响应。HTTP 本机验证与正式 HTTPS 配置应分别管理；Docker daemon 对 HTTP Registry 的要求沿用 Docker 标准配置。

Docker Desktop 本地开发时，宿主机 Go 服务与虚拟机中的 daemon 不在同一网络环境。本地 Make 命令默认设置 `PUBLIC_URL=http://host.docker.internal:8080`，执行 `make dev-backend` 即可使用该认证入口，无需额外传参。若 `docker login localhost:8080` 超时，但宿主机执行 `curl http://localhost:8080/v2/` 正常返回带 `WWW-Authenticate` 的 401，应从 daemon 环境检查宿主访问地址。使用 `host.docker.internal:8080` 登录时，认证令牌地址也必须使用此入口；已有后端进程需重启以应用配置。仅本地 HTTP 调试时，把该地址合并到 Docker Desktop 的 `insecure-registries` 并 Apply & Restart；已有配置应保留。Linux 本机 Docker Engine 可用 `make dev-backend PUBLIC_URL=http://localhost:8080` 覆盖，正式容器部署仍通过 `.env` 指定实际入口。步骤见 README 的本地开发部分。

`DATA_DIR` 默认 `./data`，容器中为 `/var/lib/dockerun`：

| 目录 | 内容 |
|---|---|
| `keys/` | 网页、Registry、上传 HMAC 及凭据加密密钥 |
| `registry/` | Distribution 管理的镜像内容与上传状态 |
| `dockerun.db` | 默认 SQLite 元数据；可切换专用 MySQL 数据库 |
| `uploads/` | 上传分块及已导入归档 |
| `releases/` | 离线解包材料 |
| `jobs/` | 渲染配置、任务材料与恢复定义 |
| `backups/` | 完整备份及升级前数据库快照 |
| `imports/` | 管理员可指定导入的服务器文件目录 |

目录需持久化；使用绑定卷时保持 Dockerun 和宿主路径一致。不要只备份数据库后将其当作完整镜像备份。`REGISTRY_ENABLED=false` 可以关闭内置引擎，不删除内容。原有外部 Registry 及默认选项保留。

## 用户与凭据

管理员可创建、启停、重设密码、调整角色及仓库权限。普通用户默认无仓库权限，仅能访问个人账号。只读可以浏览和拉取；读写增加推送。普通用户不能导入 Docker 镜像、导入发布包、部署应用、管理外部连接或宿主资源。最后一个启用管理员不能被停用或降级。

网页旧 AES 密码传输入口保留兼容，存储仍为 bcrypt。Docker 使用标准账号密码交换令牌，不使用网页加密格式。网页和 Registry JWT 的密钥、用途及 audience 独立；Registry 默认 300 秒，无刷新令牌。禁用、改密和授权更新递增认证版本，后续请求立即重新检查。已开始的数据流不强制中断。

新外部仓库凭据使用持久化密钥进行 AES-GCM 加密，兼容读取历史 AES-CBC 数据。若历史安装自定义过 `AES_KEY` / `AES_IV`，升级和恢复时保留相同配置，重新保存连接后会写入新格式。显式通过环境变量提供的签名或加密密钥也应纳入安装配置备份。

自动化令牌在个人页面创建，最长 90 天，默认 30 天，只显示一次。可选择 pull、push、deploy，不能超出当前账号授权；普通用户不能取得 deploy。自动化令牌不能管理账号、签发子令牌或管理宿主和系统。可通过 `docker login -u 用户名 --password-stdin` 将 Registry 令牌作为密码使用；页面命令不嵌入凭据。

外部 Registry 使用独立账号。Agent 默认不会接收管理连接凭据；管理员须另外配置在外部 Registry 中被限制为只读的拉取账号。外部账号的实际 ACL 由外部 Registry 执行；Dockerun 无法代替外部系统授予或撤销该 ACL。

## 在线发布

### 从应用模板开始

「发布工作区 → 应用模板」统一提供单服务应用、MySQL、Redis、Nacos 单机及 Nacos 单机三节点集群。选择模板，填写应用名称、首个环境、目标主机、镜像仓库和运行参数，点击「创建并准备发布」进入已填好的发布向导，检查后确认部署。创建与预览不会启动容器。

默认使用内置仓库，镜像必须先准备到所选仓库；也可以切换已有外部仓库，表单会保留镜像名称和版本、切换仓库地址。镜像版本沿用项目旧模板的固定版本，发布时继续锁定摘要，不自动跟随新标签。中间件默认手动更新；需要持续交付的业务应用，在首次成功后按环境开启镜像自动发布。

模板包含参数表单、Compose、非敏感配置文件、部署脚本、初始化说明和访问入口。应用记录创建来源及模板版本，并保存独立定义；修改或新增模板版本不会修改已有应用。首次网页发布成功准备版本后会保存尚未插值的应用定义，后续发布可沿用该定义，参数值仍属于对应环境。配置准备成功不等于服务已经部署成功。

MySQL、Redis 以及 Nacos 各节点采用环境独立的命名卷，卷名随应用及环境 ID 固定，重新部署和手动升级复用原数据。中间件不能通过无状态自动恢复回滚数据。应用详情展示持久化说明、首次初始化规则以及依据环境参数生成的连接地址；远程节点须替换为实际可达的主机地址。

Nacos 三节点模板在一台主机部署三个 Nacos 节点及独立 MySQL，MySQL 不映射宿主端口。集群建表文件来自 [Nacos 3.1.1 官方 SQL](https://github.com/alibaba/nacos/blob/3.1.1/distribution/conf/mysql-schema.sql)，保留原许可，并随程序打包；由 MySQL 在空卷首次启动时执行，不在每次发布时重复建表。更改初始密码不会修改已有数据库的实际密码。Nacos 健康检查验证服务端口，集群成员状态需在控制台确认。此模板不提供跨主机高可用，页面及接口拒绝用固定主机组重复部署来代替集群编排。

「另存为应用模板」保存当前应用的未插值定义、非敏感文件和应用脚本，以新名称与版本存储，不覆盖已有模板。环境参数值、主机及业务数据不会复制；Compose 中可识别的固定凭据和绝对数据路径转换成待填写参数，固定容器名移除，普通数据卷重新按环境隔离。脚本及非敏感配置文件中的固定凭据需由管理员确认并移入环境参数。自定义模板随后出现在同一列表中。

「系统管理 → 历史脚本」继续兼容原有独立脚本与其旧模板，只支持手动执行；新服务从应用模板创建。本轮迁移上述常用中间件，其他旧脚本不会被自动执行、改写或重建已有容器。跨主机集群编排及自动数据库迁移需另行实现。

1. 创建应用，添加 test / prod 等环境，选择本机、Agent 或固定主机组。
2. 保存环境参数及健康、无状态和恢复策略。生产默认手动发布。
3. 先将镜像推送到仓库。发布来源默认选择内置仓库，也可以明确切换到外部仓库；提供包含所选仓库域名的完整镜像引用。版本创建时解析并保存 digest。
4. 简单容器表单或 Compose 定义描述服务。构建在开发机进行，部署阶段禁止 build；配置放入 `configs/`，敏感环境参数在目标环境配置。
5. 预览定义、镜像、目标及当前 Compose，确认后创建任务。
6. 执行器准备全部镜像、生成锁定镜像的配置，执行部署前脚本，检查平台/端口/挂载/网络，再启动本应用 project，核对实际 Image ID 和健康，最后执行部署后脚本。

### 应用和环境的部署脚本

在应用详情的「应用部署脚本」中配置部署前、部署、部署后三段 Bash 脚本。应用脚本在创建新版本时保存快照，之后编辑不会改变已有版本；手动发布、镜像自动发布、从完整包创建的新版本都采用此规则。

环境配置中的「环境部署脚本」可选择沿用版本中的应用脚本、使用此环境的独立脚本、停用脚本。环境独立脚本在提交任务时保存快照，固定主机组中的每台主机使用同一配置。脚本在实际目标执行器中运行：本机任务在 Dockerun 运行环境中执行，远程任务在对应 Agent 主机上执行；需要的 Bash、命令和目录必须在该执行环境可用。控制台及 Agent 均需更新到支持脚本的版本。

部署脚本留空时系统运行 Compose；填写后由脚本接管启动步骤，系统仍要求应用栈中的服务运行且镜像身份一致。部署前脚本可创建挂载目录或准备外部资源，部署后脚本可追加检查；任一脚本退出码非零会使任务失败，失败命令或管道不会继续执行后续脚本。预检查页面只展示脚本，不执行。

脚本工作目录为当前任务目录，`configs/` 中包含该版本的非敏感配置。环境参数作为环境变量传入，另提供 `WORK_DIR`、`COMPOSE_FILE`、`COMPOSE_PROJECT_NAME`、`DOCKERUN_APPLICATION_ID`、`DOCKERUN_ENVIRONMENT_ID`、`DOCKERUN_ENVIRONMENT`、`DOCKERUN_VERSION`。系统变量以实际任务为准。自定义部署脚本的最小示例：

```bash
docker compose --env-file "$WORK_DIR/empty.env" \
  --project-name "$COMPOSE_PROJECT_NAME" --project-directory "$WORK_DIR" \
  -f "$COMPOSE_FILE" up -d --pull never --no-build --wait --wait-timeout 120
```

所有阶段共享任务超时，日志按阶段持久化。脚本执行被进程重启打断时标记为「需要核对」，不会自动重跑。持久数据应放在环境指定的挂载路径；任务目录属于单次发布。自动恢复只恢复上一成功版本的服务和配置，不重跑其脚本，也不撤销脚本修改的数据库或外部系统。恢复历史成功配置时同时恢复当时的环境参数和有效脚本。历史脚本入口继续保留；已有脚本可人工迁移到应用或环境配置，附加配置文件放入发布版本的 `configs/`。

任务与浏览器连接无关。历史与 SSE 可重新读取。进程重启后排队任务继续；已开始任务先对账，无法确认时标为“需要核对”，不盲目重复替换。应用的部署结果、运行状态和健康分别存储；没有健康检查时显示“已启动，未验证健康”。

自动恢复仅限明确无状态且开启策略的环境。恢复使用上次成功的配置和镜像身份，生成的失败记录保留，恢复结果单独展示。应用数据卷不由该功能回滚或清空。默认保护各环境、目标最近 3 个成功版本、当前版本以及排队/执行中版本。

## 自动发布授权

先在对应环境使用仓库镜像完成一次手动发布，确认镜像、端口、挂载、配置文件和脚本。随后在环境配置中开启「允许镜像提交后自动发布」，明确绑定仓库来源、镜像名称（例如 `team/gateway`，不含域名和标签）及标签规则（如 `latest`、`test-*`、`*`）。尚未成功发布、来源仓库不一致或当前版本没有匹配镜像时，页面及接口会明确提示，不能直接开启。

内置仓库为新发布配置的默认来源，直接在完整 manifest / 最终命名索引提交后记入持久化事件，**无需配置 Webhook、回调地址或密钥**。开启后，授权推送人的匹配镜像事件生成新版本，沿用此环境当前成功版本的服务定义和其他镜像、使用当前应用脚本的新快照及环境脚本规则，随后自动提交部署任务。其他环境的草稿或最新版本不会替换此环境的发布基础。镜像层、未完成提交、按 digest 推送的 Buildx 子 manifest 不直接触发部署。

管理员推送默认可触发已启用的环境；普通用户须在该环境的允许推送人列表中。创建及执行任务都检查当前资格。普通推送也可形成待发布版本。包导入和显式包镜像入库不触发自动替换。

外部 Webhook 要求启用、配置密钥，并由管理员保存来源规则。请求必须验签，镜像 digest 须从已配置 Registry 实际确认，才能持久化处理。载荷 actor 不作为 Dockerun 用户。旧来源和目标没有明确绑定的自动脚本在升级时关闭，保留定义供人工迁移。

## 离线发布包

```text
application-release.tar.gz
├── release.json
├── compose.yaml
├── images/*.tar
├── configs/*
├── env.example
└── checksums.sha256
```

`release.json` 的 `schema_version=1`，记录应用、版本、目标平台、每个镜像完整 Image ID、归档路径及 SHA-256。一个包包含一个应用栈所有服务镜像，支持 Linux amd64 或 arm64。外部依赖明确说明；生产密码、数据卷和可执行安装脚本不放入包中。

```bash
# 已在开发机准备好所选平台镜像；默认不联网补齐。
dockerun package gateway --file compose.yaml --platform linux/amd64 --output gateway-release.tar.gz
# --defaults defaults.json 仅提供非敏感打包插值默认值。
dockerun import gateway-release.tar.gz --wait
dockerun deploy gateway --env prod --bundle gateway-release.tar.gz --wait
```

网页支持完整包和普通 `docker save` 归档两种材料，分块最大 16 MiB，客户端使用 8 MiB。文件 SHA-256 标识上传，偏移校验与 fsync 支持续传。服务器目录导入只读取 `IMPORT_DIR` 内文件。未知格式、错误平台、缺少镜像、校验失败、重复路径、越界路径、链接和特殊文件都会拒绝。

控制台和包平台一致时，校验通过后加载本机；平台不一致时仅保存完整材料，明确显示“由目标节点确认”，不将控制台状态视为其他节点已就绪。节点加载时解析 Docker 的 JSON 错误帧，再检查实际加载的 Image ID 和平台。导入只准备材料；创建待发布版本时不修改已有环境参数。严格离线运行只引用已有 Image ID，禁用拉取、构建、远程 include、extends、env_file 与不明确的 profiles。

仓库、Docker 本机缓存和离线材料是三类独立存储。本机导入不会自动入库；管理员可通过 `/api/bundles/publish` 显式将已导入包版本镜像推送到内置仓库，生成在线版本用于内网复用。

## CLI 与 Agent

开发构建使用现有 Docker Compose / Buildx；可用 `--build-script ./build.sh` 复用脚本。缓存默认开启，`--no-cache` 显式关闭；默认从目标元数据取得平台，也可显式 `--platform`。跨架构组需要具备多平台能力的 builder。

```bash
dockerun deploy gateway --env test --build --wait
dockerun deploy gateway --env test --release 12 --wait
dockerun promote gateway --release 12 --to prod --wait
dockerun rollback gateway --release 10 --env prod --wait
dockerun status 42
```

失败返回非零退出码，`--wait` 返回实际任务结果，主机组会等全部节点。版本晋升保持镜像身份并采用目标环境参数。恢复可选择该版本在此环境上次成功的配置；CLI rollback 默认请求恢复该配置，找不到历史快照时明确拒绝。网页确认差异后再发布；历史配置快照用于失败自动恢复。

Agent 要求 HTTPS，回环地址仅供本机验证。通过 `DOCKERUN_URL`、`DOCKERUN_AGENT_TOKEN` 和持久化 `--data-dir` 启动。实例标识、文件锁、任务租约、本机执行日志和结果文件共同避免重复替换。凭据只访问绑定节点任务，不能管理用户或其他节点。目标离线保持等待，不自动改派。固定组逐机准备、部署、检查；失败暂停后续目标，逐节点记录真实版本。

## 维护与恢复

仓库维护先切换只读，等待在途请求结束，再创建持久化 GC 任务，保留未打标签的 manifest。任务在后台执行，离开页面不会中断；完成后保持管理员开启的只读状态。原始 `/v2/` 和网页删除都检查保护引用；镜像层不开放独立 DELETE，交由 GC 清理。

管理员从“仓库与维护”创建持久化完整备份任务，刷新页面后仍可查询状态并下载结果。SQLite 使用一致性快照；MySQL 通过 `mysqldump --single-transaction`。本机执行和包导入在备份窗口暂停，Registry 只读。备份包含元数据、keys、Registry 内容、发布材料和任务材料；业务数据卷及外部 Registry 自身内容需单独备份。

备份和 GC 任务按提交顺序串行执行，同类任务排队或执行期间重复提交会返回已有任务。备份结束后恢复原来的手动维护状态，执行中不能恢复写入；排队或运行中的 GC 也会阻止结束只读维护。页面只在存在活动任务时轮询；失败任务可重新提交。进程重启后，正在运行的任务会标记为失败，未完成的备份不会提供下载。

```bash
# 停止当前 Dockerun，恢复到新的空目录；不会覆盖已有目录。
dockerun restore --file dockerun-backup.tar.gz --data-dir /var/lib/dockerun-restored
```

MySQL 备份会解出 `metadata.sql`，需恢复到专用 MySQL 数据库，再配置连接启动。不要将材料已解包视为数据库已恢复。使用原安装的外部密钥配置；导入材料路径会在启动时依据恢复目录重新定位。仅数据库文件不属于完整备份。

## API 摘要

| API | 行为 |
|---|---|
| `GET/POST /api/users`、`PATCH /api/users/:id` | 管理员账号维护 |
| `GET /api/auth/me`、`POST /api/auth/change-password` | 本人信息与密码 |
| `GET/POST/DELETE /api/auth/tokens[/:id]` | 本人自动化令牌 |
| `GET /api/builtin/catalog`、`GET /api/builtin/tags` | 有权限用户浏览内置仓库 |
| `DELETE /api/builtin/manifests` | 管理员受保护删除 |
| `POST /api/builtin/maintenance` | 指定 `read_only` 切换管理员只读窗口；只读窗口内以 `gc: true` 创建后台 GC 任务，返回 `job_id` |
| `GET /api/builtin/maintenance` | 最近 100 条 GC 任务及执行结果 |
| `GET/POST /api/system/backups`、`GET /api/system/backups/:id/download` | 完整备份任务及已完成文件下载 |
| `GET/POST /api/applications`、`GET /api/applications/:id` | 应用及环境、版本 |
| `GET /api/managed-templates` | 内置及自定义应用模板，含版本、参数表单及独立应用定义 |
| `POST /api/managed-templates/:id/applications` | 原子创建应用及首个环境；校验目标主机和模板参数，不直接部署 |
| `POST /api/applications/:id/template` | 将未插值的应用定义保存为新的自定义模板版本 |
| `PUT /api/applications/:id/scripts` | 管理员保存应用脚本，正文为 `{"scripts":{"before":"…","deploy":"…","after":"…"}}` |
| `PUT /api/applications/:id/environments` | 创建或更新环境，支持固定目标列表；`script_mode` 为 `inherit`、`override` 或 `disabled`，`scripts` 为独立脚本；未指定仓库时采用内置来源 |
| `POST /api/applications/:id/releases` | 创建不可变本机/仓库版本 |
| `POST /api/deployments/preflight` | 定义与差异预览，执行前再检查镜像及资源 |
| `POST /api/deployments` | 以幂等标识创建持久化任务；`restore_configuration` 可请求历史成功参数 |
| `GET /api/deployments[/:id]`、`GET /api/deployments/:id/events` | 历史、逐机状态、增量日志及 SSE |
| `POST /api/releases/:id/promote` | 用相同镜像身份在目标环境创建新任务 |
| `GET/PUT /api/bundles/uploads/:sha256` | 上传偏移与分块 |
| `POST /api/bundles/uploads/:sha256/complete` | 校验并创建异步导入任务 |
| `POST /api/bundles/:id/version` | 确认材料为待发布版本 |
| `POST /api/bundles/publish` | 显式将包版本镜像入库 |
| `GET/POST/PATCH /api/targets[/:id]` | 管理员主机配置 |
| `/agent/poll`、`/agent/tasks/:id/...` | 绑定节点领取、心跳、日志及结果 |

所有业务管理接口均在服务端鉴权，普通用户无法通过直接请求绕过菜单。历史 Shell 通过统一任务、日志和超时保留兼容；严格离线和通用恢复用于受管 Compose。
