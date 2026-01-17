# MySQL 数据库配置指南

Dockerun 支持使用 MySQL 作为外部数据库存储，替代默认的 SQLite。

## 配置方式

通过环境变量配置：

```bash
# 设置数据库类型为 MySQL
export DATABASE_TYPE=mysql

# 设置 MySQL 连接字符串
export MYSQL_DSN="user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
```

## DSN 格式说明

MySQL DSN（Data Source Name）格式：

```
username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
```

参数说明：
- `username`: MySQL 用户名
- `password`: MySQL 密码
- `host`: MySQL 服务器地址
- `port`: MySQL 端口（默认 3306）
- `database`: 数据库名称
- `charset=utf8mb4`: 字符集（推荐使用 utf8mb4）
- `parseTime=True`: 自动解析时间类型
- `loc=Local`: 时区设置

## 配置示例

### 1. 使用 Docker Compose

```yaml
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: dockerun
      MYSQL_USER: dockerun
      MYSQL_PASSWORD: dockerun123
    ports:
      - "3306:3306"
    volumes:
      - mysql-data:/var/lib/mysql

  dockerun:
    image: dockerun:latest
    environment:
      DATABASE_TYPE: mysql
      MYSQL_DSN: "dockerun:dockerun123@tcp(mysql:3306)/dockerun?charset=utf8mb4&parseTime=True&loc=Local"
    ports:
      - "8080:8080"
    depends_on:
      - mysql

volumes:
  mysql-data:
```

### 2. 使用外部 MySQL

```bash
# 配置环境变量
export DATABASE_TYPE=mysql
export MYSQL_DSN="dockerun:dockerun123@tcp(192.168.1.100:3306)/dockerun?charset=utf8mb4&parseTime=True&loc=Local"

# 启动应用
./dockerun
```

### 3. 在 Docker 运行时配置

```bash
docker run -d \
  -e DATABASE_TYPE=mysql \
  -e MYSQL_DSN="dockerun:dockerun123@tcp(mysql:3306)/dockerun?charset=utf8mb4&parseTime=True&loc=Local" \
  -p 8080:8080 \
  dockerun:latest
```

## 数据库准备

在使用 MySQL 之前，需要先创建数据库：

```sql
CREATE DATABASE dockerun CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'dockerun'@'%' IDENTIFIED BY 'dockerun123';
GRANT ALL PRIVILEGES ON dockerun.* TO 'dockerun'@'%';
FLUSH PRIVILEGES;
```

## 注意事项

1. **自动迁移**: 应用启动时会自动创建所需的表结构，无需手动创建
2. **数据迁移**: 如果从 SQLite 迁移到 MySQL，需要手动导出导入数据
3. **连接池**: GORM 会自动管理数据库连接池
4. **安全性**: 生产环境中请使用强密码，并限制数据库访问权限

## 切换回 SQLite

如果需要切换回 SQLite，只需：

```bash
# 删除或不设置 DATABASE_TYPE 环境变量
unset DATABASE_TYPE
unset MYSQL_DSN

# 或者显式设置为 sqlite
export DATABASE_TYPE=sqlite
export DATABASE_PATH=./data/dockerun.db
```

## 故障排查

### 连接失败

1. 检查 MySQL 服务是否运行
2. 验证 DSN 中的用户名、密码、主机和端口是否正确
3. 确认防火墙允许连接到 MySQL 端口
4. 检查 MySQL 用户权限

### 查看日志

应用启动时会输出使用的数据库类型：

```
Using MySQL database
```

或

```
Using SQLite database: ./data/dockerun.db
```

## 性能优化建议

1. 使用 MySQL 8.0 或更高版本
2. 配置适当的连接池大小
3. 定期优化表结构和索引
4. 配置 MySQL 慢查询日志监控性能

