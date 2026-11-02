# 外部仓库凭据加密

本说明针对 Dockerun 保存的外部 Registry 管理凭据与专用拉取凭据。用户登录密码使用 bcrypt，Docker Registry 的令牌签名使用独立密钥，不属于同一种存储格式。

## 当前存储格式

新保存的外部仓库凭据使用 AES-GCM：每次加密生成随机 nonce，密文以 `gcm:` 标记，解密时校验认证信息。密钥默认自动生成并保存到 `${DATA_DIR}/keys/credentials.key`，容器中为 `/var/lib/dockerun/keys/credentials.key`。

可通过 `CREDENTIALS_SECRET` 显式提供密钥，值必须为 64 位十六进制字符串。一般安装保留默认生成的密钥即可；使用环境密钥时需要在完整备份之外保存原安装配置。

实现入口为 `backend/internal/utils/credentials.go` 与 `backend/internal/config/config.go`。备份和恢复必须同时保留元数据与对应密钥，仅恢复数据库不能解密仓库凭据。

## 旧凭据兼容

不带 `gcm:` 标记的历史密文通过旧 AES-CBC 逻辑读取。已有安装若自定义过 `AES_KEY`、`AES_IV`，升级和恢复时继续提供原值；重新录入并保存凭据后会使用当前 AES-GCM 格式，仅保存其他连接字段不会改写旧密文。

更新程序不会凭空找回遗失的旧密钥，也不会仅因启动而把所有历史连接重新加密。发现连接解密失败时，先核对原密钥与数据目录；仍无法恢复时，由管理员重新录入真实凭据。

## 维护注意事项

- 不把真实密钥、数据库副本和完整备份提交到版本库。
- 不直接替换凭据密钥后继续使用旧密文；需要先设计并验证数据重加密过程。
- 外部 Registry ACL 仍由该仓库管理；加密存储不会限制一个外部账号本身拥有的权限。
- Agent 使用的外部拉取凭据应在外部仓库中限制为只读，默认不下发管理连接凭据。

相关说明：[配置参考](CONFIGURATION.md#密钥与凭据)、[安全与权限](SECURITY.md)、[仓库使用](REGISTRY_GUIDE.md)。
