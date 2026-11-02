# 配置参考

本说明以 `backend/internal/config/config.go`、根目录 `Makefile`、`.env.example` 和 `docker-compose.yml` 为准。默认部署使用 SQLite，密钥自动生成并持久化，不需要填写固定的默认 JWT 密钥。

## 配置如何生效

- **容器部署**：根目录 `.env` 供 Docker Compose 插值，只有 Compose 的 `environment` 中引用的变量才会传给服务。修改后执行 `docker compose up -d --no-build --pull never` 重建服务，`restart` 不会应用新的环境变量。
- **本地开发**：`make dev-backend`、`make run` 不自动读取部署用 `.env`，通过进程环境或 Make 参数传入。Make 会为本地开发覆盖数据目录、数据库类型、模板目录、监听地址和公开入口。
- **直接运行二进制**：后端读取进程环境；未设置时使用代码默认值。

```bash
# 本地开发示例
make dev-backend PUBLIC_URL=http://localhost:8080
# MySQL 账号和连接串通过当前进程环境提供
make dev-backend DATABASE_TYPE=mysql
```

仅在 `.env` 中新增 `REGISTRY_ENABLED` 等变量不会自动传入容器，需要同时在 Compose 服务中添加对应的 `environment` 项。

## 服务、存储与数据库

下表“代码默认值”指直接运行服务时的默认行为。

| 环境变量 | 代码默认值 | 作用与部署差异 |
| --- | --- | --- |
| `SERVER_ADDR` | `:8080` | 服务监听地址；镜像使用 `:8080`，本地 Make 使用 `127.0.0.1:8080` |
| `PUBLIC_URL` | `http://localhost:8080` | 对外根地址，也是 Registry token realm 的来源；本地 Make 使用 `http://host.docker.internal:8080` |
| `DATA_DIR` | `./data`，启动时转绝对路径 | 镜像及 Compose 使用 `/var/lib/dockerun`，本地 Make 使用仓库 `data/local` |
| `HOST_DATA_DIR` | 空 | 后端容器连接本机 Docker 时，目录绑定对应的宿主绝对路径；与 `DATA_VOLUME` 互斥 |
| `DATA_VOLUME` | 空 | 后端挂载的数据命名卷名称；启动时从 Docker 读取真实挂载点，用于转换业务容器的绑定源 |
| `DATABASE_TYPE` | `sqlite` | 仅支持 `sqlite` 或 `mysql` |
| `DATABASE_PATH` | `${DATA_DIR}/dockerun.db` | SQLite 文件；完整备份以数据库快照保存，恢复后默认文件名为 `dockerun.db` |
| `MYSQL_DSN` | 空 | MySQL 模式必填，示例见下文 |
| `GIN_MODE` | `debug` | 镜像设为 `release` |
| `TEMPLATE_DIR` | `./templates` | 旧脚本模板目录；镜像为 `/app/templates`，本地 Make 指向仓库 `templates/` |
| `IMPORT_DIR` | `${DATA_DIR}/imports` | 服务器目录导入允许读取的根目录 |
| `REGISTRY_ENABLED` | `true` | 仅值为 `true` 时启用内置引擎；设为 `false` 不删除仓库内容 |
| `TLS_CERT`、`TLS_KEY` | 空 | 服务直接启用 TLS 的证书链及私钥路径，必须同时设置 |

`PUBLIC_URL` 只接受完整 `http://` 或 `https://` 根地址，不允许路径前缀、凭据、查询参数或 fragment。所有 Docker daemon、构建器与客户端必须能访问该地址。设置为 HTTPS 不会自动生成证书或配置代理。

MySQL DSN 示例：

```dotenv
DATABASE_TYPE=mysql
MYSQL_DSN=dockerun:替换为实际密码@tcp(mysql.example.com:3306)/dockerun?charset=utf8mb4&parseTime=True&loc=Local
```

MySQL 需先创建专用数据库和账号；切换数据库类型不会搬迁原有数据，详见[MySQL 配置](MYSQL_SETUP.md)。若自定义 `DATABASE_PATH` 到数据目录之外，需要自行保存这项配置，并在恢复时核对文件位置。

## 密钥与凭据

首次启动在 `${DATA_DIR}/keys/` 中生成独立随机密钥。正常部署保留整个目录即可；不要在重建容器时清空，也不要把密钥写进仓库。

| 可选环境变量 | 自动持久化文件 | 用途及约束 |
| --- | --- | --- |
| `JWT_SECRET` | `keys/web.key` | 网页令牌签名；显式值至少 32 字节 |
| `REGISTRY_SECRET` | `keys/registry.key` | Registry 令牌签名；显式值至少 32 字节 |
| `REGISTRY_HTTP_SECRET` | `keys/http.key` | Distribution HTTP 状态签名；显式值至少 32 字节 |
| `CREDENTIALS_SECRET` | `keys/credentials.key` | 外部仓库凭据 AES-GCM 加密；显式值必须为 64 位十六进制字符串 |

显式环境变量优先于文件，环境提供的值不会自动写回相应密钥文件，必须额外备份安装配置。轮换凭据加密密钥需要配套的数据重加密，不能仅更换变量后继续使用旧密文。

`AES_KEY`、`AES_IV` 是历史网页密码传输与旧外部仓库凭据的兼容配置，不是新凭据存储密钥。已有安装若自定义过，应在升级和恢复时保留；网页的 `VITE_AES_KEY`、`VITE_AES_IV` 属于前端构建时配置，修改容器运行环境不会改变已构建网页。此兼容格式不能替代 HTTPS。详见[安全说明](SECURITY.md)及[仓库凭据加密](REGISTRY_PASSWORD_ENCRYPTION.md)。

## Compose 配置示例

根目录 Compose 已传入 `PUBLIC_URL`、`DATABASE_TYPE`、`MYSQL_DSN`、`TLS_CERT`、`TLS_KEY`，并固定 `DATA_DIR=/var/lib/dockerun`。`DOCKERUN_IMAGE` 控制服务镜像标签，不是后端环境变量，默认 `dockerun:2.0.0`。

需要关闭内置仓库或修改其他后端变量时，在原服务的 `environment` 中补充，例如：

```yaml
services:
  dockerun:
    environment:
      REGISTRY_ENABLED: "false"
      IMPORT_DIR: /var/lib/dockerun/imports
```

此片段需合并进现有配置，保留原有环境变量及挂载。默认 Compose 使用 `DATA_VOLUME=dockerun-data` 的命名卷。目录绑定在 `.env` 中设置 `COMPOSE_FILE=docker-compose.yml:docker-compose.bind.yml`、`DATA_DIR=/ingot-data/dockerun`，并提前创建该宿主目录。覆盖文件会清空后端的 `DATA_VOLUME`，传入 `HOST_DATA_DIR`，仍挂载到 `/var/lib/dockerun`。这里 `.env` 的 `DATA_DIR` 供 Compose 插值，容器内实际 `DATA_DIR` 固定，不随宿主路径变化。

自行编写 `docker run` 时，必须让 `HOST_DATA_DIR` 或 `DATA_VOLUME` 与实际挂载一致，两者只选一个。服务不会根据名称创建数据卷，也不会硬编码 `/var/lib/docker/volumes`；卷不存在或挂载点不可用会阻止启动。原生进程和旧版同路径绑定未设置这两个变量时沿用原路径。服务在容器临时目录 `/tmp/dockerun-api-*/docker.sock` 创建 Docker 路径代理，退出时清理，不写入持久化数据，也无需备份。路径转换范围见[应用脚本](INTEGRATED_DEPLOYMENT.md#应用和环境的部署脚本)。

## 本地开发参数

| Make 参数 | 默认值 | 用途 |
| --- | --- | --- |
| `FRONTEND_PORT` | `5999` | 前端开发服务端口，被占用时直接报错 |
| `BACKEND_URL` | `http://127.0.0.1:8080` | 前端开发代理的后端地址 |
| `SERVER_ADDR` | `127.0.0.1:8080` | 本地后端监听地址 |
| `PUBLIC_URL` | `http://host.docker.internal:8080` | Docker Desktop daemon 可达的宿主入口 |
| `GOPROXY` | `https://goproxy.cn,direct` | Go 工具链及模块下载代理 |
| `GOSUMDB` | `sum.golang.org https://goproxy.cn/sumdb/sum.golang.org` | 保持官方校验身份，通过镜像传输 |

```bash
make dev-backend SERVER_ADDR=127.0.0.1:8081 PUBLIC_URL=http://host.docker.internal:8081
make dev-frontend BACKEND_URL=http://127.0.0.1:8081 FRONTEND_PORT=6000
```

直接在 `frontend/` 运行 Vite 时，代理变量为 `VITE_BACKEND_URL`；默认端口同为 `5999`。更多说明见[开发指南](DEVELOPMENT.md)。

## CLI、Agent 与 Docker 连接

| 变量或参数 | 用途 |
| --- | --- |
| `DOCKERUN_URL` | CLI / Agent 连接的控制台根地址；CLI 未设置时使用 `http://localhost:8080` |
| `DOCKERUN_TOKEN` | CLI API 受限自动化令牌，部署需要管理员账号的 deploy 权限 |
| `DOCKERUN_AGENT_TOKEN` | 绑定目标主机的 Agent 凭据，与 CLI 令牌分开 |
| `dockerun agent --data-dir` | Agent 持久化目录，默认 `/var/lib/dockerun-agent` |
| `DOCKER_HOST`、`DOCKER_TLS_VERIFY`、`DOCKER_CERT_PATH` | Docker SDK / CLI 连接配置；默认容器部署挂载本机 socket |

Agent 正式连接要求 HTTPS，回环地址只用于本机验证；实际执行节点需具备 Docker、Compose、Bash 以及脚本依赖的命令。详见[一体化操作指南](INTEGRATED_DEPLOYMENT.md#cli-与-agent)。

## 系统页面与历史网络配置

任务超时、历史脚本执行配置等通过系统页面保存到数据库，不通过任意 `.env` 字段配置。环境运行参数属于业务应用，不能与 Dockerun 服务本身的环境变量混用。

代码仍保留 `DOCKER_NETWORK_NAME`、`DOCKER_NETWORK_DRIVER`、`DOCKER_NETWORK_SUBNET`、`DOCKER_NETWORK_GATEWAY`，默认值分别为 `ingot-net`、`bridge`、`172.88.0.0/16`、`172.88.0.1`。当前服务启动入口未调用旧的自动网络初始化函数，设置这些变量不会在启动时自动建网。新应用在 Compose 中声明网络，或通过网络管理页面创建后显式引用，见[网络指南](NETWORK_GUIDE.md)。
