# UI 设计说明

本项目的 UI 设计完全参考 Portainer 的深色风格，提供现代化、专业的 Docker 管理界面。

## 设计理念

### 1. 深色主题 (Dark Theme)

整体采用深色主题，减少眼睛疲劳，提升长时间使用体验。

**主要颜色：**
- 主背景：`#1e1e1e`
- 卡片背景：`#2a2d31`
- 侧边栏背景：`#23252a`
- 主题色（蓝色）：`#4a9eff` / `#388bfd`
- 成功色（绿色）：`#2ecc71`
- 警告色（橙色）：`#f39c12`
- 危险色（红色）：`#e74c3c`
- 文本主色：`#ffffff`
- 文本次要色：`#8a929c`

### 2. Portainer 风格元素

- **LOGO 设计**: 大写字母 + "Ingot Cloud" 小标签
- **侧边栏**: 深色背景，图标+文字导航
- **顶部栏**: 用户信息下拉菜单
- **表格样式**: 深色表格，状态标签，快捷操作按钮
- **按钮组**: 批量操作按钮栏

## 页面布局

### 登录页面 (Login)

```
特点：
- 居中的卡片式布局
- 渐变深色背景
- 半透明卡片效果
- DOCKERUN Logo
- 简洁的表单设计
- 蓝色主题按钮
```

**设计细节：**
- 背景渐变：`linear-gradient(135deg, #1a1a2e 0%, #16213e 100%)`
- 卡片背景：`rgba(42, 42, 56, 0.95)`
- 输入框背景：`rgba(255, 255, 255, 0.08)`
- 按钮渐变：`linear-gradient(135deg, #388bfd 0%, #2c6dd5 100%)`

### 初始设置页面 (Setup)

```
特点：
- 与登录页面相似的布局
- 引导用户设置管理员密码
- 密码强度提示
- 确认密码验证
```

**首次运行流程：**
1. 检测数据库中是否有用户
2. 如果没有用户或密码为空，跳转到 Setup 页面
3. 设置密码后跳转到登录页面

### 主布局 (MainLayout)

```
布局结构：
┌────────────┬──────────────────────────┐
│  Sidebar   │      Top Header          │
│            ├──────────────────────────┤
│  - Logo    │                          │
│  - Menu    │      Main Content        │
│  - Items   │                          │
│            │                          │
│  - Footer  │                          │
└────────────┴──────────────────────────┘
```

**侧边栏 (Sidebar):**
- 宽度：200px
- 背景色：`#23252a`
- Logo 区域：`#1a1c20`
- 菜单项高度：44px
- 激活状态：左侧蓝色边框 + 蓝色背景

**顶部栏 (Header):**
- 高度：60px
- 背景色：`#2a2d31`
- 用户信息下拉菜单
- 修改密码/退出登录

**主内容区:**
- 背景色：`#1e1e1e`
- 内边距：24px

### 容器列表页面 (Containers)

```
页面结构：
1. 页面标题 + 刷新按钮
2. 搜索框 + 操作按钮
3. 内容卡片
   - 卡片标题（带图标）
   - 数据表格
     * 多选框
     * 名称（带状态图标）
     * 状态标签
     * 快捷操作（日志/统计/终端）
     * Stack
     * 镜像
     * 创建时间
     * IP地址
     * 发布端口
     * 所有权
   - 底部操作栏
     * 批量操作按钮组
```

**表格特点：**
- 深色背景，半透明
- Hover 效果：`rgba(74, 158, 255, 0.08)`
- 状态图标：
  - 运行中：绿色圆点
  - 已停止：灰色圆点
- 容器名称：蓝色链接样式
- 端口显示：蓝色链接图标 + 端口信息

**批量操作按钮：**
- Start（成功色）
- Stop（警告色）
- Restart（主题色）
- Kill（信息色）
- Pause（默认色）
- Resume（默认色）
- Remove（危险色）

## 组件样式规范

### 按钮 (Buttons)

```css
/* 主要按钮 */
Primary: #388bfd → #4a9eff (hover)

/* 成功按钮 */
Success: #2ecc71

/* 警告按钮 */
Warning: #f39c12

/* 危险按钮 */
Danger: #e74c3c

/* 信息按钮 */
Info: #5dade2

/* 默认按钮 */
Default: rgba(255, 255, 255, 0.08)
Border: rgba(255, 255, 255, 0.2)
```

### 输入框 (Inputs)

```css
背景色: rgba(255, 255, 255, 0.08)
边框: rgba(255, 255, 255, 0.15)
Hover边框: rgba(56, 139, 253, 0.5)
Focus边框: #388bfd
文本颜色: #ffffff
占位符: rgba(255, 255, 255, 0.4)
```

### 标签 (Tags)

```css
/* 运行中 */
Success: #2ecc71

/* 已停止 */
Info: #95a5a6

/* 警告 */
Warning: #f39c12

/* 错误 */
Danger: #e74c3c
```

### 表格 (Tables)

```css
表头背景: #23252a
表头文字: #8a929c
行背景: transparent
Hover背景: rgba(74, 158, 255, 0.08)
边框: rgba(255, 255, 255, 0.05)
```

### 卡片 (Cards)

```css
背景色: #2a2d31
边框: 1px solid rgba(255, 255, 255, 0.1)
圆角: 4px
内边距: 16px 20px
```

### 对话框 (Dialogs)

```css
背景色: #2a2d31
边框: 1px solid rgba(255, 255, 255, 0.1)
标题边框: 1px solid rgba(255, 255, 255, 0.1)
```

## 图标使用

使用 Element Plus Icons：

- `Box` - 容器
- `Picture` - 镜像
- `Connection` - 网络
- `FolderOpened` - 卷
- `Document` - 文档/日志
- `Setting` - 设置
- `User` - 用户
- `Refresh` - 刷新
- `Search` - 搜索
- `CircleFilled` - 运行状态
- `CircleClose` - 停止状态
- `DataAnalysis` - 统计
- `Monitor` - 终端
- `Link` - 链接/端口
- `VideoPlay` - 启动/播放
- `VideoPause` - 暂停
- `Delete` - 删除

## 交互效果

### 1. Hover 效果

```css
/* 按钮 */
transform: translateY(-1px)
box-shadow: 0 4px 12px rgba(56, 139, 253, 0.4)

/* 侧边栏菜单项 */
background-color: rgba(255, 255, 255, 0.05)
color: #4a9eff

/* 表格行 */
background-color: rgba(74, 158, 255, 0.08)
```

### 2. Focus 效果

```css
/* 输入框 */
border-color: #388bfd
box-shadow: 0 0 0 2px rgba(56, 139, 253, 0.15)
```

### 3. Active 效果

```css
/* 侧边栏激活项 */
background-color: rgba(74, 158, 255, 0.1)
border-left: 3px solid #4a9eff
color: #4a9eff
```

### 4. 过渡动画

```css
transition: all 0.3s ease
```

## 响应式设计

### 断点

- Desktop: > 1200px
- Tablet: 768px - 1200px
- Mobile: < 768px

### 适配策略

1. 侧边栏在移动端折叠
2. 表格在小屏幕上横向滚动
3. 按钮组在小屏幕上垂直排列
4. 对话框宽度根据屏幕自适应

## 滚动条样式

```css
宽度: 8px
轨道背景: rgba(255, 255, 255, 0.05)
滑块背景: rgba(255, 255, 255, 0.2)
Hover滑块: rgba(255, 255, 255, 0.3)
圆角: 4px
```

## 字体规范

### 字体家族

```css
font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 
             Roboto, 'Helvetica Neue', Arial, sans-serif
```

### 字号规范

- 标题 (H1): 24px - 32px
- 副标题 (H2): 18px - 24px
- 正文: 14px
- 小字: 12px - 13px
- 极小字: 10px - 11px

### 字重

- Light: 300
- Normal: 400
- Medium: 500
- Semi-bold: 600
- Bold: 700

## 阴影效果

```css
/* 卡片阴影 */
box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4)

/* 按钮阴影 */
box-shadow: 0 4px 12px rgba(56, 139, 253, 0.4)

/* 下拉菜单阴影 */
box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3)
```

## 代码字体

用于日志、代码显示：

```css
font-family: 'Consolas', 'Monaco', 'Courier New', monospace
font-size: 12px
```

## 可访问性

1. **颜色对比度**: 确保文本与背景对比度 > 4.5:1
2. **焦点状态**: 所有交互元素都有明显的焦点样式
3. **键盘导航**: 支持 Tab 键导航
4. **屏幕阅读器**: 为图标和按钮添加 aria-label

## 性能优化

1. **CSS 过渡**: 使用 transform 而非 position
2. **图片懒加载**: 大图片使用懒加载
3. **虚拟滚动**: 长列表使用虚拟滚动
4. **防抖节流**: 搜索、滚动事件使用防抖节流

## 浏览器兼容性

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## 开发建议

1. **组件复用**: 提取通用组件（按钮、卡片、表格等）
2. **CSS 变量**: 使用 CSS 变量定义主题色
3. **BEM 命名**: 遵循 BEM 命名规范
4. **Scoped 样式**: 组件样式使用 scoped
5. **深度选择器**: 使用 :deep() 修改 Element Plus 样式

## 设计资源

参考 Portainer 官方界面：
- 登录页：简洁、专业
- 主界面：深色、现代
- 表格：信息密集、易读
- 按钮：清晰、易识别

## 更新日志

- 2025-12-08: 完成 Portainer 风格的深色主题设计
- 2025-12-08: 实现登录、设置、主布局、容器列表页面
- 2025-12-08: 添加全局深色主题样式

