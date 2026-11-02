# Docker 网络使用指南

管理员可在网络页面查看、创建、删除 Docker 网络，以及连接或断开容器。操作作用于当前连接的 Docker Engine；普通用户的内置仓库权限不包含网络管理权限。

## 应用网络

新应用优先在 Compose 中声明网络，由对应应用环境的 Compose project 管理。例如：

```yaml
services:
  web:
    image: nginx:1.28-alpine
    networks:
      - app
networks:
  app:
    driver: bridge
```

此片段需放入完整部署定义，并按实际选择镜像来源。只有同一网络中的容器才能通过相应服务名通信，发布端口用于网络外部访问。不同宿主机的同名 bridge 网络不会因此互通。

## 复用已有网络

先由管理员在网络页面创建所需网络，或在实际执行节点手动创建，例如：

```bash
docker network create shared-apps
```

然后在 Compose 中明确使用外部网络：

```yaml
services:
  web:
    image: nginx:1.28-alpine
    networks:
      - shared
networks:
  shared:
    external: true
    name: shared-apps
```

目标机上网络必须提前存在。远程 Agent 和固定主机组需逐机准备，不会自动复制本机网络。选择子网时核对宿主、已有 Docker 网络与 VPN 路由，避免重叠。

## 历史自动网络配置

配置结构仍保留以下字段，但**当前服务启动入口未调用旧自动网络初始化函数**，设置它们不会在启动时自动创建网络：

| 环境变量 | 代码默认值 |
| --- | --- |
| `DOCKER_NETWORK_NAME` | `ingot-net` |
| `DOCKER_NETWORK_DRIVER` | `bridge` |
| `DOCKER_NETWORK_SUBNET` | `172.88.0.0/16` |
| `DOCKER_NETWORK_GATEWAY` | `172.88.0.1` |

不要依赖旧文档所述的默认 `dockerun-net` 或启动自动建网。历史脚本若引用固定网络名，需先核对并显式创建，或迁移为 Compose 管理的网络定义。

## 查看与排查

```bash
docker network ls
docker network inspect shared-apps
```

删除网络前核对关联容器；被使用的网络需要先处理对应连接。容器无法互通时，依次核对执行节点、网络成员、服务名、监听地址和端口。固定主机组顺序发布不提供跨主机网络编排，当前安装不应按 Swarm 集群教程部署多个共享数据目录的控制台副本。

更多说明见[应用操作指南](INTEGRATED_DEPLOYMENT.md)、[配置参考](CONFIGURATION.md)和[故障排查](TROUBLESHOOTING.md)。
