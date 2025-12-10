# Dockerun 模板规范

## 目录结构

每个模板是一个独立的目录，包含以下文件：

```
template-name/
├── meta.json           # 必需：模板元数据
├── deploy.sh           # 必需：主部署脚本
└── files/              # 可选：附加文件目录
    ├── config.conf
    ├── init.sql
    └── ...
```

## meta.json 格式

```json
{
  "id": "template-id",              // 必需：唯一标识符
  "name": "Template Name",          // 必需：显示名称
  "description": "Description",     // 必需：描述
  "category": "Category",           // 必需：分类
  "icon": "🎯",                      // 必需：图标（emoji）
  "main_script": "deploy.sh",       // 必需：主脚本文件名
  "work_dir": "/opt/app",           // 必需：工作目录
  "env_vars": {                     // 可选：环境变量
    "KEY1": "value1",
    "KEY2": "value2"
  },
  "files": [                        // 可选：附加文件列表
    {
      "file_name": "config.conf",   // 文件名
      "file_path": "conf/config.conf", // 部署时的相对路径
      "source": "files/config.conf", // 模板目录中的源文件路径
      "file_type": "config",        // 文件类型：config, compose, script, other
      "description": "配置文件"      // 描述
    }
  ]
}
```

## 主脚本规范

主脚本必须：
1. 使用 `#!/usr/bin/env bash` 作为 shebang
2. 使用 `set -e` 确保错误时停止
3. 输出容器ID（最后一行或包含 "Container ID:" 的行）
4. 可以使用环境变量（从 `env_vars` 定义）
5. 可以使用 `${WORK_DIR}` 环境变量访问工作目录

示例：
```bash
#!/usr/bin/env bash
set -e

echo "Deploying application..."

# 使用环境变量
docker run -d \
  --name ${APP_NAME} \
  -p ${APP_PORT}:80 \
  -v ${WORK_DIR}/data:/data \
  nginx:latest

# 输出容器ID
CONTAINER_ID=$(docker ps -q --filter name=${APP_NAME})
echo "Container ID: $CONTAINER_ID"
```

## 创建新模板

1. 在 `templates/` 目录下创建新目录
2. 创建 `meta.json` 文件
3. 创建 `deploy.sh` 主脚本
4. （可选）创建 `files/` 目录和附加文件
5. 重启应用，新模板会自动加载

## 分类建议

- Web Server: Nginx, Apache, Caddy
- Database: MySQL, PostgreSQL, MongoDB
- Cache: Redis, Memcached
- Message Queue: RabbitMQ, Kafka
- Multi-Container: Docker Compose 应用
- Other: 其他类型

## 最佳实践

1. **命名规范**：使用小写字母和连字符，如 `nginx-php`
2. **工作目录**：使用 `/opt/app-name` 或 `/ingot-data/dockerun/app-name`
3. **端口映射**：通过环境变量配置，提供默认值
4. **数据持久化**：使用卷挂载保存重要数据
5. **容器命名**：使用有意义的名称，避免冲突
6. **错误处理**：检查现有容器并清理
7. **输出信息**：提供清晰的部署状态和访问信息

