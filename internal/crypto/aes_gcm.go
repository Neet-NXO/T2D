package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
)

// AESGCMCipher AES-GCM加密器
type AESGCMCipher struct {
	gcm cipher.AEAD
}

// NewAESGCMCipher 创建AES-GCM加密器
func NewAESGCMCipher(keySize int, password string) (*AESGCMCipher, error) {
	if keySize != 16 && keySize != 32 {
		return nil, errors.New("key size must be 16 or 32 bytes")
	}

	// 从密码派生密钥
	key := deriveKey(password, keySize)

	// 创建AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &AESGCMCipher{gcm: gcm}, nil
}

// Encrypt 加密数据
func (c *AESGCMCipher) Encrypt(plaintext []byte) ([]byte, error) {
	// 生成随机nonce
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// 加密数据
	ciphertext := c.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt 解密数据
func (c *AESGCMCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := c.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	// 提取nonce和密文
	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	// 解密数据
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// Overhead 返回加密开销
func (c *AESGCMCipher) Overhead() int {
	return c.gcm.NonceSize() + c.gcm.Overhead()
}

// deriveKey 从密码派生密钥
func deriveKey(password string, keySize int) []byte {
	hash := sha256.Sum256([]byte(password))
	if keySize <= 32 {
		return hash[:keySize]
	}
	// 如果需要更长的密钥，可以使用PBKDF2或其他KDF
	return hash[:32]
}
