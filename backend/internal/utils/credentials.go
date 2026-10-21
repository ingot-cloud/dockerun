package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"dockerun/internal/config"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

func credentialCipher() (cipher.AEAD, error) {
	b := []byte(config.CredentialsKey())
	key, err := hex.DecodeString(string(b))
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
func SealCredential(plain string) (string, error) {
	a, err := credentialCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, a.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	return "gcm:" + base64.RawStdEncoding.EncodeToString(a.Seal(nonce, nonce, []byte(plain), []byte("dockerun-registry-credentials"))), nil
}
func OpenCredential(value string) (string, error) {
	if !strings.HasPrefix(value, "gcm:") {
		return AESDecrypt(value)
	}
	a, err := credentialCipher()
	if err != nil {
		return "", err
	}
	b, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, "gcm:"))
	if err != nil || len(b) < a.NonceSize() {
		return "", fmt.Errorf("invalid encrypted credential")
	}
	p, err := a.Open(nil, b[:a.NonceSize()], b[a.NonceSize():], []byte("dockerun-registry-credentials"))
	return string(p), err
}
