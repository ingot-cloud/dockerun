# 从 npm 迁移到 pnpm 指南

本指南帮助你将项目从 npm 迁移到 pnpm。

## 为什么迁移到 pnpm？

### 优势

1. **⚡ 速度更快**
   - 安装速度比 npm 快 2-3 倍
   - 使用硬链接和符号链接节省磁盘空间

2. **💾 节省空间**
   - 全局存储避免重复下载
   - 多个项目共享相同版本的包

3. **🔒 更安全**
   - 严格的依赖解析
   - 避免幽灵依赖（phantom dependencies）

4. **📦 更好的 Monorepo 支持**
   - 原生支持工作区（workspace）
   - 更好的依赖管理

### 性能对比

```
操作          npm 7.x    pnpm 8.x   提升
安装时间      45s        15s        3x
磁盘空间      250MB      100MB      2.5x
缓存清理      手动       自动        -
```

## 安装 pnpm

### 方法一：使用 npm（推荐）

```bash
npm install -g pnpm
```

### 方法二：使用独立脚本

```bash
# macOS/Linux
curl -fsSL https://get.pnpm.io/install.sh | sh -

# Windows (PowerShell)
iwr https://get.pnpm.io/install.ps1 -useb | iex
```

### 方法三：使用 Homebrew (macOS)

```bash
brew install pnpm
```

### 验证安装

```bash
pnpm --version
# 应该输出 8.x.x 或更高版本
```

## 迁移步骤

### 1. 删除旧的依赖文件

```bash
# 删除 node_modules 和 package-lock.json
rm -rf node_modules package-lock.json

# 如果有 yarn.lock，也删除
rm -f yarn.lock
```

### 2. 安装依赖

```bash
# 使用 pnpm 安装
pnpm install
```

这会创建 `pnpm-lock.yaml` 文件，这是 pnpm 的锁文件。

### 3. 更新 .gitignore

确保 `.gitignore` 包含：

```gitignore
node_modules/
package-lock.json
yarn.lock
```

并添加 `pnpm-lock.yaml` 到版本控制：

```bash
git add pnpm-lock.yaml
```

### 4. 更新 CI/CD 配置

#### GitHub Actions

```yaml
# .github/workflows/ci.yml
- name: Setup Node
  uses: actions/setup-node@v4
  with:
    node-version: '20'

- name: Install pnpm
  uses: pnpm/action-setup@v2
  with:
    version: 8

- name: Install dependencies
  run: pnpm install --frozen-lockfile
```

#### GitLab CI

```yaml
# .gitlab-ci.yml
before_script:
  - npm install -g pnpm
  - pnpm install --frozen-lockfile
```

## 命令对照表

| npm 命令 | pnpm 命令 | 说明 |
|---------|----------|------|
| `npm install` | `pnpm install` | 安装所有依赖 |
| `npm install <pkg>` | `pnpm add <pkg>` | 添加依赖 |
| `npm install -D <pkg>` | `pnpm add -D <pkg>` | 添加开发依赖 |
| `npm uninstall <pkg>` | `pnpm remove <pkg>` | 删除依赖 |
| `npm update` | `pnpm update` | 更新依赖 |
| `npm run <script>` | `pnpm run <script>` 或 `pnpm <script>` | 运行脚本 |
| `npm list` | `pnpm list` | 查看依赖树 |
| `npm outdated` | `pnpm outdated` | 查看过期依赖 |
| `npx <cmd>` | `pnpm dlx <cmd>` | 执行包命令 |
| `npm cache clean` | `pnpm store prune` | 清理缓存 |

## 常用 pnpm 命令

### 基本操作

```bash
# 安装依赖
pnpm install

# 添加依赖
pnpm add vue
pnpm add -D typescript
pnpm add -g pm2

# 删除依赖
pnpm remove vue

# 更新依赖
pnpm update
pnpm update vue
pnpm update -i  # 交互式更新

# 运行脚本
pnpm dev
pnpm build
pnpm run dev  # 等同于 pnpm dev
```

### 查看信息

```bash
# 查看依赖树
pnpm list
pnpm list --depth=1

# 查看过期依赖
pnpm outdated

# 查看依赖的原因
pnpm why vue

# 查看全局安装的包
pnpm list -g
```

### 缓存管理

```bash
# 查看存储路径
pnpm store path

# 查看存储状态
pnpm store status

# 清理不使用的包
pnpm store prune
```

## 配置文件

### .npmrc

在项目根目录创建 `.npmrc` 文件：

```ini
# 使用严格的包管理器
engine-strict=true

# 自动安装 peer 依赖
auto-install-peers=true

# 使用国内镜像（可选）
# registry=https://registry.npmmirror.com/

# 提升幽灵依赖（根据需要）
shamefully-hoist=false
```

### package.json 配置

添加 engines 和 packageManager 字段：

```json
{
  "engines": {
    "node": ">=20.0.0",
    "pnpm": ">=8.0.0"
  },
  "packageManager": "pnpm@8.15.0"
}
```

## 常见问题

### 1. pnpm 找不到模块

**问题**：某些依赖无法找到

**解决方案**：
```bash
# 方法1：启用 shamefully-hoist
echo "shamefully-hoist=true" >> .npmrc

# 方法2：明确安装缺失的依赖
pnpm add <missing-package>
```

### 2. peer 依赖警告

**问题**：显示 peer 依赖警告

**解决方案**：
```bash
# 自动安装 peer 依赖
echo "auto-install-peers=true" >> .npmrc
pnpm install
```

### 3. 全局命令不可用

**问题**：全局安装的包命令找不到

**解决方案**：
```bash
# 查看 pnpm 全局 bin 路径
pnpm root -g

# 将路径添加到 PATH（添加到 ~/.bashrc 或 ~/.zshrc）
export PATH="$(pnpm root -g)/bin:$PATH"
```

### 4. CI/CD 构建失败

**问题**：CI 环境构建失败

**解决方案**：
```bash
# 使用 --frozen-lockfile 确保依赖一致性
pnpm install --frozen-lockfile

# 或在没有 lockfile 时自动生成
pnpm install --no-frozen-lockfile
```

### 5. 安装速度仍然慢

**问题**：首次安装仍然慢

**解决方案**：
```bash
# 使用国内镜像
pnpm config set registry https://registry.npmmirror.com/

# 或只对当前项目生效
echo "registry=https://registry.npmmirror.com/" >> .npmrc
```

## 进阶技巧

### 1. 使用别名

```bash
# 在 ~/.bashrc 或 ~/.zshrc 中添加
alias pi="pnpm install"
alias pa="pnpm add"
alias pr="pnpm remove"
alias pu="pnpm update"
alias pd="pnpm dev"
alias pb="pnpm build"
```

### 2. 过滤依赖安装

```bash
# 只安装生产依赖
pnpm install --prod

# 只安装开发依赖
pnpm install --dev
```

### 3. 导出和导入

```bash
# 导出依赖列表
pnpm list --json > dependencies.json

# 查看包大小
pnpm why <package-name>
```

### 4. 执行本地脚本

```bash
# 直接运行 node_modules/.bin 中的命令
pnpm exec vite
pnpm exec tsc --noEmit

# 或使用 dlx 临时安装并执行
pnpm dlx create-vite
```

## Workspace (Monorepo)

如果你有 Monorepo 需求：

### pnpm-workspace.yaml

```yaml
packages:
  - 'packages/*'
  - 'apps/*'
```

### 跨包依赖

```bash
# 在 workspace 根目录
pnpm add vue -w  # 添加到根 workspace

# 为特定包添加依赖
pnpm add vue --filter @my-org/my-package
```

## 回滚到 npm

如果需要回退：

```bash
# 1. 删除 pnpm 相关文件
rm -rf node_modules pnpm-lock.yaml .npmrc

# 2. 使用 npm 重新安装
npm install
```

## 性能监控

```bash
# 查看安装时间
time pnpm install

# 查看磁盘使用
du -sh node_modules

# 查看存储使用
pnpm store status
```

## 最佳实践

1. **提交 lockfile** - 确保 `pnpm-lock.yaml` 在版本控制中
2. **使用 --frozen-lockfile** - CI/CD 中使用此标志
3. **定期清理** - 定期运行 `pnpm store prune`
4. **配置镜像** - 使用合适的镜像源加速下载
5. **版本约束** - 在 package.json 中设置 engines 字段

## 资源链接

- [pnpm 官方文档](https://pnpm.io/)
- [pnpm vs npm vs yarn](https://pnpm.io/benchmarks)
- [pnpm CLI](https://pnpm.io/cli/add)
- [pnpm Workspace](https://pnpm.io/workspaces)

## 总结

迁移到 pnpm 的好处：
- ✅ 安装速度提升 2-3 倍
- ✅ 磁盘空间节省 50%+
- ✅ 更严格的依赖管理
- ✅ 更好的 Monorepo 支持
- ✅ 与 npm 命令高度兼容

现在你已经成功迁移到 pnpm！享受更快的开发体验吧！🚀

