// Package crypto 提供加密工具：
//   - AES-GCM 对称加密（用于配置中心敏感字段）
//   - bcrypt 密码哈希（用于用户密码）
//
// AES-GCM 密钥从配置读取（base64 编码的 32 字节），
// 生成方式: openssl rand -base64 32
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// ── AES-GCM ──

// AESGCM 封装 AES-GCM 加解密器。
type AESGCM struct {
	gcm cipher.AEAD
}

// NewAESGCM 用 base64 编码的 32 字节密钥构造 AESGCM。
func NewAESGCM(b64Key string) (*AESGCM, error) {
	if b64Key == "" {
		return nil, errors.New("AES 密钥为空")
	}
	key, err := base64.StdEncoding.DecodeString(b64Key)
	if err != nil {
		return nil, fmt.Errorf("AES 密钥 base64 解码失败: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("AES 密钥长度必须为 32 字节，当前 %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建 AES cipher 失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}
	return &AESGCM{gcm: gcm}, nil
}

// Encrypt 加密明文，返回 base64 编码的密文（含 nonce 前缀）。
func (a *AESGCM) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, a.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成 nonce 失败: %w", err)
	}
	// nonce 拼在密文前，解密时取前 NonceSize 字节
	ciphertext := a.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密 Encrypt 产生的 base64 密文。
func (a *AESGCM) Decrypt(b64Ciphertext string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(b64Ciphertext)
	if err != nil {
		return "", fmt.Errorf("密文 base64 解码失败: %w", err)
	}
	nonceSize := a.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("密文长度不足")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := a.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}
	return string(plaintext), nil
}

// ── bcrypt 密码哈希 ──

// HashPassword 用 bcrypt 哈希密码（cost=12，兼顾安全与性能）。
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("密码哈希失败: %w", err)
	}
	return string(hash), nil
}

// CheckPassword 验证密码是否匹配哈希。
// 匹配返回 nil；不匹配返回非 nil error。
func CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
