# Dockerun 应用模板格式

随程序分发的应用模板统一从 `TEMPLATE_DIR` 指向的目录加载。默认值是进程工作目录下的 `./templates`；`make run`、`make dev-backend` 使用仓库内的绝对路径，容器镜像使用 `/app/templates`。

「发布工作区 → 应用模板」的 `/api/managed-templates` 接口每次查询或创建应用时重新读取目录，再合并数据库中的自建及导入模板。新增、修改或移除目录文件后刷新列表即可，无需修改 Go 代码或重启进程。未挂载外部模板目录的容器，需重新构建镜像才能分发新的目录模板；ZIP 导入的模板保存在数据库中，不修改模板目录，重启后仍可使用。

## 目录结构

```text
templates/
├── README.md
├── web-app/
│   ├── template.json          # 必需：新版元数据、参数及文件引用
│   └── compose.yaml           # 必需：由 compose_file 指定，可自行命名
├── redis/
│   ├── template.json
│   ├── compose.yaml
│   └── configs/
│       └── redis.conf        # 可选：非敏感配置文件
└── example/
    ├── template.json
    ├── compose.yaml
    └── scripts/
        ├── before.sh         # 可选：发布前附加脚本
        └── after.sh          # 可选：发布后附加脚本
```

只扫描模板根目录的直接子目录；其中存在 `template.json` 才作为新版应用模板加载。子目录名称建议与 `id` 一致。README、许可文件和仅包含旧 `meta.json` 的目录不参与新版加载。

模板按 `order` 升序排列，同值按 `id` 排序。目录模板统一标记为内置模板，网页不能删除；需要停用时移走该目录或 `template.json`。`custom-` 开头的标识保留给数据库自定义模板。

## 最小示例

`template.json` 必须是合法 JSON，不支持注释或尾随逗号：

```json
{
  "schema_version": 1,
  "id": "example",
  "name": "示例服务",
  "version": "1.0.0",
  "description": "从镜像创建一个独立服务。",
  "category": "business",
  "order": 200,
  "definition": {
    "compose_file": "compose.yaml",
    "parameters": [
      {
        "key": "IMAGE",
        "label": "应用镜像",
        "type": "image",
        "default": "team/app:v1",
        "required": true,
        "help": "请先将镜像准备到所选仓库。"
      },
      {
        "key": "HOST_PORT",
        "label": "访问端口",
        "type": "port",
        "default": "8080",
        "required": true,
        "help": "填写目标主机未占用的端口。"
      }
    ],
    "stateful": false,
    "topology": "replicated",
    "instructions": "首次手动发布成功后，可按环境配置镜像自动发布。",
    "endpoints": [
      { "name": "服务入口", "port": "HOST_PORT", "protocol": "http", "path": "" }
    ]
  }
}
```

对应 `compose.yaml`：

```yaml
services:
  app:
    image: ${IMAGE:?请填写应用镜像}
    restart: unless-stopped
    ports:
      - '${HOST_PORT}:80'
```

创建和预览只准备配置，确认发布后才启动业务容器。

## 字段说明

| 字段 | 说明 |
| --- | --- |
| `schema_version` | 必填，当前只支持整数 `1`，表示文件格式版本 |
| `id` | 必填，目录内唯一；小写字母、数字及连字符，最多 100 字符，不能以 `custom-` 开头 |
| `name`、`version` | 必填，各最多 100 字符；`version` 是模板内容版本，与文件格式版本不同 |
| `description` | 用途说明，界面文案使用简体中文，品牌与技术标识保留原文 |
| `category` | 必填，`business`（业务应用）或 `middleware`（中间件） |
| `order` | 可选，排序整数，默认 `0` |
| `definition.compose_file` | 必填，当前模板目录内的 Compose 文件相对路径 |
| `definition.parameters` | 参数表单定义；Compose 使用的 `${参数}` 均须在此声明 |
| `definition.config_files` | 可选，发布配置目标路径 → 模板内源文件相对路径 |
| `definition.stateful` | 是否含有状态业务数据；为 `true` 时禁止无状态自动恢复 |
| `definition.topology` | 必填，`single_host`（整套服务单台部署）或 `replicated`（允许固定主机组重复部署整套服务） |
| `definition.storage` | 可选，数据卷与持久化说明 |
| `definition.initialization` | 可选，首次启动、建表或外部材料准备说明 |
| `definition.instructions` | 可选，使用、发布与更新说明 |
| `definition.endpoints` | 可选，访问入口；`port` 引用参数名称，`protocol`、`name` 必填，可附 `path` |
| `definition.platform` | 可选，镜像目标平台，例如 `linux/amd64` |
| `definition.execution` | 可选，沿用应用定义的执行设置，导出和导入时保留；使用前应检查其中的部署命令和检查命令 |
| `script_files` | 可选，附加脚本阶段 → 当前模板目录内的脚本相对路径 |

元数据不支持未知字段，字段拼写错误会明确报错。文件引用只允许模板目录内的普通文件，不接受绝对路径、`..`、反斜杠或越界符号链接。元数据上限 1 MiB，Compose 上限 2 MiB，单个配置文件上限 8 MiB，单个附加脚本上限 256 KiB；文件不能含空字符。

目录不存在、已声明文件缺失、格式错误、ID 重复或参数未声明时，接口返回带模板目录的错误，整个列表加载失败，修复后重试；不会静默跳过损坏的新版模板，也不会退回代码内置目录。空目录返回空的预设列表，数据库自定义模板仍会合并。

## 参数与镜像

参数字段为 `key`、`label`、`type`、`default`、`required` 和 `help`。`key` 以字母或下划线开头，只含字母、数字及下划线，模板内不得重复；`label` 为非空中文名称。

| 类型 | 行为 |
| --- | --- |
| `string` | 普通文本 |
| `image` | 完整镜像名或相对仓库的镜像名称与版本；相对名称自动加所选仓库地址 |
| `port` | 1–65535 的端口 |
| `path` | 目标主机的绝对路径；实例数据目录不应写入公共默认值 |
| `password` | 默认隐藏，必须留空默认值，凭据仅保存在环境中 |

非敏感参数未提供时使用 `default`。参数名称包含 `password`、`passwd`、`secret`、`token`、`credential` 或 `private_key` 时必须使用 `password` 类型；其他密钥字段也应显式选择此类型。不要把密码、令牌、私钥或固定实例凭据写进 Compose、配置文件及脚本。

Compose 用 `${KEY}` 引用参数，必填值建议使用 `${KEY:?中文提示}`。容器内 Shell 的变量写成 `$$KEY`，避免被 Compose 提前插值。Nacos 模板的 `NACOS_GRPC_PORT` 是系统根据 `NACOS_PORT + 1000` 派生的唯一保留参数，无需重复声明；令牌仍须满足 Base64 解码后至少 32 字节。

镜像标签使用固定版本，不自动跟随新标签。默认使用内置仓库，镜像须提前推送到所选仓库。切换仓库会保留镜像名称及版本；发布时继续锁定镜像摘要。

## 配置文件和脚本

在 `definition` 中添加配置文件引用，例如：

```json
"config_files": {
  "configs/redis.conf": "configs/redis.conf"
}
```

左侧是发布材料中的文件路径，必须位于 `configs/`，不能存在文件与目录冲突；右侧是当前模板内的源文件路径。Compose 通过顶层 `configs` 的 `file: ./configs/redis.conf` 引用，再挂载到服务中。文件内容及空文件会原样保留，参数值通过服务环境变量或环境配置提供，配置文件内容不会自动进行参数替换。

可选附加脚本示例：

```json
"script_files": {
  "before": "scripts/before.sh",
  "after": "scripts/after.sh"
}
```

支持 `before`、`deploy`、`after` 三个阶段，文件内容加载后存为应用脚本快照。普通模板使用 `before` 与 `after`，由系统执行 Compose 主部署。`deploy` 保留专用启动脚本的兼容能力：未配置显式主部署命令时，它会替代系统 Compose 启动命令，之后仍按 Compose 定义核对容器及健康状态；它不是部署后的附加阶段。

脚本由 Bash 执行，创建应用时检查语法，非零退出码停止后续阶段。脚本只能在确认部署后运行，不会在模板加载、创建应用或预览时执行。前后置脚本不要重复启动整套 Compose 或删除其他应用容器，也不要依赖旧脚本导出的变量；参数统一在环境中声明。

受管 Compose 不接受 `include`、`build`、`extends`、`env_file`、`profiles` 等外部构建或包含方式，需提供完整 Compose 与已构建镜像。

## 持久化、版本与现有应用

- 普通数据采用不带固定 `name` 的命名卷，让 Compose 按应用和环境隔离。不要写固定 `container_name`。
- 外部共享网络由管理员预先创建，并在模板参数中选择；固定主机组只是逐机重复部署整套应用，不能代替跨主机集群编排。
- 必须使用主机文件的服务（如 Registry 的 htpasswd、Docker Socket）应在初始化说明中注明，文件由目标主机提供，不能伪装为随程序分发的非敏感配置。
- 应用创建后保存 Compose、配置文件、参数定义与附加脚本的独立快照。更新或移除模板不会修改已有应用、历史版本或业务数据。
- 修改模板内容时同步更新 `version`；创建接口会拒绝页面持有的旧版本，管理员应刷新后重新确认配置。
- 迁移模板格式不会导入旧容器或旧主机目录中的数据；已有服务的切换和业务数据迁移需要另外安排。应用备份也不等于业务数据卷备份。

## ZIP 导入与下载

管理员可在「发布工作区 → 应用模板」点击「导入模板」，选择或拖入 ZIP，先预览校验结果，再确认保存。预览可检查模板标识、版本、参数数量、配置文件、附加脚本以及各文件原文。预览和确认导入都不会执行脚本、拉取镜像或创建应用。

ZIP 使用本文的标准格式，支持以下组织方式：

- 单个模板的 `template.json`、Compose 和相关文件直接放在 ZIP 根目录。
- 根目录包含一个或多个模板目录，每个目录有自己的 `template.json`。
- 上述内容再包一层 `templates/` 或项目目录，最多自动展开四层单一外层目录。

ZIP 最大 32 MiB，解压后总内容最大 128 MiB，最多 2048 个目录和文件、100 个模板，单文件最大 8 MiB；元数据、Compose、脚本继续遵守各自更小的上限。单个模板序列化后的定义及附件分别不超过 15 MiB。内容必须是 UTF-8 文本；不接受绝对路径、路径穿越、符号链接、特殊文件、加密条目、重复路径及未声明的业务文件。系统忽略 `.DS_Store` 和 `__MACOSX/`。

可附带 `README`、`LICENSE`、`COPYING`、`NOTICE`（支持规定的 Markdown 或文本后缀）、`*-LICENSE` 和 `docs/` 下的说明文件，导出时一并保留。包根目录的公共说明会随每个模板保存；若与模板自己的同名说明不同，公共文件放入 `docs/package/`，保留两份原文。不要将镜像归档、主机凭据、实例环境值、业务数据或其他无关文件放入模板包。下载保留模板中已有的文本，管理员仍应检查脚本和配置中是否误写了固定凭据。

跨实例用 `template.json` 的 `id` 作为稳定模板标识，用 `version` 区分版本。接口同时返回稳定的 `key`，本机 `id`（例如 `custom-12`）只用于本实例的资源操作，不作为跨实例身份。

| 预览结果 | 确认行为 |
| --- | --- |
| 新增模板 | 保存为导入模板 |
| 已有标识的新版本 | 新增版本，保留旧版本 |
| 相同标识、版本和内容 | 默认跳过，避免重复导入 |
| 相同标识和版本、内容不同 | 默认跳过；可填写新标识和名称，另存为新模板 |

不会覆盖已有版本。确认时重新校验冲突，并在单个数据库事务中保存所选模板；任一模板失败时整批不写入。同一预览重复提交相同选择返回原结果。预览仅供上传它的管理员访问，有效期 30 分钟；每位管理员最多保留 3 个待确认预览，服务最多 64 个。关闭弹窗或重新选择会清理预览，服务启动及后续预览请求也会清理过期文件。

每行的下载按钮可下载单个模板，勾选后可「批量下载」。内置、自建和导入模板均可下载；导入模板可删除，内置模板继续由目录管理。下载生成标准 ZIP，包含参数定义、Compose、配置文件、附加脚本、执行设置和说明许可，不包含应用实例的环境值、目标主机、仓库关联、镜像或业务数据。在另一实例导入后，按当地环境重新选择主机、仓库并填写参数。

同一 ZIP 可包含同标识的多个不同版本。批量下载这类模板时，目录名称会附加内容摘要，目录内 `template.json` 的稳定标识保持一致。Nacos 集群模板下载同时包含 `NACOS-LICENSE`。

## 内置模板

当前共 13 个预设，均采用 `template.json` 与 Compose 格式：

- 单服务应用、MySQL、Redis、Nacos 单机、Nacos 单机三节点集群：提供持久化和初始化说明；MySQL 业务账号按实际需要创建。MySQL 8.0.44 首次初始化密码不能包含单引号、双引号、反斜杠或控制字符，创建应用时会提前校验。
- Kafka 单机、Kafka 单机三节点集群（含 AKHQ）：使用内部服务 DNS 与独立数据卷；对外地址和集群 ID 创建时填写，Controller 端口不映射宿主机。三节点仍在同一台主机内。
- Registry：补充镜像存储卷；htpasswd 由目标主机认证目录提供。
- Portainer：保留 Docker Socket 连接、环境独立的配置卷及首次管理员初始化。
- Nginx Proxy 与 ACME Companion：保留共享外部网络，证书和 ACME 状态改为环境独立的数据卷；ID 使用 `nginx-proxy`。
- Ingot 认证、网关、权限服务：使用固定镜像版本，依赖地址和凭据按环境填写；应用数据保存在独立命名卷，标记为有状态，默认不映射宿主端口；共享网络中的服务名分别为 `ingot-auth`、`ingot-gateway` 和 `ingot-pms`，不同环境应使用各自的共享网络。
- 单服务应用首次手动发布后，可在环境中配置镜像自动发布；重复的 `webhook-example` 示例已合并至此模板。

Nacos 建表 SQL（含版权声明）保存在 `nacos-cluster/configs/nacos-mysql.sql`，并随模板加载；完整许可保存在 `NACOS-LICENSE`。

核对依据包括 [Kafka 3.8.0 镜像参数检查](https://github.com/apache/kafka/blob/3.8.0/docker/resources/common-scripts/configure)、[镜像数据目录定义](https://github.com/apache/kafka/blob/3.8.0/docker/jvm/Dockerfile)、[Nacos 3.1.1 镜像配置](https://github.com/nacos-group/nacos-docker/blob/v3.1.1/build/conf/application.properties)、[Nacos 就绪接口](https://nacos.io/docs/latest/manual/admin/admin-api/)及 [ACME Companion 2.6.0 用法](https://github.com/nginx-proxy/acme-companion/blob/v2.6.0/README.md)。

## Ingot 0.1.0 初始化与运行参数

这三个默认镜像构建于 2025-12-29 至 2025-12-30。数据库和 Nacos 初始化材料应与镜像版本匹配，可参考 [Ingot 对应时期的初始化材料](https://github.com/ingot-cloud/ingot/tree/f24c43a3/databases)。新版 Ingot 的模块名称、配置 ID 和表结构已经变化，不应直接混用。

| 模板 | 默认镜像架构 | 业务端口 | 容器内管理端口 | 默认业务数据库 |
| --- | --- | --- | --- | --- |
| `ingot-auth` | `linux/amd64` | 5100 | 6100 | `ingot_auth` |
| `ingot-gateway` | `linux/amd64` | 7980 | 8798 | 无 |
| `ingot-pms` | `linux/arm64` | 5200 | 6200 | `ingot_core` |

原始镜像标签的架构不同；目标主机需支持对应架构，或在所选仓库准备适配主机的同版本镜像，再填写 `IMAGE`。本次混合架构验证使用 Docker Desktop 的 amd64 兼容运行能力，不代表不同架构的 Linux 主机能够原生运行这些镜像。

先创建应用共享网络及 `NACOS_NAMESPACE` 指定的命名空间，在 `NACOS_GROUP` 指定的分组发布 `application-common.yml`，以及各自的 `ingot-service-auth.yml`、`ingot-service-gateway.yml`、`ingot-service-pms.yml`。认证和权限服务还需 `application-mysql.yml` 及已经初始化的业务数据库。三个服务的命名空间、分组和依赖凭据应保持一致。

`JAVA_OPTS` 默认使用 256 MiB 初始堆和 768 MiB 最大堆，可按主机容量调整；`SPRING_PROFILES_ACTIVE=prod` 单独启用生产配置。管理端口仅监听容器回环地址，健康检查必须取得 `UP` 状态。网关使用实际的 8798 管理端口，覆盖镜像内错误的 8980 健康检查。

每个模板的 `configs/runtime.yml` 由 Spring 在容器内解析环境变量，数据库名称、AES 密钥、认证 JWK 主密钥和 MinIO 参数会覆盖旧 Nacos 配置中的固定值。配置文件只包含参数引用，凭据仍由环境提供。认证主密钥及 AES 密钥使用原始 UTF-8 文本，长度必须为 16、24 或 32 字节；当前镜像不会先对主密钥做 Base64 解码。推荐用 `openssl rand -hex 16` 生成 32 字节文本，并妥善保存已有密钥。

`SERVICE_AUTH_HOST` 填写客户端可达的认证主机及端口，不含协议，当前配置添加 `http://`；默认 `ingot-auth:5100` 用于共享网络内部访问。对外部署时应与实际认证入口一致，并在对应业务配置中安排 TLS 与登录页面。业务接口使用用户令牌，并携带与令牌一致的 `Tenant` 请求头；客户端凭据模式的机器令牌不能替代用户登录。

## 历史脚本兼容

仓库不再分发旧版 `meta.json`、`deploy.sh` 和 `files/`，避免与应用模板重复，以及误用旧固定地址、示例凭据和不完整脚本。原 `webhook-example` 与单服务应用的功能相同，已移除；镜像自动发布继续在应用环境中配置。原 `ingot-member` 默认镜像 `ingot/member:0.1.0` 及对应镜像仓库不存在，作为失效模板移除。

「系统管理 → 历史脚本」及 `/api/templates` 仍兼容管理员放入 `TEMPLATE_DIR` 的外部旧模板；没有外部旧模板时返回空列表。已保存到数据库的历史脚本、已有应用、历史发布和业务数据不受模板文件清理影响。新增模板统一采用本文格式。

## 开发验证

在仓库根目录运行：

```bash
cd backend
TEMPLATE_DIR=../templates go test ./internal/apptemplate ./internal/database ./internal/handler ./internal/router ./internal/release
```

测试会检查所有预设的参数、Compose、配置文件、数据卷隔离、目录刷新、错误信息、文件访问范围、从模板创建应用和内置模板删除保护；同时检查 ZIP 往返一致性、恶意归档拒绝、冲突与另存、确认重试、旧模板数据迁移和管理员权限。测试不拉取镜像，也不启动业务容器。真实服务启动、镜像可用性及目标网络仍需在部署环境验证。

2026-11-03 核验范围：

| 范围 | 结果 |
| --- | --- |
| 全部 13 个模板 | 元数据、参数插值、Compose、配置文件及应用环境的数据卷隔离校验通过；目录内无缺失引用或未使用文件 |
| 单服务应用、MySQL、Redis、Registry | 临时 Compose 项目启动及 HTTP、SQL、缓存读写、仓库认证检查通过；单服务应用使用测试 Nginx 镜像 |
| Kafka 单机与三节点集群 | 固定版本镜像启动、消息生产与消费、AKHQ 页面检查通过 |
| Nacos 单机与三节点集群 | 控制台及各节点实际就绪响应检查通过；集群 MySQL 建表完成。健康检查同时验证响应内容，避免 HTTP 200 的未就绪响应被误判 |
| Portainer、Nginx Proxy 与 ACME Companion | 固定版本启动、管理接口或 Nginx 配置、共享卷检查通过；Socket 模式的 Portainer 无需 Agent 密钥 |
| Ingot 认证服务 | 固定版本启动和健康检查通过；OAuth2 元数据、JWK 公钥、Redis 私钥加密标记、客户端令牌签发及正常用户名密码登录检查通过 |
| Ingot 网关服务 | 固定版本启动、8798 管理端口健康检查、经 Nacos 发现并转发认证元数据与 JWK 接口检查通过 |
| Ingot 权限服务 | 固定版本启动和健康检查通过；认证服务查询用户成功；正常登录后携带用户令牌及租户头上传文件到临时 MinIO，读回内容一致 |

临时项目使用独立名称、网络、数据卷与本机随机端口，核验后已清理。Ingot 验证使用对应时期的源码初始化材料、临时客户端和测试账号，经过正常密码登录流程，不包含真实环境数据；未覆盖所有业务接口、浏览器登录页面及跨版本数据库迁移。ACME 检查覆盖容器与配置，不包含公网域名解析和证书签发。以上结果不替代目标环境的镜像准备、网络连通性与业务验收。
