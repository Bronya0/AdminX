package crypto

import (
	"encoding/base64"
	"testing"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("mypassword123")
	if err != nil {
		t.Fatalf("HashPassword 失败: %v", err)
	}
	if len(hash) == 0 {
		t.Fatal("哈希结果为空")
	}
}

func TestCheckPassword(t *testing.T) {
	hash, _ := HashPassword("correct123")
	if err := CheckPassword(hash, "correct123"); err != nil {
		t.Errorf("正确密码应匹配: %v", err)
	}
	if err := CheckPassword(hash, "wrongpassword"); err == nil {
		t.Error("错误密码不应匹配")
	}
}

func TestNewAESGCM(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	b64Key := base64.StdEncoding.EncodeToString(key)

	aes, err := NewAESGCM(b64Key)
	if err != nil {
		t.Fatalf("NewAESGCM 失败: %v", err)
	}
	if aes == nil {
		t.Fatal("aes 为 nil")
	}
}

func TestNewAESGCM_Invalid(t *testing.T) {
	t.Run("空密钥", func(t *testing.T) {
		_, err := NewAESGCM("")
		if err == nil {
			t.Error("空密钥应返回 error")
		}
	})
	t.Run("无效 base64", func(t *testing.T) {
		_, err := NewAESGCM("!!!not-valid!!!")
		if err == nil {
			t.Error("无效 base64 应返回 error")
		}
	})
	t.Run("长度不对", func(t *testing.T) {
		shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
		_, err := NewAESGCM(shortKey)
		if err == nil {
			t.Error("非 32 字节密钥应返回 error")
		}
	})
}

func TestAESGCM_EncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	aes, _ := NewAESGCM(base64.StdEncoding.EncodeToString(key))

	plaintext := "hello world, 你好世界"

	encrypted, err := aes.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt 失败: %v", err)
	}
	if encrypted == plaintext || len(encrypted) == 0 {
		t.Fatal("加密结果不应与原文明文相同")
	}

	decrypted, err := aes.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt 失败: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("解密结果 = %s, want %s", decrypted, plaintext)
	}
}

func TestAESGCM_Decrypt_Invalid(t *testing.T) {
	key := make([]byte, 32)
	aes, _ := NewAESGCM(base64.StdEncoding.EncodeToString(key))

	t.Run("无效 base64", func(t *testing.T) {
		_, err := aes.Decrypt("not-base64!!!")
		if err == nil {
			t.Error("无效 base64 应返回 error")
		}
	})
	t.Run("太短", func(t *testing.T) {
		_, err := aes.Decrypt(base64.StdEncoding.EncodeToString([]byte("ab")))
		if err == nil {
			t.Error("密文太短应返回 error")
		}
	})
}

func TestAESGCM_Encrypt_DifferentCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	aes, _ := NewAESGCM(base64.StdEncoding.EncodeToString(key))

	c1, _ := aes.Encrypt("same text")
	c2, _ := aes.Encrypt("same text")
	if c1 == c2 {
		t.Error("相同明文加密两次应产生不同密文（nonce 随机）")
	}
}
