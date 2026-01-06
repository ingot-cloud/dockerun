import CryptoJS from 'crypto-js'

// AES 加密密钥（与后端保持一致）
// 注意：生产环境应该从环境变量读取
const AES_KEY = import.meta.env.VITE_AES_KEY || 'dockerun-aes-key-2025-secret-key'
const AES_IV = import.meta.env.VITE_AES_IV || 'dockerun-aes-iv16'

/**
 * AES 加密
 * @param plainText 明文
 * @returns 加密后的密文（Base64）
 */
export function aesEncrypt(plainText: string): string {
  const key = CryptoJS.enc.Utf8.parse(AES_KEY.padEnd(32, '0').substring(0, 32))
  const iv = CryptoJS.enc.Utf8.parse(AES_IV.padEnd(16, '0').substring(0, 16))
  
  const encrypted = CryptoJS.AES.encrypt(plainText, key, {
    iv: iv,
    mode: CryptoJS.mode.CBC,
    padding: CryptoJS.pad.Pkcs7
  })
  
  return encrypted.toString()
}

/**
 * AES 解密（前端一般不需要，但提供完整性）
 * @param cipherText 密文（Base64）
 * @returns 解密后的明文
 */
export function aesDecrypt(cipherText: string): string {
  const key = CryptoJS.enc.Utf8.parse(AES_KEY.padEnd(32, '0').substring(0, 32))
  const iv = CryptoJS.enc.Utf8.parse(AES_IV.padEnd(16, '0').substring(0, 16))
  
  const decrypted = CryptoJS.AES.decrypt(cipherText, key, {
    iv: iv,
    mode: CryptoJS.mode.CBC,
    padding: CryptoJS.pad.Pkcs7
  })
  
  return decrypted.toString(CryptoJS.enc.Utf8)
}

/**
 * 加密登录密码
 * @param password 原始密码
 * @returns 加密后的密码
 */
export function encryptPassword(password: string): string {
  return aesEncrypt(password)
}

