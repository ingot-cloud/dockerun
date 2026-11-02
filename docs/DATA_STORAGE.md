# 数据存储与迁移

Dockerun 支持命名卷和目录绑定。两种模式的容器内部目录固定为 `/var/lib/dockerun`，都保存数据库文件、密钥、内置仓库和发布材料。MySQL 只替代元数据库，仍需保留这些文件。

## 选择模式

新安装默认使用命名卷，Docker 管理目录和权限，重建容器不会删除数据。需要直接通过宿主文件系统查看、备份或管理存储位置时，可使用目录绑定。

| 场景 | `deploy.env`（deploy.sh） | `.env`（Compose） |
| --- | --- | --- |
| 命名卷 | `DATA_MOUNT_TYPE=volume`、`DATA_VOLUME=dockerun-data` | `DATA_VOLUME=dockerun-data` |
| 目录绑定 | `DATA_MOUNT_TYPE=bind`、`DATA_DIR=/ingot-data/dockerun` | `COMPOSE_FILE=docker-compose.yml:docker-compose.bind.yml`、`DATA_DIR=/ingot-data/dockerun` |

绑定目录需为宿主机绝对路径；部署脚本可在部署时创建，Compose 覆盖文件要求提前创建。`deploy.env` 与 `.env` 是不同入口的配置文件，不能混用。旧 `deploy.env` 仅设置 `DATA_DIR` 时继续采用绑定模式；使用示例重新配置信息时，也要显式保留 bind，避免误用空卷。

服务实际读取的 `DATA_DIR` 始终为容器内路径。脚本在 bind 模式额外传入 `HOST_DATA_DIR`，在 volume 模式传入 `DATA_VOLUME`；后端通过 Docker 查询命名卷挂载点，转换应用发布中的绑定源，不要求宿主与容器使用相同路径。原生进程不设置这两个映射变量时保持原行为。请使用本次更新后重新构建的镜像，旧镜像没有路径转换能力。

## 查看和取出数据

容器运行时，使用真实容器名查看文件：

```bash
docker exec dockerun ls -lah /var/lib/dockerun
docker cp dockerun:/var/lib/dockerun/metadata.sql ./metadata.sql
```

`metadata.sql` 仅在 MySQL 备份恢复后存在。密钥和数据库包含敏感内容，取出文件应保存在受控目录。

命名卷可在 Docker Desktop 的 Volumes 页面查看。命令行先确认卷名，避免拼写错误时 Docker 创建新空卷；容器停止时也能用只读辅助容器查看或导出单个文件：

```bash
docker volume inspect dockerun-data
docker run --rm --pull never --network none --entrypoint /bin/sh \
  --mount type=volume,src=dockerun-data,dst=/data,readonly \
  dockerun:2.0.0 -c 'ls -lah /data'
# 恢复 MySQL 备份后，可将上一条的命令换成 cat /data/metadata.sql，并重定向到受控文件。
```

用实际部署镜像替换 `dockerun:2.0.0`。`docker volume inspect` 的 `Mountpoint` 是 Docker daemon 所在主机的路径；Docker Desktop 中通常位于虚拟机内，不能当成 macOS/Windows 的普通目录直接访问。不要硬编码 `/var/lib/docker/volumes`，也不要直接修改运行中的 SQLite、仓库内容或密钥。

目录绑定可在宿主 `/ingot-data/dockerun` 查看，容器内仍使用 `/var/lib/dockerun`。完整备份与恢复优先使用 Dockerun 维护页面和 CLI，步骤见[部署指南](DEPLOYMENT.md#备份与恢复)。直接复制活跃的数据库和仓库目录不能替代一致性备份。

## 从目录绑定迁移到命名卷

不需要为了升级强制迁移，保留原绑定目录即可。确需切换时安排停机窗口；更改配置只会切换挂载，不会自动复制文件。下面假定原目录为 `/ingot-data/dockerun`，新卷为尚未使用的 `dockerun-data-migrated`：

1. 暂停发布、上传和其他写入，创建并下载完整备份，另存部署配置。MySQL 同时保留独立数据库备份。
2. 停止 Dockerun：脚本安装用 `./deploy.sh stop`，Compose 安装用 `docker compose stop dockerun`。确认旧实例不会再启动或写入。
3. 用实际部署镜像，将原目录复制到新的空卷。以下命令拒绝覆盖非空目标，并排除可重新生成的运行时 socket：

```bash
docker volume create dockerun-data-migrated
docker run --rm --pull never --network none --entrypoint /bin/bash \
  --mount type=bind,src=/ingot-data/dockerun,dst=/source,readonly \
  --mount type=volume,src=dockerun-data-migrated,dst=/target \
  dockerun:2.0.0 -e -o pipefail -c '
    test -z "$(ls -A /target)"
    tar -C /source --exclude=./run -cpf - . | tar -C /target -xpf -
  '
```

4. 脚本安装在 `deploy.env` 设置 `DATA_MOUNT_TYPE=volume`、`DATA_VOLUME=dockerun-data-migrated`，执行 `./deploy.sh check` 和 `./deploy.sh update`。Compose 安装移除 bind 覆盖文件设置及宿主 `DATA_DIR`，设置 `DATA_VOLUME=dockerun-data-migrated`，核对 `docker compose config` 后执行 `docker compose up -d --no-build --pull never`。
5. 核对原用户、应用记录、仓库拉取、凭据解密和一次实际发布。已有业务容器会保留创建时的绑定源；重新发布并确认挂载指向新存储后，才能考虑清理旧目录。写死旧宿主路径的历史脚本和应用定义需要核对；读取任务材料改用 `WORK_DIR`，不要把旧根目录写死在脚本中。

迁移过程中保留原目录和备份。不要同时运行两个控制台实例共享同一数据库；恢复旧文件而不恢复对应数据库状态可能造成不一致。命名卷迁移到目录绑定时同理：停机、复制到新空目录，再切换配置并验证。

## 升级和删除

只更新镜像版本时，保留原模式与存储名称，脚本安装执行 `./deploy.sh update` 即可。`stop`、`restart` 不会应用新环境变量；`remove` 不删除数据卷或宿主目录。Compose 的 `down` 默认保留命名卷，**`down -v`、`docker volume rm` 以及卷清理命令可能删除数据**，不作为常规升级步骤。
