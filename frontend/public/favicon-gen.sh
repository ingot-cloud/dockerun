#!/bin/bash
# 使用 ImageMagick 从 SVG 生成 ICO
# 如果没有安装 ImageMagick，可以手动上传 ICO 文件

if command -v convert &> /dev/null; then
    echo "生成 favicon.ico..."
    convert favicon.svg -define icon:auto-resize=64,48,32,16 favicon.ico
    echo "✅ favicon.ico 生成成功"
else
    echo "⚠️ ImageMagick 未安装，跳过 ICO 生成"
    echo "你可以使用在线工具将 favicon.svg 转换为 favicon.ico:"
    echo "  - https://www.favicon-generator.org/"
    echo "  - https://realfavicongenerator.net/"
fi
