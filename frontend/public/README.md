# Public Assets

此目录包含项目的公共静态资源，会被直接复制到构建输出目录。

## Favicon

### 当前文件
- **`favicon.svg`** - SVG 格式图标（推荐，支持所有现代浏览器）
- **`favicon.ico`** - ICO 格式图标（可选，兼容老浏览器）

### 自定义图标

如果需要自定义 favicon，可以：

1. **替换 SVG**
   - 直接编辑 `favicon.svg`
   - 或使用设计工具创建新的 SVG

2. **生成 ICO**
   ```bash
   # 方式1: 使用 ImageMagick
   convert favicon.svg -define icon:auto-resize=64,48,32,16 favicon.ico
   
   # 方式2: 在线工具
   # https://www.favicon-generator.org/
   # https://realfavicongenerator.net/
   ```

3. **多尺寸图标**
   ```bash
   # 生成不同尺寸的 PNG（可选）
   convert favicon.svg -resize 16x16 favicon-16x16.png
   convert favicon.svg -resize 32x32 favicon-32x32.png
   convert favicon.svg -resize 180x180 apple-touch-icon.png
   ```

### 浏览器支持

| 格式 | 用途 | 支持 |
|------|------|------|
| SVG | 现代浏览器 | Chrome 80+, Firefox 41+, Safari 9+ |
| ICO | 传统浏览器 | IE, 老版本浏览器 |
| PNG | 移动设备 | iOS, Android |

### 当前设计

图标采用 Docker 蓝色 (#2496ED) 作为主色调，展示容器堆叠的视觉效果。

