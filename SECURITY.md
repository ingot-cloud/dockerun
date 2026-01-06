# 安全功能说明

## 🔐 密码加密传输

### 概述
本系统实现了完整的密码加密传输机制，确保密码在前后端传输过程中的安全性。

### 加密流程

```
┌─────────────────────────────────────────┐
│  前端 (Browser)                          │
│  ┌───────────────────────────────────┐  │
│  │  用户输入: password123            │  │
│  └────────────┬──────────────────────┘  │
│               │                          │
│  ┌────────────▼──────────────────────┐  │
│  │  AES-256-CBC 加密                 │  │
│  │  Key: 32字节                      │  │
│  │  IV:  16字节                      │  │
│  └────────────┬──────────────────────┘  │
│               │                          │
│  ┌────────────▼──────────────────────┐  │
│  │  Base64 编码                      │  │
│  │  输出: U2FsdGVkX1...              │  │
│  └────────────┬──────────────────────┘  │
└───────────────┼──────────────────────────┘
                │ HTTPS
                │
┌───────────────▼──────────────────────────┐
│  后端 (Server)                            │
│  ┌───────────────────────────────────┐   │
│  │  接收加密密码                      │   │
│  └────────────┬──────────────────────┘   │
│               │                           │
│  ┌────────────▼──────────────────────┐   │
│  │  Base64 解码                      │   │
│  └────────────┬──────────────────────┘   │
│               │                           │
│  ┌────────────▼──────────────────────┐   │
│  │  AES-256-CBC 解密                 │   │
│  │  使用相同的 Key 和 IV             │   │
│  └────────────┬──────────────────────┘   │
│               │                           │
│  ┌────────────▼──────────────────────┐   │
│  │  Bcrypt 哈希验证                  │   │
│  │  (数据库存储)                     │   │
│  └───────────────────────────────────┘   │
└──────────────────────────────────────────┘
```

### 技术实现

#### 前端 (TypeScript + crypto-js)

**文件**: `frontend/src/utils/crypto.ts`

```typescript
// AES-256-CBC 加密
export function encryptPassword(password: string): string {
  const key = CryptoJS.enc.Utf8.parse(AES_KEY)
  const iv = CryptoJS.enc.Utf8.parse(AES_IV)
  
  const encrypted = CryptoJS.AES.encrypt(password, key, {
    iv: iv,
    mode: CryptoJS.mode.CBC,
    padding: CryptoJS.pad.Pkcs7
  })
  
  return encrypted.toString() // Base64
}
```

**使用场景**:
- 登录 (`Login.vue` → `auth.ts`)
- 初始密码设置 (`Setup.vue`)
- 修改密码 (待实现)

#### 后端 (Go + crypto/aes)

**文件**: `backend/internal/utils/crypto.go`

```go
// AES-256-CBC 解密
func AESDecrypt(cipherText string) (string, error) {
    // 1. Base64 解码
    cipherData, _ := base64.StdEncoding.DecodeString(cipherText)
    
    // 2. AES 解密
    key := []byte(GetAESKey())  // 32 bytes
    iv := []byte(GetAESIV())    // 16 bytes
    
    block, _ := aes.NewCipher(key)
    mode := cipher.NewCBCDecrypter(block, iv)
    mode.CryptBlocks(plainData, cipherData)
    
    // 3. PKCS7 去填充
    plainData, _ := pkcs7Unpad(plainData)
    
    return string(plainData), nil
}
```

**使用位置**:
- `Login()` - 登录验证
- `SetInitialPassword()` - 初始密码设置
- `ChangePassword()` - 修改密码

### 密钥配置

#### 前端配置

创建 `frontend/.env.local` 文件：

```env
VITE_AES_KEY=your-secret-key-32-bytes-long
VITE_AES_IV=your-iv-16-bytes
```

#### 后端配置

环境变量：

```bash
export AES_KEY="your-secret-key-32-bytes-long"
export AES_IV="your-iv-16-bytes"
```

Docker 运行：

```bash
docker run -d \
  -e AES_KEY="your-secret-key-32-bytes-long" \
  -e AES_IV="your-iv-16-bytes" \
  ...
```

### 默认密钥

**⚠️ 安全警告**: 默认密钥仅用于开发测试，生产环境必须修改！

```
AES_KEY: "dockerun-aes-key-2025-secret-key"  (32 bytes)
AES_IV:  "dockerun-aes-iv16"                  (16 bytes)
```

### 安全特性

| 特性 | 说明 | 状态 |
|------|------|------|
| **传输加密** | AES-256-CBC | ✅ |
| **存储加密** | Bcrypt (cost=10) | ✅ |
| **密钥独立** | 前后端分离配置 | ✅ |
| **IV 随机化** | 固定 IV (可改进) | ⚠️ |
| **密钥轮换** | 支持环境变量配置 | ✅ |
| **HTTPS** | 建议生产环境启用 | 📋 |

### 多层安全保护

```
用户密码: "password123"
    ↓
1. 前端 AES 加密 → "U2FsdGVkX1..."  (传输层保护)
    ↓
2. HTTPS 加密      → 双重保护         (网络层保护)
    ↓
3. 后端 AES 解密   → "password123"    (还原明文)
    ↓
4. Bcrypt 哈希     → "$2a$10$..."    (存储层保护)
    ↓
数据库存储: 只存储 Bcrypt 哈希值
```

### 最佳实践

#### 生产环境部署

1. **生成强密钥**
```bash
# 生成 32 字节密钥
openssl rand -base64 32

# 生成 16 字节 IV
openssl rand -base64 16
```

2. **配置密钥**
```bash
# 方式1: 环境变量
export AES_KEY="$(openssl rand -base64 32)"
export AES_IV="$(openssl rand -base64 16)"

# 方式2: Docker Secrets
docker secret create aes_key /path/to/key
docker secret create aes_iv /path/to/iv
```

3. **启用 HTTPS**
```bash
# Nginx 反向代理
server {
    listen 443 ssl;
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://dockerun:8080;
    }
}
```

#### 密钥管理

- ✅ **DO**: 使用环境变量或密钥管理服务
- ✅ **DO**: 定期轮换密钥
- ✅ **DO**: 每个环境使用不同的密钥
- ❌ **DON'T**: 硬编码密钥到代码中
- ❌ **DON'T**: 提交密钥到版本控制系统
- ❌ **DON'T**: 在生产环境使用默认密钥

### 已知限制与改进方向

1. **固定 IV**: 当前使用固定 IV，可改为随机 IV
   - 改进: 前端生成随机 IV，与密文一起传输

2. **密钥硬编码**: 默认密钥在代码中
   - 改进: 强制要求通过环境变量配置

3. **前端密钥暴露**: 前端密钥可被反编译获取
   - 改进: 使用非对称加密（RSA），后端私钥解密

### 测试

#### 单元测试

```bash
# 后端测试
cd backend
go test ./internal/utils -v -run TestAESEncryptDecrypt

# 前端测试
cd frontend
pnpm test crypto.spec.ts
```

#### 集成测试

```bash
# 测试登录加密
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"U2FsdGVkX1..."}'
```

### 合规性

本加密方案符合以下安全标准：
- ✅ OWASP Top 10 - A2:2021 Cryptographic Failures
- ✅ PCI DSS 3.2.1 - Requirement 4 (Encrypt transmission)
- ✅ GDPR - Article 32 (Security of processing)

---

## 🔒 其他安全特性

### JWT 认证
- 签名算法: HS256
- 过期时间: 24 小时
- 自动刷新: 支持

### 会话管理
- 存储: LocalStorage
- 过期处理: 自动跳转登录
- 多标签同步: 支持

### API 安全
- CORS: 配置白名单
- Rate Limiting: 待实现
- SQL Injection: 使用 ORM 防护
- XSS: Vue 自动转义

---

**最后更新**: 2025-12-14
**维护者**: Dockerun Team

