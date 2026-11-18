# 三组真实 Docker 场景测试计划

日期：2026-11-01  
原则：不改产品代码；只在 `artifacts/` 放测试助手。断言必须落到 Docker 容器/镜像身份，不能只看 HTTP 200。

## 隔离

| 项 | 值 |
|---|---|
| 数据目录 | `/tmp/dockerun-scenario-2222` |
| 服务 | `127.0.0.1:18080`（沿用已配置的 insecure-registries） |
| 内置仓库仓库名 | `dockerun-scenario/demo` |
| 命令应用容器 | `dockerun-sc1-app-2222` |
| 本机导入应用容器 | `dockerun-sc3-app-2222` |
| 镜像 A / B | 不同 LABEL 与 CMD 的 alpine 派生镜像，Image ID 必须不同 |

产品路径（来自 `RunDeployment` / `refreshImageReferences`）：

- 命令部署且 `replace_containers=true` 且**自动发布**：先停删关联容器 → 清理同 tag 本地引用（无其他容器占用时）→ 按 digest 拉取 → 刷新本地 tag → 执行命令。
- 同上但**手动发布**：先准备镜像，再停删关联容器，不清理本地 tag。
- Compose：由 Compose 重建服务；不走上述命令替换链。
- 发布页「离线镜像归档」：`POST /api/bundles/uploads/:sha/complete` `kind=image`，再 `POST /releases` `source=local` + `bundle_id`。导入只 `docker load`，不停止容器。

## 场景

### 场景 1：同 tag 重推自动发布

1. 保存命令部署配置（`$DOCKERUN_IMAGE`、关联容器、开启替换），开启自动发布，监听 `dockerun-scenario/demo`，规则 `*`。
2. 推送镜像 A 到 `demo:latest`，等待自动任务成功；记录容器 ID、Image ID、digest。
3. 将**不同内容**的镜像 B 推到**同一 tag** `demo:latest`。
4. 断言：
   - 产生新的 `automatic` 任务且 success
   - 任务日志顺序包含：停止关联容器 → 删除关联容器 → 清理同 tag 本地镜像引用 → 拉取/准备新镜像
   - 新容器运行中，Image ID / digest **不同于**步骤 2
   - `docker inspect` 中该 tag 指向新 Image ID

### 场景 2：不同 tag 部署

当前运行 tag A（`latest` = B）。将镜像 A 推到 **tag `v2`**（内容与当前运行不同）。

- **2a 自动**：推送触发自动发布；断言停删当前容器、拉取 `v2`、运行 Image ID 等于 `v2` 对应 ID。
- **2b 手动 Compose**：独立应用手动创建 `source=registry` 版本（先 `v1`=A，再 `latest` 或 `v2`）；断言第二次部署后运行 Image ID 切换到目标 tag。

### 场景 3：发布时手动导入本机镜像

路径：发布工作区「离线镜像归档」（`kind=image`），**不是**完整发布包 `kind=bundle`。

1. 预先运行关联容器（镜像 A）。
2. `docker save` 镜像 B 为 tar，分块上传并 `kind=image` 完成导入。
3. 断言导入期间原容器仍在运行、容器 ID 不变（导入只准备镜像）。
4. 用该材料创建 `source=local` 命令版本并部署。
5. 断言：停删原容器；新容器使用导入材料锁定的本地 Image ID（B）。

## 证据

每个场景写入 `artifacts/evidence/scenario/`：docker ps/inspect、镜像 ID、digest、任务日志、命令。清理隔离 project、容器、数据目录与测试镜像。
