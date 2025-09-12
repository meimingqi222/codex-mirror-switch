package internal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// CryptoManager 加密管理器.
type CryptoManager struct {
	key []byte
}

// NewCryptoManager 创建新的加密管理器.
func NewCryptoManager(password string) *CryptoManager {
	// 使用SHA256将密码转换为32字节密钥
	hash := sha256.Sum256([]byte(password))
	return &CryptoManager{
		key: hash[:],
	}
}

// Encrypt 使用AES-GCM加密数据.
func (cm *CryptoManager) Encrypt(plaintext []byte) ([]byte, error) {
	// 创建AES cipher
	block, err := aes.NewCipher(cm.key)
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM模式失败: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成nonce失败: %w", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt 使用AES-GCM解密数据.
func (cm *CryptoManager) Decrypt(ciphertext []byte) ([]byte, error) {
	// 创建AES cipher
	block, err := aes.NewCipher(cm.key)
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM模式失败: %w", err)
	}

	// 检查数据长度
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("密文长度不足")
	}

	// 提取nonce和密文
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %w", err)
	}

	return plaintext, nil
}

// GenerateKey 生成随机加密密钥.
func GenerateKey() (string, error) {
	key := make([]byte, 32) // 256位密钥
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("生成密钥失败: %w", err)
	}
	return hex.EncodeToString(key), nil
}

// DeriveKeyFromPassword 从密码派生密钥.
func DeriveKeyFromPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}