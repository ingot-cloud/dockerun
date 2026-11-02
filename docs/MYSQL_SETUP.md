# MySQL 元数据库配置

Dockerun 默认使用 SQLite，也可将应用、用户、版本、任务等元数据存入专用 MySQL 数据库。内置仓库镜像、密钥、上传文件和发布材料仍保存在 `DATA_DIR`，切换到 MySQL 后仍须持久化 `/var/lib/dockerun`。

本文配置的是 Dockerun 自身数据库，与应用模板发布的业务 MySQL 服务分别管理。已有验证记录覆盖 MySQL 8.4 的迁移及参数读写，完整备份恢复的专门环境演练仍需完成，见[验证记录](VALIDATION.md#mysql)。

## 准备数据库

先在 MySQL 创建专用数据库和账号。以下密码及账号来源仅为示例，部署时替换实际值，按数据库可见的来源地址限制账号访问：

```sql
CREATE DATABASE dockerun CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'dockerun'@'10.0.0.%' IDENTIFIED BY '替换为实际强密码';
GRANT ALL PRIVILEGES ON dockerun.* TO 'dockerun'@'10.0.0.%';
```

服务启动会创建或迁移表结构，账号需要相应权限。根目录 Compose 不会自动创建 MySQL，也不会将数据库放入同一个容器；连接前保证数据库已就绪、网络可达。

## 容器部署配置

编辑安装目录 `.env`，保留实际 `PUBLIC_URL` 和其他安装配置：

```dotenv
DATABASE_TYPE=mysql
MYSQL_DSN='dockerun:替换为实际密码@tcp(mysql.example.com:3306)/dockerun?charset=utf8mb4&parseTime=True&loc=Local'
```

DSN 包含用户、密码、连接地址和数据库名称；`parseTime=True` 用于时间值解析，`charset=utf8mb4` 用于中文等字符。单引号可避免 Compose 对值内 `$` 进行插值，特殊字符还需满足 DSN 本身的格式要求。不要在共享日志中打印解析后的完整配置。

容器中的 `localhost` 指向 Dockerun 容器本身，不是宿主 MySQL；使用可达的 DNS、宿主地址或同一容器网络内的服务名。

```bash
docker compose config --quiet
docker compose up -d --no-build --pull never
docker compose logs --tail=100 dockerun
```

启动日志中的 `Using MySQL database` 表示选择了 MySQL，仍需确认没有连接或迁移错误。修改 `.env` 后要重新创建服务，单纯 `docker compose restart` 不加载新环境。

本地开发时，先通过当前进程环境提供 `MYSQL_DSN`，再运行：

```bash
make dev-backend DATABASE_TYPE=mysql
```

## 数据迁移与恢复

切换 `DATABASE_TYPE` 不会迁移另一种数据库的数据。将空 MySQL 接入原 SQLite 安装会得到另一套元数据，不能作为原用户、应用和版本已经迁移的依据。迁移需另行准备数据转换及核对，先保留原完整备份。

完整备份会调用 `mysqldump --single-transaction` 生成 `metadata.sql`，并归档密钥、内置仓库、发布和任务材料。交付镜像包含相关客户端工具；直接运行二进制时要在运行环境准备 `mysqldump`。备份失败时核对数据库权限、客户端与服务端兼容性及日志。

恢复先按[部署指南](DEPLOYMENT.md#备份与恢复)停止控制台并将完整包解到新空目录。MySQL 备份解包后，CLI 会返回需要手动恢复数据库的提示及非零退出码。随后将 SQL 导入新的专用数据库，例如：

```bash
mysql --host mysql.example.com --user dockerun --password dockerun_restored < /var/lib/dockerun-restored/metadata.sql
```

示例数据库 `dockerun_restored` 需预先创建并授权，命令交互输入密码。核对导入结果后，将 `MYSQL_DSN` 指向该数据库，恢复原安装的环境密钥配置，切换文件数据目录后启动。不要把 SQL 已解出当成数据库已经恢复。

## 连接失败排查

检查日志中的实际错误，依次核对数据库服务、DNS / 端口、账号来源、密码和数据库名，以及建表迁移权限。服务启动时不会自动回退到另一数据库。更多变量见[配置参考](CONFIGURATION.md)，常见问题见[故障排查](TROUBLESHOOTING.md)。
