# 跨项目接入

## 交付内容与使用方式

整个 `compact-dark-console/` 目录就是可迁移的技能：`SKILL.md` 为入口，`references/` 为可直接阅读的规范，`assets/console-tokens.css` 为可选主题变量，`agents/openai.yaml` 为技能显示信息。复制时保持目录结构，无需复制 Dockerun 项目。

在其他项目中可直接提出：

```text
使用 $compact-dark-console，为这个项目实现资源列表、详情和编辑页。
沿用项目现有技术栈，采用紧凑深色工作台的视觉与交互。
```

也可只采用部分内容，例如「使用 $compact-dark-console 调整这个编辑页的布局与草稿保护」，不要由局部任务推导为全项目改版。

在本机跨项目复用时，将目录放入 Codex 用户技能目录：`$CODEX_HOME/skills/compact-dark-console/`，未设置 `CODEX_HOME` 时使用 `~/.codex/skills/compact-dark-console/`。安装后在新会话中检查技能是否可用；若已有同名目录，先比较内容，避免覆盖自己的修改。

跟随某个项目分发时，将目录放入该项目的 `.agents/skills/compact-dark-console/`，并在项目 `AGENTS.md` 声明：

```markdown
本项目管理后台采用紧凑深色工作台风格。
涉及前端页面、组件、样式或交互时，读取并应用
[紧凑深色工作台规范](.agents/skills/compact-dark-console/SKILL.md)。
保留本项目的业务语义、权限、技术栈及明确的产品要求。
```

其他开发工具也可直接读取 `SKILL.md` 和对应参考文件作为规范；CSS 是标准自定义属性，不依赖某个组件库。引入主题变量不会自动实现完整组件样式。

## 接入顺序与映射

先定位目标项目现有主题和一类代表性页面，在本次需求范围内完成主题与交互映射，再将确有重复的结构提取为公共组件。没有得到全站改版任务时，只调整相关页面。

| Dockerun 中的实现 | 迁移后保留的契约 | 目标项目的接入位置 |
| --- | --- | --- |
| 领域 CSS 中的颜色、尺寸 | 深灰层级、浅色主按钮、紧凑密度 | CSS 变量、主题对象、样式工具或现有设计令牌 |
| `WorkspaceEditor` | 固定顶部操作、限宽居中、内容独立滚动 | 现有页面布局、插槽或组合组件 |
| `WorkspaceHeading` 与导航工具 | 站内来源后退、父级回退、合法上下文 | 现有路由与面包屑 |
| 领域列表工具函数 | 查询顺序、页码修正、稳定选择 | 现有查询状态、数据缓存与分页接口 |
| `useUnsavedChanges` 与 `FormDialog` | 草稿基线、离开确认、提交保护 | 现有表单状态、导航守卫与弹层关闭入口 |
| 领域删除弹窗 | 固定目标、逐项结果、只重试失败项 | 现有操作服务与结果反馈组件 |
| 请求层错误策略 | 就地错误、单次通知、取消与旧响应保护 | 现有请求客户端与异步状态管理 |

这些名称只用于解释来源，不要求目标项目使用同名文件或实现。Vue、React、Svelte 或服务端模板均按相同契约适配；优先复用目标项目已具备的能力。

## CSS 最小接入示例

将 [console-tokens.css](../assets/console-tokens.css)复制到目标项目自己的样式目录，并由现有入口加载。在采用该风格的根节点设置属性，再映射到组件：

```html
<section class="orders-workspace" data-console-theme="dark">
  <!-- 使用项目现有页面、表单、表格与按钮组件 -->
</section>
```

```css
.orders-workspace {
  background: var(--cdc-bg-page);
  color: var(--cdc-text-primary);
  font-family: var(--cdc-font-sans);
  font-size: var(--cdc-font-body);
  line-height: var(--cdc-line-body);
  padding: var(--cdc-page-block) var(--cdc-page-inline);
}
.orders-workspace .orders-card {
  background: var(--cdc-bg-surface);
  border: 1px solid var(--cdc-border);
  border-radius: var(--cdc-radius-card);
}
.orders-workspace .action-primary {
  background: var(--cdc-action-bg);
  border-color: var(--cdc-action-border);
  color: var(--cdc-action-text);
}
.orders-workspace .action-primary:not(:disabled):not([aria-disabled="true"]):hover {
  background: var(--cdc-action-hover-bg);
  border-color: var(--cdc-action-hover-bg);
  color: var(--cdc-action-hover-text);
}
```

示例只说明映射关系。按钮的真实禁用态、加载态、焦点与组件库优先级仍需接入。输入边框、链接、危险按钮和状态反馈使用对应语义变量；不要把 `--cdc-focus` 同时当作所有主按钮背景。

根部浮层在自身根节点同样设置 `data-console-theme="dark"`，或通过组件库的主题提供器与浮层容器传递变量。仅给工作区设置属性不会让挂载到其外部的弹窗自动继承。不要为解决浮层主题而无条件给整个网站切换深色。

## 可以适配与需要保持的内容

可以适配：品牌色、项目明确的语言、图标库、框架、组件 API、导航机制、业务术语、接口能力、断点和可访问性要求。组件尺寸可按触屏需求加大；示例尺寸为桌面基准。

保持关系：页面与卡片有灰度层次、主按钮与链接语义区分、顶部关键动作可见、列表局部滚动、查询和选择范围真实、草稿不静默丢失、刷新失败保留旧数据、部分成功逐项说明、未知状态不伪造为正常。

不移植：Docker 系统网络保护、卷引用限制、镜像摘要影响范围、发布版本保留规则、维护权限矩阵、加密与认证字段、指定 SVG 路径、端口和接口地址。这些需要目标系统自己的业务依据。

## 来源与取舍

提炼于 Dockerun 前端，核对日期为 **2026-10-24**。以下路径相对原仓库根目录，仅用于溯源，技能运行不需要访问它们。

| 原始材料 | 提炼内容 |
| --- | --- |
| `.agents/skills/dockerun-frontend/SKILL.md` 及两份参考规范 | 项目已明确的视觉与交互要求 |
| `frontend/src/views/Containers.vue`、`Networks.vue`、`Volumes.vue`、`Registry.vue`、`Builtin.vue` | 五类基准列表与交互风格 |
| `frontend/src/styles/networks.css`、`volumes.css`、`registries.css`、`builtin.css` | 色彩、密度、表格、状态、浮层与响应式 |
| `frontend/src/layouts/MainLayout.vue`、`styles/management.css`、`components/WorkspaceEditor.vue`、`WorkspaceHeading.vue` | 内容区、固定顶部与局部滚动 |
| `frontend/src/components/FormDialog.vue`、`composables/useUnsavedChanges.ts` | 草稿基线、取消关闭与离开保护 |
| `frontend/src/components/networks/NetworkDeleteDialog.vue` | 批量结果与失败项重试 |

这是共同风格与交互契约的规范化提炼，不表示原项目每个历史页面已完全一致。原仓库 `frontend/src/styles/design-system.css` 的蓝色主按钮、旧背景与圆角存在历史差异；本技能采用五类基准页面的共同风格，不混用旧主题默认值。具体实现与规范不同之处，应在目标项目按本次任务范围核对，不直接复制历史缺陷。
