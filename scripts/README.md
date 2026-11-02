# 辅助脚本目录

本目录包含历史运维脚本、外部 Registry 配置工具及独立 Docker 部署工具。当前 Dockerun 一体化安装与维护以[部署指南](../docs/DEPLOYMENT.md)为准，不直接套用旧的数据目录和容器名。

## 脚本适用范围

| 路径 | 用途与边界 |
| --- | --- |
| `setup-insecure-registry.sh` | HTTP Registry 的 daemon 配置辅助工具，执行前核对目标配置并保留已有条目 |
| `setup-registry-https.sh` | 独立外部 Registry 的证书配置工具，不是内置引擎的安装步骤 |
| `install-docker.sh` | 主机 Docker 安装辅助工具；当前控制台安装要求预先准备 Docker Engine |
| `backup.sh`、`restore.sh`、`update.sh` | 旧版运维流程，默认 `./data`、旧 Compose 命令；不适用于当前一体化完整备份恢复 |
| [docker-deploy/](docker-deploy/README.md) | 独立 Bash 部署工具及示例，按该目录文档使用 |

旧备份脚本直接复制数据库并打包目录，没有当前应用的完整备份格式和一致性协调；旧恢复脚本不能用于恢复网页下载的完整备份。

## 当前入口

- **构建镜像**：根目录 `build.sh`，按 `PLATFORMS`、`IMAGE`、`PUSH` 等环境变量配置，见[构建与交付](../docs/BUILD_AND_DEPLOY.md)。
- **启动服务**：根目录 `run.sh` 只运行 `docker compose up -d --no-build --pull never`，没有子命令。
- **备份恢复与升级**：通过「仓库与维护」创建完整备份，CLI 恢复到新空目录，见[部署指南](../docs/DEPLOYMENT.md#备份与恢复)。
- **HTTP 本地调试**：见[开发指南](../docs/DEVELOPMENT.md#本地-http-仓库调试)。
- **内置仓库 HTTPS**：见[部署指南](../docs/DEPLOYMENT.md#https-反向代理)，不需要独立 Registry 容器。

这些脚本的保留不表示当前控制台会自动调用它们。执行会修改主机或重启 daemon 的辅助脚本前，先阅读脚本内容并核对实际环境。
