package utils

import (
	"testing"
)

// TestAESEncryptDecrypt 测试 AES 加密解密
func TestAESEncryptDecrypt(t *testing.T) {
	testCases := []struct {
		name      string
		plainText string
	}{
		{"简单密码", "password123"},
		{"复杂密码", "P@ssw0rd!2025#Complex"},
		{"空密码", ""},
		{"中文密码", "密码123"},
		{"长密码", "ThisIsAVeryLongPasswordWithMoreThan32Characters1234567890"},
		{"特殊字符", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 加密
			encrypted, err := AESEncrypt(tc.plainText)
			if err != nil {
				t.Fatalf("加密失败: %v", err)
			}

			// 验证加密后的内容不为空（除了空字符串）
			if tc.plainText != "" && encrypted == "" {
				t.Error("加密后的内容为空")
			}

			// 验证加密后的内容不等于原文
			if tc.plainText != "" && encrypted == tc.plainText {
				t.Error("加密后的内容与原文相同")
			}

			// 解密
			decrypted, err := AESDecrypt(encrypted)
			if err != nil {
				t.Fatalf("解密失败: %v", err)
			}

			// 验证解密后的内容与原文相同
			if decrypted != tc.plainText {
				t.Errorf("解密后的内容不匹配: 期望 %q, 得到 %q", tc.plainText, decrypted)
			}

			t.Logf("原文: %q", tc.plainText)
			t.Logf("加密: %q", encrypted)
			t.Logf("解密: %q", decrypted)
		})
	}
}

// TestDecryptPassword 测试密码解密便捷函数
func TestDecryptPassword(t *testing.T) {
	password := "mySecretPassword"

	// 加密
	encrypted, err := AESEncrypt(password)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 解密
	decrypted, err := DecryptPassword(encrypted)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	// 验证
	if decrypted != password {
		t.Errorf("解密后的密码不匹配: 期望 %q, 得到 %q", password, decrypted)
	}
}

// TestAESDecryptInvalidData 测试解密无效数据
func TestAESDecryptInvalidData(t *testing.T) {
	testCases := []struct {
		name       string
		cipherText string
	}{
		{"无效 Base64", "this-is-not-base64!@#"},
		{"过短的密文", "YWJj"},
		{"空字符串", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := AESDecrypt(tc.cipherText)
			if err == nil {
				t.Error("应该返回错误，但没有")
			}
			t.Logf("预期的错误: %v", err)
		})
	}
}

// TestGetDecryptedPasswordWithPlaintext 测试兼容性：明文密码处理
func TestGetDecryptedPasswordWithPlaintext(t *testing.T) {
	// 这个测试模拟旧数据（明文密码）的处理
	// 注意：这个函数在 registry.go 中，这里只是概念测试
	
	plainPassword := "plainTextPassword"
	
	// 尝试解密明文（应该失败）
	_, err := AESDecrypt(plainPassword)
	if err == nil {
		t.Error("解密明文应该失败，但成功了")
	}
	
	// 在实际使用中，GetDecryptedPassword 函数会捕获这个错误
	// 并返回原始字符串（明文）
	t.Logf("明文密码解密失败（符合预期）: %v", err)
}

// BenchmarkAESEncrypt 性能测试：加密
func BenchmarkAESEncrypt(b *testing.B) {
	password := "mySecretPassword123"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = AESEncrypt(password)
	}
}

// BenchmarkAESDecrypt 性能测试：解密
func BenchmarkAESDecrypt(b *testing.B) {
	password := "mySecretPassword123"
	encrypted, _ := AESEncrypt(password)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = AESDecrypt(encrypted)
	}
}
