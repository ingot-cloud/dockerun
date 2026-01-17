# 代码编辑器功能说明

## 功能概述

为部署脚本页面添加了 Monaco Editor（VS Code 使用的编辑器），提供专业的代码编辑体验。

## 新增功能

### 1. **智能语法高亮**
根据文件类型和文件名自动识别语言，支持：

- **Shell 脚本** (.sh, 脚本类型)
- **Docker Compose** (.yml, .yaml, compose 类型)
- **配置文件** (.properties, .conf, .config, .ini)
- **环境变量** (.env)
- **JSON** (.json)
- **XML** (.xml)
- **JavaScript/TypeScript** (.js, .ts)
- **Python** (.py)
- **Java** (.java)
- **Go** (.go)
- **SQL** (.sql)
- **Markdown** (.md)
- **HTML/CSS** (.html, .css)
- **Dockerfile**
- 以及更多...

### 2. **编辑器特性**

#### 主脚本编辑
- 暗色主题，与整体 UI 风格一致
- 500px 高度，带小地图（minimap）
- 自动格式化
- 代码折叠
- 行号显示
- 自动换行

#### 文件内容编辑
- 对话框宽度增加至 80%，提供更大编辑空间
- 600px 高度，带小地图
- 自动识别文件语言
- 保存按钮便于确认修改

### 3. **智能语言识别**

`getFileLanguage()` 函数根据以下规则自动判断：

1. **优先级 1**：文件类型（file_type）
   - `compose` → YAML
   - `script` → Shell

2. **优先级 2**：文件扩展名
   - `.yml`/`.yaml` → YAML
   - `.sh` → Shell
   - `.json` → JSON
   - `.env` → Shell
   - 等等...

3. **默认**：纯文本（plaintext）

## 技术实现

### 依赖库
```json
{
  "@guolao/vue-monaco-editor": "^1.6.0",
  "monaco-editor": "^0.55.1"
}
```

### 新增组件

#### `CodeEditor.vue`
可复用的代码编辑器组件，支持：
- 双向绑定 (`v-model`)
- 自定义语言
- 自定义高度
- 只读模式
- 小地图开关
- 暗色主题

**Props:**
- `modelValue`: 编辑器内容
- `language`: 语言类型（默认：shell）
- `label`: 可选标签
- `height`: 高度（默认：400px）
- `readonly`: 只读模式（默认：false）
- `minimap`: 是否显示小地图（默认：false）

**使用示例:**
```vue
<CodeEditor
  v-model="form.main_script"
  language="shell"
  height="500px"
  :minimap="true"
/>
```

### 修改的文件

1. **frontend/src/components/CodeEditor.vue** (新增)
   - Monaco Editor 封装组件

2. **frontend/src/views/DeployScripts.vue**
   - 导入 `CodeEditor` 组件
   - 替换主脚本的 `<el-input>` 为 `<CodeEditor>`
   - 替换文件编辑对话框的 `<el-input>` 为 `<CodeEditor>`
   - 添加 `getFileLanguage()` 函数

3. **frontend/package.json**
   - 添加 Monaco Editor 依赖

## 使用说明

### 编辑主脚本
1. 创建或编辑部署脚本
2. 在"Main Script"区域直接编辑
3. 享受语法高亮、代码补全等功能

### 编辑关联文件
1. 在"Files"区域点击"Edit"按钮
2. 在弹出的对话框中编辑文件内容
3. 编辑器会自动根据文件类型选择合适的语言
4. 点击"Save"保存修改

### 自定义文件语言
如果需要强制指定文件语言，可以：
1. 修改文件扩展名
2. 或者在 `getFileLanguage()` 函数中添加自定义规则

## 优势

1. **专业体验**：使用 VS Code 同款编辑器
2. **智能提示**：自动语法高亮和代码补全
3. **易于阅读**：行号、代码折叠、小地图
4. **减少错误**：实时语法检查
5. **提高效率**：快捷键、多光标编辑等高级功能

## 性能优化

- Monaco Editor 使用 Web Workers 处理语法分析
- 按需加载语言支持
- 虚拟滚动处理大文件

## 未来扩展

可以考虑添加：
- 代码片段（snippets）
- 自定义主题
- Vim/Emacs 键绑定
- 差异对比（diff）视图
- 多文件标签页编辑
