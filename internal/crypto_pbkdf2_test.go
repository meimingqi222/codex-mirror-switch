package internal

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

// TestPBKDF2Compatibility 测试 PBKDF2 向后兼容性
func TestPBKDF2Compatibility(t *testing.T) {
	password := "test-password-123"

	// 创建两个 CryptoManager 实例
	cm1 := NewCryptoManager(password)
	cm2 := NewCryptoManager(password)

	// 测试数据
	plaintext := []byte("sensitive data")

	// 使用第一个实例加密
	encrypted, err := cm1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 使用第二个实例解密
	decrypted, err := cm2.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	// 验证解密结果
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("解密后数据不匹配\n期望: %s\n得到: %s", plaintext, decrypted)
	}
}

// TestPBKDF2DeterministicKey 测试相同密码生成相同密钥
func TestPBKDF2DeterministicKey(t *testing.T) {
	password := "my-secret-password"

	key1 := DeriveKeyFromPassword(password)
	key2 := DeriveKeyFromPassword(password)

	if key1 != key2 {
		t.Errorf("相同密码应生成相同的派生密钥\nkey1: %s\nkey2: %s", key1, key2)
	}

	// 验证密钥长度
	if len(key1) != 64 { // 32 字节 hex 编码后是 64 字符
		t.Errorf("派生密钥长度错误，期望 64，得到 %d", len(key1))
	}
}

// TestPBKDF2DifferentPasswords 测试不同密码生成不同密钥
func TestPBKDF2DifferentPasswords(t *testing.T) {
	key1 := DeriveKeyFromPassword("password1")
	key2 := DeriveKeyFromPassword("password2")

	if key1 == key2 {
		t.Error("不同密码不应生成相同的派生密钥")
	}
}

// TestBackwardCompatibilityWithOldHexKey 测试与旧版本 hex 密钥的向后兼容性
func TestBackwardCompatibilityWithOldHexKey(t *testing.T) {
	// 模拟旧版本生成的 hex 密钥 (64字符) - 测试用假数据
	oldHexKey := "0000000000000000000000000000000000000000000000000000000000000001"

	// 旧方法：SHA256 直接哈希
	oldHash := sha256.Sum256([]byte(oldHexKey))

	// 新方法：通过 NewCryptoManager 处理（应自动检测并使用 SHA256）
	cm := NewCryptoManager(oldHexKey)

	// 验证密钥一致性
	if !bytes.Equal(cm.key, oldHash[:]) {
		t.Error("向后兼容性失败：旧 hex 密钥应使用 SHA256 处理")
		t.Errorf("期望密钥: %x", oldHash[:])
		t.Errorf("实际密钥: %x", cm.key)
	}
}

// TestOldEncryptedDataCanBeDecrypted 测试旧加密数据可以被解密
func TestOldEncryptedDataCanBeDecrypted(t *testing.T) {
	// 模拟旧版本的工作流程 - 测试用假数据
	oldHexKey := "0000000000000000000000000000000000000000000000000000000000000002"

	// 使用旧方法加密（模拟）
	oldCM := NewCryptoManager(oldHexKey)
	plaintext := []byte("secret message")
	encrypted, err := oldCM.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("旧方法加密失败: %v", err)
	}

	// 使用新版本解密（应该成功，因为自动检测 hex 格式）
	newCM := NewCryptoManager(oldHexKey)
	decrypted, err := newCM.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("新版本解密旧数据失败: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("解密结果不匹配\n期望: %s\n得到: %s", plaintext, decrypted)
	}
}

// TestNewPasswordUsePBKDF2 测试新密码使用 PBKDF2
func TestNewPasswordUsePBKDF2(t *testing.T) {
	password := "my-new-password" // 不是64字符hex

	// 使用 PBKDF2
	cm := NewCryptoManager(password)

	// 使用 SHA256（旧方法）
	oldHash := sha256.Sum256([]byte(password))

	// 密钥应该不同（PBKDF2 vs SHA256）
	if bytes.Equal(cm.key, oldHash[:]) {
		t.Error("新密码应使用 PBKDF2，不应与 SHA256 结果相同")
	}
}

// TestIsHexString 测试 hex 字符串检测
func TestIsHexString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abcdef1234567890", true}, // 16字符(8字节)
		{"ABCDEF1234567890", true}, // 大写hex
		{"1234567890abcdef", true}, // 16字符
		{"not-hex-string", false},  // 包含非hex字符
		{"12345g", false},          // 包含非hex字符
		{"123456789", false},       // 奇数长度
		{"", false},                // 空字符串
		{"0000000000000000000000000000000000000000000000000000000000000001", true}, // 64字符hex
	}

	for _, tt := range tests {
		result := isHexString(tt.input)
		if result != tt.expected {
			t.Errorf("isHexString(%q) = %v, 期望 %v", tt.input, result, tt.expected)
		}
	}
}

// TestHexKeyExactly64Chars 测试只有恰好64字符的hex才被识别为旧密钥
func TestHexKeyExactly64Chars(t *testing.T) {
	// 63字符 hex - 应使用 PBKDF2 - 测试用假数据
	key63 := "00000000000000000000000000000000000000000000000000000000000000a"
	cm63 := NewCryptoManager(key63)
	old63 := sha256.Sum256([]byte(key63))
	if bytes.Equal(cm63.key, old63[:]) {
		t.Error("63字符hex应使用PBKDF2，不应使用SHA256")
	}

	// 65字符 hex - 应使用 PBKDF2 - 测试用假数据
	key65 := "00000000000000000000000000000000000000000000000000000000000000abcde"
	cm65 := NewCryptoManager(key65)
	old65 := sha256.Sum256([]byte(key65))
	if bytes.Equal(cm65.key, old65[:]) {
		t.Error("65字符hex应使用PBKDF2，不应使用SHA256")
	}

	// 恰好64字符 hex - 应使用 SHA256（向后兼容）- 测试用假数据
	key64 := "0000000000000000000000000000000000000000000000000000000000000001"
	cm64 := NewCryptoManager(key64)
	old64 := sha256.Sum256([]byte(key64))
	if !bytes.Equal(cm64.key, old64[:]) {
		t.Error("64字符hex应使用SHA256保持向后兼容")
	}
}

// TestDecryptBackwardCompatibility 测试解密时的向后兼容性
func TestDecryptBackwardCompatibility(t *testing.T) {
	// 测试场景1：旧版本使用明文密码 + SHA256 加密的数据
	oldPassword := "my-secret-password"

	// 模拟旧版本的加密方式：直接使用 SHA256(password)
	oldKey := sha256.Sum256([]byte(oldPassword))
	tempCM := &CryptoManager{key: oldKey[:]}

	plaintext := []byte("test data for compatibility")
	encrypted, err := tempCM.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("模拟旧版本加密失败: %v", err)
	}

	// 新版本应该能够解密（通过fallback机制）
	newCM := NewCryptoManager(oldPassword)
	decrypted, err := newCM.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("新版本解密旧数据失败: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("解密结果不匹配\n期望: %s\n得到: %s", plaintext, decrypted)
	}
}
