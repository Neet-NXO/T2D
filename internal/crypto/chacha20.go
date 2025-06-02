package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"errors"

	"golang.org/x/crypto/chacha20poly1305"
)

// ChaCha20Poly1305Cipher ChaCha20-Poly1305加密器
type ChaCha20Poly1305Cipher struct {
	aead      cipher.AEAD
	isXChaCha bool
}

// NewChaCha20Poly1305Cipher 创建ChaCha20-Poly1305加密器
func NewChaCha20Poly1305Cipher(password string, useXChaCha bool) (*ChaCha20Poly1305Cipher, error) {
	// 从密码派生32字节密钥
	key := deriveKey(password, 32)

	var aead cipher.AEAD
	var err error

	if useXChaCha {
		// 使用XChaCha20-Poly1305
		aead, err = chacha20poly1305.NewX(key)
	} else {
		// 使用ChaCha20-Poly1305
		aead, err = chacha20poly1305.New(key)
	}

	if err != nil {
		return nil, err
	}

	return &ChaCha20Poly1305Cipher{
		aead:      aead,
		isXChaCha: useXChaCha,
	}, nil
}

// Encrypt 加密数据
func (c *ChaCha20Poly1305Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	// 生成随机nonce
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// 加密数据
	ciphertext := c.aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt 解密数据
func (c *ChaCha20Poly1305Cipher) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := c.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	// 提取nonce和密文
	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	// 解密数据
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// Overhead 返回加密开销
func (c *ChaCha20Poly1305Cipher) Overhead() int {
	return c.aead.NonceSize() + c.aead.Overhead()
}
