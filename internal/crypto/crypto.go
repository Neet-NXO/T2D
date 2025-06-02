package crypto

import (
	"errors"
	"fmt"
)

// CipherType 加密类型
type CipherType string

const (
	CipherAES256GCM         CipherType = "aes-256-gcm"
	CipherAES128GCM         CipherType = "aes-128-gcm"
	CipherChaCha20Poly1305  CipherType = "chacha20-poly1305"
	CipherXChaCha20Poly1305 CipherType = "xchacha20-poly1305"
	CipherNone              CipherType = "none"
	CipherXOR               CipherType = "xor"
)

// Cipher 加密接口
type Cipher interface {
	// Encrypt 加密数据
	Encrypt(plaintext []byte) ([]byte, error)
	// Decrypt 解密数据
	Decrypt(ciphertext []byte) ([]byte, error)
	// Overhead 返回加密开销（字节数）
	Overhead() int
}

// Config 加密配置
type Config struct {
	Type     CipherType `json:"type"`
	Password string     `json:"password,omitempty"`
}

// NewCipher 创建加密器
func NewCipher(config *Config) (Cipher, error) {
	switch config.Type {
	case CipherNone:
		return &NoneCipher{}, nil
	case CipherXOR:
		return &XORCipher{}, nil
	case CipherAES256GCM:
		if config.Password == "" {
			return nil, errors.New("password required for AES-256-GCM")
		}
		return NewAESGCMCipher(32, config.Password)
	case CipherAES128GCM:
		if config.Password == "" {
			return nil, errors.New("password required for AES-128-GCM")
		}
		return NewAESGCMCipher(16, config.Password)
	case CipherChaCha20Poly1305:
		if config.Password == "" {
			return nil, errors.New("password required for ChaCha20-Poly1305")
		}
		return NewChaCha20Poly1305Cipher(config.Password, false)
	case CipherXChaCha20Poly1305:
		if config.Password == "" {
			return nil, errors.New("password required for XChaCha20-Poly1305")
		}
		return NewChaCha20Poly1305Cipher(config.Password, true)
	default:
		return nil, fmt.Errorf("unsupported cipher type: %s", config.Type)
	}
}

// NoneCipher 无加密
type NoneCipher struct{}

func (c *NoneCipher) Encrypt(plaintext []byte) ([]byte, error) {
	return plaintext, nil
}

func (c *NoneCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	return ciphertext, nil
}

func (c *NoneCipher) Overhead() int {
	return 0
}

// XORCipher XOR加密（简单混淆）
type XORCipher struct{}

func (c *XORCipher) Encrypt(plaintext []byte) ([]byte, error) {
	result := make([]byte, len(plaintext))
	for i, b := range plaintext {
		result[i] = b ^ 0xAA // 简单XOR
	}
	return result, nil
}

func (c *XORCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	return c.Encrypt(ciphertext) // XOR是对称的
}

func (c *XORCipher) Overhead() int {
	return 0
}
