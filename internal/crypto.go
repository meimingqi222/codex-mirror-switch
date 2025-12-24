package internal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// PBKDF2 参数常量.
const (
	pbkdf2Iterations = 100000 // OWASP 推荐的最小迭代次数
	pbkdf2KeyLen     = 32     // AES-256 需要 32 字节密钥
)

// 固定盐值（用于向后兼容，新版本应使用随机盐）.
var defaultSalt = []byte("codex-mirror-v1-salt")

// CryptoManager 加密管理器.
type CryptoManager struct {
	key      []byte
	password string // 保存原始密码用于向后兼容
}

// NewCryptoManager 创建新的加密管理器.
// 使用 PBKDF2 密钥派生函数，提供更强的暴力破解防护.
// 向后兼容：自动检测旧的 hex 密钥格式（64字符）并使用 SHA256.
func NewCryptoManager(password string) *CryptoManager {
	var key []byte

	// 检测是否是旧的 hex 密钥格式 (generateEncryptKey 生成的64字符hex)
	if len(password) == 64 && isHexString(password) {
		// 向后兼容：使用旧的 SHA256 方法处理 hex 密钥
		hash := sha256.Sum256([]byte(password))
		key = hash[:]
	} else {
		// 新方法：使用 PBKDF2 将密码派生为 32 字节密钥
		key = pbkdf2.Key([]byte(password), defaultSalt, pbkdf2Iterations, pbkdf2KeyLen, sha256.New)
	}

	return &CryptoManager{
		key:      key,
		password: password, // 保存原始密码用于向后兼容
	}
}

// isHexString 检查字符串是否为有效的十六进制字符串.
func isHexString(s string) bool {
	if s == "" || len(s)%2 != 0 {
		return false // 空字符串或奇数长度不是有效hex
	}
	_, err := hex.DecodeString(s)
	return err == nil
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
// 如果解密失败且 password 不是 64 字符 hex，会尝试使用旧的 SHA256 方法进行向后兼容.
func (cm *CryptoManager) Decrypt(ciphertext []byte) ([]byte, error) {
	// 首先尝试使用当前密钥解密
	plaintext, err := cm.decryptWithKey(cm.key, ciphertext)
	if err == nil {
		return plaintext, nil
	}

	// 如果当前密钥解密失败，且密码不是64字符hex，尝试旧的SHA256方法
	if len(cm.password) != 64 || !isHexString(cm.password) {
		// 使用旧的SHA256方法派生密钥
		oldKey := sha256.Sum256([]byte(cm.password))
		plaintext, err = cm.decryptWithKey(oldKey[:], ciphertext)
		if err == nil {
			return plaintext, nil
		}
	}

	// 所有方法都失败，返回最后一个错误
	return nil, err
}

// decryptWithKey 使用指定密钥解密数据.
func (cm *CryptoManager) decryptWithKey(key, ciphertext []byte) ([]byte, error) {
	// 创建AES cipher
	block, err := aes.NewCipher(key)
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
// 使用 PBKDF2 提供更强的安全性.
func DeriveKeyFromPassword(password string) string {
	key := pbkdf2.Key([]byte(password), defaultSalt, pbkdf2Iterations, pbkdf2KeyLen, sha256.New)
	return hex.EncodeToString(key)
}
