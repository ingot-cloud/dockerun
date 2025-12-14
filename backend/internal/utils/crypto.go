package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"os"
)

// GetAESKey 获取 AES 密钥
func GetAESKey() string {
	key := os.Getenv("AES_KEY")
	if key == "" {
		key = "dockerun-aes-key-2024-secret-key"
	}
	// 确保密钥长度为32字节（AES-256）
	return padKey(key, 32)
}

// GetAESIV 获取 AES 初始化向量
func GetAESIV() string {
	iv := os.Getenv("AES_IV")
	if iv == "" {
		iv = "dockerun-aes-iv16"
	}
	// 确保 IV 长度为16字节
	return padKey(iv, 16)
}

// padKey 填充密钥到指定长度
func padKey(key string, length int) string {
	keyBytes := []byte(key)
	if len(keyBytes) >= length {
		return string(keyBytes[:length])
	}
	// 用 '0' 填充到指定长度
	for len(keyBytes) < length {
		keyBytes = append(keyBytes, '0')
	}
	return string(keyBytes)
}

// AESDecrypt AES 解密（CBC 模式）
func AESDecrypt(cipherText string) (string, error) {
	// Base64 解码
	cipherData, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	key := []byte(GetAESKey())
	iv := []byte(GetAESIV())

	// 创建 AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(cipherData) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	// CBC 模式解密
	mode := cipher.NewCBCDecrypter(block, iv)
	plainData := make([]byte, len(cipherData))
	mode.CryptBlocks(plainData, cipherData)

	// 去除 PKCS7 填充
	plainData, err = pkcs7Unpad(plainData)
	if err != nil {
		return "", err
	}

	return string(plainData), nil
}

// AESEncrypt AES 加密（CBC 模式）- 用于测试
func AESEncrypt(plainText string) (string, error) {
	key := []byte(GetAESKey())
	iv := []byte(GetAESIV())

	plainData := []byte(plainText)

	// PKCS7 填充
	plainData = pkcs7Pad(plainData, aes.BlockSize)

	// 创建 AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// CBC 模式加密
	cipherData := make([]byte, len(plainData))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherData, plainData)

	// Base64 编码
	return base64.StdEncoding.EncodeToString(cipherData), nil
}

// pkcs7Pad PKCS7 填充
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// pkcs7Unpad PKCS7 去填充
func pkcs7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("data is empty")
	}
	padding := int(data[length-1])
	if padding > length {
		return nil, errors.New("invalid padding")
	}
	return data[:length-padding], nil
}

// DecryptPassword 解密密码（便捷函数）
func DecryptPassword(encryptedPassword string) (string, error) {
	return AESDecrypt(encryptedPassword)
}

