# 镜像仓库使用指南

Dockerun 同时提供内置 Registry 和外部 Registry 连接。两者与宿主 Docker 的本地镜像、离线材料分别存储，导入本地镜像不会自动推送到仓库。

## 内置仓库

内置仓库由同一 Dockerun 进程中的 CNCF Distribution v3.1.2 提供，不需要独立 Registry 容器或单独的 `config.yml`。网页、`/v2/` 和 `/registry/token` 共用 `PUBLIC_URL` 所指向的入口，镜像保存在 `${DATA_DIR}/registry/`。

先从网页初始化 `admin`，再使用标准 Docker 命令。以下假定已经配置可信 HTTPS 的 `dockerun.example.com`：

```bash
docker login -u admin dockerun.example.com
docker pull nginx:1.28-alpine
docker tag nginx:1.28-alpine dockerun.example.com/demo/nginx:1.28-alpine
docker push dockerun.example.com/demo/nginx:1.28-alpine
docker pull dockerun.example.com/demo/nginx:1.28-alpine
```

登录输入网页账号密码，或该账号创建且具有相应权限的自动化令牌。命令地址不带 `https://`，非默认端口需带上端口；镜像名称示例为 `demo/nginx`，不在配置镜像名称时重复填写域名。

管理员可在用户管理中授予普通用户无权限、只读或读写权限。只读可浏览和拉取，读写增加推送；普通用户不能部署应用或管理宿主资源。自动化令牌权限不能超出账号当前授权。

初始化后，未认证访问 `/v2/` 返回带认证挑战的 `401` 是正常行为；令牌 realm 必须指向 daemon 可达的实际入口。HTTP 本地调试及 Docker Desktop 设置见[开发指南](DEVELOPMENT.md#本地-http-仓库调试)。正式部署见[HTTPS 配置](DEPLOYMENT.md#https-反向代理)。

## 外部仓库连接

管理员在外部仓库页面添加已有 Registry，填写名称、完整 URL、认证信息及默认选项，通过连接测试后保存。外部 URL 按实际使用 `https://registry.example.com` 或受控环境中的 HTTP 地址。

连接测试与网页浏览由 Dockerun 服务发起；实际拉取、推送或远程发布可能由其他 Docker daemon、Buildx 或 Agent 发起。因此连接测试通过不等于所有执行节点都具备网络、证书及账号权限。

“不安全连接”影响 Dockerun 对该外部连接的 HTTPS 校验，不会自动修改宿主 Docker 的 `insecure-registries` 或构建器信任。私有 CA 应在各实际连接环境安装信任。保存的外部凭据采用持久化密钥加密，详见[凭据加密](REGISTRY_PASSWORD_ENCRYPTION.md)。

远程 Agent 默认不使用外部管理凭据，管理员需另外设置外部仓库中实际受限为只读的拉取账号；其 ACL 由外部 Registry 执行。删除连接配置不删除外部服务器上的镜像内容。

## 用于应用发布

应用发布可按镜像地址拉取，也可选择内置或外部仓库。仓库来源需使用包含所选仓库域名的镜像引用，创建版本时解析并锁定 digest；环境中的标签、参数和自动发布规则分别保存。

应用模板默认使用内置仓库，需要先把镜像准备到该仓库。镜像自动发布需为对应环境明确保存监听仓库、镜像名称、标签规则及允许触发的用户；单纯推送镜像不会替未配置的环境发布。

离线归档导入仅准备材料，既不自动推送入库，也不自动替换服务。需要在内网复用时，由管理员显式执行将包内镜像发布到内置仓库的操作。详细流程见[一体化操作指南](INTEGRATED_DEPLOYMENT.md)。

## 删除、维护与备份

内置仓库的镜像删除由管理员执行，当前版本、恢复历史和多架构索引引用受保护。直接调用 `/v2/` 也会执行保护检查；镜像层不提供独立删除，交由维护 GC 处理。

从「仓库与维护」切换只读后提交 GC，等待后台任务完成，再按需恢复写入。GC 保留未打标签的 manifest，不能简单理解成删除所有没有标签的镜像。应用历史版本保留策略也会影响可删除范围。

完整备份包含内置仓库内容和对应元数据、密钥；外部 Registry 内容需单独备份。关闭内置引擎 `REGISTRY_ENABLED=false` 不会删除已有镜像，额外环境变量需显式传入 Compose，见[配置参考](CONFIGURATION.md)。

## 兼容与排查

已验证的 OCI profile 范围和已知边界见[验证记录](VALIDATION.md)，不宣称 OCI 1.1 全量一致。认证超时、证书或推送失败见[故障排查](TROUBLESHOOTING.md)。旧的独立 Registry 证书和 HTTP 配置专题仅供外部仓库排查，不应当作内置引擎的安装步骤。
