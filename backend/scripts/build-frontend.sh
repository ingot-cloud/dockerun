#!/bin/bash

# 构建前端脚本

set -e

echo "🔨 Building frontend..."

# 进入前端目录
cd "$(dirname "$0")/../../frontend"

# 安装依赖（如果需要）
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    pnpm install
fi

# 构建
echo "🏗️  Building..."
pnpm run build

# 复制到后端 static 目录
echo "📋 Copying to backend static directory..."
rm -rf ../backend/static
cp -r dist ../backend/static

echo "✅ Frontend build complete!"
echo "📁 Static files are in: backend/static"

