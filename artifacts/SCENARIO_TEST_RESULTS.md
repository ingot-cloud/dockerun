# 三组真实 Docker 场景测试结果

日期：2026-11-01  
环境：Cursor Cloud VM，隔离栈 `127.0.0.1:18080`，`DATA_DIR=/tmp/dockerun-scenario-2222`  
产品代码：未修改  
助手：`artifacts/run_scenario_tests.py`

镜像 A：`sha256:b03446808ccbef93d09d41ae78607baed92ca98f3f56a16384e196c2ccee8339`（label `dockerun.scenario=a`）  
镜像 B：`sha256:3f88ddd2cc81c89db34eb596a811dbd7613ea41a746be7ecc24b8033b1976a2a`（label `dockerun.scenario=b`）

**总览：30 pass / 0 fail / 0 skip / 0 blocked**

使用的产品路径：

- 场景 1、2a：命令部署 + `replace_containers=true` + 内置仓库自动发布（`$DOCKERUN_IMAGE`）。
- 场景 2b：Compose + `source=registry` 手动预检查/确认部署。
- 场景 3：发布工作区「离线镜像归档」——`kind=image`（不是完整发布包 `kind=bundle`），再 `source=local` + `bundle_id` 创建命令版本并部署。

---

## 场景 1：同 tag 重推自动发布 — **pass**

先推送 A → `dockerun-scenario/demo:latest`，自动任务成功，容器 `dockerun-sc1-app-2222` Image ID = A。

再将 **B 推到同一 tag `latest`**。自动任务 #2 `success`。

| 子断言 | 结果 | 证据 |
|---|---|---|
| 自动发布被触发 | **pass** | 任务 #2 `automatic`，`发布成功`/检查通过 |
| 先停再删关联容器 | **pass** | `[containers] 停止关联容器 /dockerun-sc1-app-2222` → `删除关联容器 …，保留数据卷` |
| 清理同 tag 引用再按 digest 拉取 | **pass** | `清理同 tag 的本地镜像引用：…/demo:latest` → `Pulling from dockerun-scenario/demo` Digest `sha256:40c9075a…` → `已刷新本地 tag：…:latest → sha256:3f88ddd2…`（B） |
| 运行 Image ID 变为 B | **pass** | 容器 ID `78dde3ea…`→`94ab7590…`；Image ID A→B；digest `d355f290…`→`40c9075a…` |

日志顺序与 `RunDeployment` 自动命令路径一致：停容器 → 删容器 → 清理同 tag → pull digest → retag → 执行命令。

---

## 场景 2：不同 tag 部署 — **pass**

当时命令应用运行的是 `latest`=B。将 A 推到 **`v2`**。

### 2a 自动发布

| 子断言 | 结果 | 证据 |
|---|---|---|
| 推送不同 tag 触发自动发布 | **pass** | 任务 success |
| 停止当前容器 | **pass** | 日志停/删；容器 ID `94ab7590…`→`cfb88d13…` |
| 运行 ID 等于 tag v2 / 镜像 A | **pass** | 运行 Image ID = A；`demo:v2` RepoDigest `sha256:d355f290…` |

### 2b 手动 Compose

独立应用 `scenario-compose-2222`。先手动部署 `demo:v1`（A），再手动部署 `demo:latest`（B）。

| 子断言 | 结果 | 证据 |
|---|---|---|
| 手动部署 tag v1=A | **pass** | `dockerun-a2-e2-demo-1` Image ID = A |
| 手动部署不同 tag latest=B | **pass** | 任务 success |
| 容器按目标 tag 重建 | **pass** | Compose `Recreate`；容器 ID 更换；Image ID = B |

Compose 路径是「先准备镜像（保留旧服务）→ `compose up` 重建」，不走命令替换的「先停再清 tag」。这是产品对 Compose 的既定顺序，已在日志中核对。

---

## 场景 3：发布时手动导入本机镜像 — **pass**

API 路径（发布页「离线镜像归档」）：

1. `GET/PUT /api/bundles/uploads/:sha256`
2. `POST /api/bundles/uploads/:sha256/complete` `{"kind":"image","name":"image-b.tar"}`
3. `POST /api/applications/:id/releases` `source=local` + `bundle_id`
4. `POST /api/deployments/preflight` → `POST /api/deployments`

与完整发布包（`kind=bundle` + `source=bundle`）区分：本次 `kind=image`、`source=local`，清单只有一条本地 Image ID。

| 子断言 | 结果 | 证据 |
|---|---|---|
| 导入 `kind=image` 完成 | **pass** | bundle #1 `ready`，`local_loaded=true`，manifest ID = B |
| 导入不停止已有容器 | **pass** | 导入前后容器 ID 同为 `225302a0…`，仍运行 A |
| 预检查不停止容器 | **pass** | 预检查后容器 ID 仍为 `225302a0…` |
| 创建 local 版本锁定 B | **pass** | 201，`source=local`，images[0].id = B |
| 部署停删关联容器 | **pass** | 手动命令顺序：先准备镜像，再 `停止/删除关联容器`；容器换成 `b43ad4cd…` |
| 新容器使用导入的本地 Image ID | **pass** | `image` / `image_id` 均为 B 的 sha256 |

手动本地部署按产品设计**不清理**同 tag 镜像（`refreshImageReferences` 仅自动命令发布执行）。日志先 `准备全部镜像`，再停容器，符合「导入只准备、部署才替换」。

---

## 未能运行

无。Docker 可用，三组场景均完成硬断言。外部仓库未使用。

## 清理

测试结束后删除隔离数据目录、命令/Compose 容器、测试镜像标签与 `docker logout 127.0.0.1:18080`。原始 HTTP/inspect 见 `artifacts/evidence/scenario/`。
