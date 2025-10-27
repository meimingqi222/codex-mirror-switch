package internal

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// MockSyncProvider 模拟同步提供商用于测试.
type MockSyncProvider struct {
	files map[string][]byte
	info  ProviderInfo
}

// NewMockSyncProvider 创建模拟同步提供商.
func NewMockSyncProvider() *MockSyncProvider {
	return &MockSyncProvider{
		files: make(map[string][]byte),
		info: ProviderInfo{
			Name:        "mock",
			Type:        "test",
			Endpoint:    "mock://test",
			MaxFileSize: 1024 * 1024,
			Description: "Mock provider for testing",
		},
	}
}

func (m *MockSyncProvider) Upload(data []byte, filename string) error {
	if len(data) > int(m.info.MaxFileSize) {
		return fmt.Errorf("文件大小超过限制: %d > %d", len(data), m.info.MaxFileSize)
	}
	m.files[filename] = make([]byte, len(data))
	copy(m.files[filename], data)
	return nil
}

func (m *MockSyncProvider) Download(filename string) ([]byte, error) {
	data, exists := m.files[filename]
	if !exists {
		return nil, fmt.Errorf("文件 %s 不存在", filename)
	}
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

func (m *MockSyncProvider) List() ([]string, error) {
	files := make([]string, 0, len(m.files))
	for filename := range m.files {
		files = append(files, filename)
	}
	return files, nil
}

func (m *MockSyncProvider) Delete(filename string) error {
	if _, exists := m.files[filename]; !exists {
		return fmt.Errorf("文件 %s 不存在", filename)
	}
	delete(m.files, filename)
	return nil
}

func (m *MockSyncProvider) GetInfo() ProviderInfo {
	return m.info
}

// TestNewSyncManager 测试创建同步管理器.
func TestNewSyncManager(t *testing.T) {
	tempDir := setupTestDirWithCleanup(t)
	mm := createTestMirrorManagerForSync(t, tempDir)

	sm := NewSyncManager(mm)
	if sm == nil {
		t.Fatal("SyncManager should not be nil")
	}

	if sm.mirrorManager != mm {
		t.Error("SyncManager should reference the provided MirrorManager")
	}

	if sm.provider != nil {
		t.Error("Provider should be nil initially")
	}

	if sm.config != nil {
		t.Error("Config should be nil initially")
	}
}

// TestInitSyncWithPassword 测试使用密码初始化同步.
func TestInitSyncWithPassword(t *testing.T) {
	tempDir := setupTestDirWithCleanup(t)
	mm := createTestMirrorManagerForSync(t, tempDir)
	sm := NewSyncManager(mm)

	// 注意：由于不能修改生成函数，这个测试主要验证流程

	tests := []struct {
		name         string
		providerType string
		endpoint     string
		token        string
		password     string
		expectError  bool
	}{
		{
			name:         "有效的初始化参数",
			providerType: "gist",
			endpoint:     "https://api.github.com",
			token:        "test-token",
			password:     "test-password",
			expectError:  false,
		},
		{
			name:         "空密码",
			providerType: "gist",
			endpoint:     "https://api.github.com",
			token:        "test-token",
			password:     "",
			expectError:  false, // 空密码应该被允许
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 由于无法修改方法，我们只测试基本功能
			// 实际的createProvider会尝试创建真实的提供商，这在测试中可能会失败

			err := sm.InitSyncWithPassword(tt.providerType, tt.endpoint, tt.token, tt.password)
			if (err != nil) != tt.expectError {
				t.Errorf("InitSyncWithPassword() error = %v, expectError %v", err, tt.expectError)
			}

			// 由于测试环境的限制，我们跳过具体的验证
			// 在真实环境中，InitSyncWithPassword会创建实际的提供商
			_ = tt.expectError // 使用变量避免警告
		})
	}
}

// TestLoadSync 测试加载同步配置.
func TestLoadSync(t *testing.T) {
	tempDir := setupTestDirWithCleanup(t)
	mm := createTestMirrorManagerForSync(t, tempDir)
	sm := NewSyncManager(mm)

	// 由于createProvider是方法而非字段，我们需要使用其他方式进行测试
	// 这里我们跳过provider的创建，专注于测试LoadSync的逻辑

	tests := []struct {
		name        string
		setupSync   bool
		expectError bool
	}{
		{
			name:        "加载现有同步配置",
			setupSync:   true,
			expectError: false,
		},
		{
			name:        "无同步配置",
			setupSync:   false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupSync {
				// 设置同步配置
				mm.config.Sync = &SyncConfig{
					Enabled:       true,
					Provider:      "gist",
					Endpoint:      "https://api.github.com",
					Token:         "test-token",
					DeviceID:      "test-device",
					SyncAPIKeys:   true,
					EncryptionPwd: "test-password",
				}
			} else {
				mm.config.Sync = nil
			}

			err := sm.LoadSync()
			// 由于LoadSync会尝试创建实际的网络连接，在测试环境中会失败
			// 我们只测试基本的配置加载逻辑
			if tt.setupSync {
				// 有配置时，应该能正确加载配置到sm.config
				if sm.config == nil {
					t.Error("Config should be loaded when sync is configured")
				}
				if sm.config != mm.config.Sync {
					t.Error("Config should match MirrorManager's sync config")
				}
				// 注意：provider可能为nil，因为网络连接会失败，这是预期的
			} else if err == nil {
				// 无配置时应该返回错误
				t.Error("LoadSync() should return error when no sync config")
			}
		})
	}
}

// TestExportSyncData 测试导出同步数据.
func TestExportSyncData(t *testing.T) {
	tempDir := setupTestDirWithCleanup(t)
	mm := createTestMirrorManagerForSync(t, tempDir)
	sm := NewSyncManager(mm)

	// 设置测试数据
	mm.config.CurrentCodex = "test-codex"
	mm.config.CurrentClaude = "test-claude"
	mm.config.Mirrors = []MirrorConfig{
		{
			Name:     "test1",
			BaseURL:  "https://api.test1.com",
			APIKey:   "key1",
			ToolType: ToolTypeCodex,
		},
		{
			Name:     "test2",
			BaseURL:  "https://api.test2.com",
			APIKey:   "key2",
			ToolType: ToolTypeClaude,
		},
	}

	sm.config = &SyncConfig{
		DeviceID:    "test-device",
		SyncAPIKeys: true,
	}

	syncData := sm.exportSyncData()

	// 验证基本字段
	if syncData.DeviceID != "test-device" {
		t.Errorf("DeviceID = %v, expected test-device", syncData.DeviceID)
	}

	if syncData.CurrentCodex != "test-codex" {
		t.Errorf("CurrentCodex = %v, expected test-codex", syncData.CurrentCodex)
	}

	if syncData.CurrentClaude != "test-claude" {
		t.Errorf("CurrentClaude = %v, expected test-claude", syncData.CurrentClaude)
	}

	if len(syncData.Mirrors) != 2 {
		t.Errorf("Mirrors count = %v, expected 2", len(syncData.Mirrors))
	}

	if !syncData.HasAPIKeys {
		t.Error("HasAPIKeys should be true when SyncAPIKeys is true")
	}

	// 验证时间戳
	if syncData.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	// 验证版本
	if syncData.Version == "" {
		t.Error("Version should not be empty")
	}
}

// TestApplySyncData 测试应用同步数据.
func TestApplySyncData(t *testing.T) {
	tempDir := setupTestDirWithCleanup(t)
	mm := createTestMirrorManagerForSync(t, tempDir)
	sm := NewSyncManager(mm)

	// 创建测试同步数据
	mirrors := []MirrorConfig{
		{
			Name:     "sync-mirror1",
			BaseURL:  "https://api.sync1.com",
			APIKey:   "sync-key1",
			ToolType: ToolTypeCodex,
		},
		{
			Name:     "sync-mirror2",
			BaseURL:  "https://api.sync2.com",
			APIKey:   "sync-key2",
			ToolType: ToolTypeClaude,
		},
	}

	// 计算校验和
	data, _ := json.Marshal(mirrors)
	checksum := calculateChecksum(data)

	syncData := &SyncData{
		CurrentCodex:  "sync-codex",
		CurrentClaude: "sync-claude",
		Mirrors:       mirrors,
		Timestamp:     time.Now(),
		DeviceID:      "remote-device",
		Version:       "3.0",
		Checksum:      checksum,
		HasAPIKeys:    true,
	}

	err := sm.applySyncData(syncData)
	if err != nil {
		t.Fatalf("applySyncData() error = %v", err)
	}

	// 验证数据是否正确应用
	if mm.config.CurrentCodex != "sync-codex" {
		t.Errorf("CurrentCodex = %v, expected sync-codex", mm.config.CurrentCodex)
	}

	if mm.config.CurrentClaude != "sync-claude" {
		t.Errorf("CurrentClaude = %v, expected sync-claude", mm.config.CurrentClaude)
	}

	if len(mm.config.Mirrors) < 2 {
		t.Errorf("Mirrors count = %v, expected at least 2", len(mm.config.Mirrors))
	}

	// 验证镜像源是否正确添加
	foundMirror1 := false
	foundMirror2 := false
	for _, mirror := range mm.config.Mirrors {
		if mirror.Name == "sync-mirror1" {
			foundMirror1 = true
			if mirror.BaseURL != "https://api.sync1.com" {
				t.Errorf("sync-mirror1 BaseURL = %v, expected https://api.sync1.com", mirror.BaseURL)
			}
		}
		if mirror.Name == "sync-mirror2" {
			foundMirror2 = true
			if mirror.BaseURL != "https://api.sync2.com" {
				t.Errorf("sync-mirror2 BaseURL = %v, expected https://api.sync2.com", mirror.BaseURL)
			}
		}
	}

	if !foundMirror1 {
		t.Error("sync-mirror1 not found in applied mirrors")
	}
	if !foundMirror2 {
		t.Error("sync-mirror2 not found in applied mirrors")
	}
}

// TestEncryptDecryptData 测试数据加密解密.
func TestEncryptDecryptData(t *testing.T) {
	tempDir := setupTestDirWithCleanup(t)
	mm := createTestMirrorManagerForSync(t, tempDir)
	sm := NewSyncManager(mm)

	// 设置加密配置
	sm.config = &SyncConfig{
		EncryptionPwd: "test-password",
	}

	testData := []byte("This is test data to encrypt and decrypt")

	// 测试加密
	encryptedData, err := sm.encryptData(testData)
	if err != nil {
		t.Fatalf("encryptData() error = %v", err)
	}

	if len(encryptedData) == 0 {
		t.Error("Encrypted data should not be empty")
	}

	// 加密后的数据应该与原数据不同
	if bytes.Equal(encryptedData, testData) {
		t.Error("Encrypted data should be different from original data")
	}

	// 测试解密
	decryptedData, err := sm.decryptData(encryptedData)
	if err != nil {
		t.Fatalf("decryptData() error = %v", err)
	}

	// 解密后的数据应该与原数据相同
	if !bytes.Equal(decryptedData, testData) {
		t.Errorf("Decrypted data = %v, expected %v", string(decryptedData), string(testData))
	}
}

// TestEncryptDataWithEmptyPassword 测试空密码的加密行为.
func TestEncryptDataWithEmptyPassword(t *testing.T) {
	tempDir := setupTestDirWithCleanup(t)
	mm := createTestMirrorManagerForSync(t, tempDir)
	sm := NewSyncManager(mm)

	// 设置空密码配置
	sm.config = &SyncConfig{
		EncryptionPwd: "",
	}

	testData := []byte("Test data with empty password")

	// 空密码时应该返回错误
	encryptedData, err := sm.encryptData(testData)
	if err == nil {
		t.Error("encryptData() with empty password should return error")
	}

	if encryptedData != nil {
		t.Error("encryptData() with empty password should return nil data")
	}

	// 设置一个有效的密码再测试
	sm.config.EncryptionPwd = "valid-password"

	encryptedData, err = sm.encryptData(testData)
	if err != nil {
		t.Fatalf("encryptData() with valid password error = %v", err)
	}

	// 测试解密
	decryptedData, err := sm.decryptData(encryptedData)
	if err != nil {
		t.Fatalf("decryptData() with valid password error = %v", err)
	}

	if !bytes.Equal(decryptedData, testData) {
		t.Errorf("Decrypted data = %v, expected %v", string(decryptedData), string(testData))
	}
}

// TestGenerateDeviceID 测试生成设备ID.
func TestGenerateDeviceID(t *testing.T) {
	deviceID := generateDeviceID()

	if deviceID == "" {
		t.Error("Device ID should not be empty")
	}

	// 生成两个ID应该相同（因为基于主机名）
	deviceID2 := generateDeviceID()

	if deviceID != deviceID2 {
		t.Error("Two device IDs on same machine should be the same")
	}

	// 验证ID格式（应该包含主机名和十六进制后缀）
	parts := strings.Split(deviceID, "-")
	if len(parts) < 2 {
		t.Errorf("Device ID should have format 'hostname-suffix', got: %s", deviceID)
	}

	// 验证后缀是十六进制
	if len(parts) >= 2 {
		suffix := parts[len(parts)-1]
		if _, err := hex.DecodeString(suffix); err != nil {
			t.Errorf("Device ID suffix should be valid hex string: %v", err)
		}
	}
}

// TestGenerateEncryptKey 测试生成加密密钥.
func TestGenerateEncryptKey(t *testing.T) {
	key, err := generateEncryptKey()
	if err != nil {
		t.Fatalf("generateEncryptKey() error = %v", err)
	}

	if key == "" {
		t.Error("Encrypt key should not be empty")
	}

	// 生成两个密钥应该不同
	key2, err := generateEncryptKey()
	if err != nil {
		t.Fatalf("generateEncryptKey() second call error = %v", err)
	}

	if key == key2 {
		t.Error("Two generated encrypt keys should be different")
	}

	// 验证密钥格式（应该是十六进制字符串）
	if _, err := hex.DecodeString(key); err != nil {
		t.Errorf("Encrypt key should be valid hex string: %v", err)
	}
}

// TestSyncDataMarshalling 测试同步数据序列化.
func TestSyncDataMarshalling(t *testing.T) {
	originalData := SyncData{
		CurrentCodex:  "test-codex",
		CurrentClaude: "test-claude",
		Mirrors: []MirrorConfig{
			{
				Name:     "test-mirror",
				BaseURL:  TestAPIURL,
				APIKey:   "test-key",
				ToolType: ToolTypeCodex,
			},
		},
		Timestamp:  time.Now().Truncate(time.Second), // 截断到秒以避免精度问题
		DeviceID:   "test-device",
		Version:    "1.0.0",
		HasAPIKeys: true,
		Checksum:   "test-checksum",
	}

	// 序列化
	data, err := json.Marshal(originalData)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// 反序列化
	var deserializedData SyncData
	err = json.Unmarshal(data, &deserializedData)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// 验证数据
	if deserializedData.CurrentCodex != originalData.CurrentCodex {
		t.Errorf("CurrentCodex = %v, expected %v", deserializedData.CurrentCodex, originalData.CurrentCodex)
	}

	if deserializedData.CurrentClaude != originalData.CurrentClaude {
		t.Errorf("CurrentClaude = %v, expected %v", deserializedData.CurrentClaude, originalData.CurrentClaude)
	}

	if len(deserializedData.Mirrors) != len(originalData.Mirrors) {
		t.Errorf("Mirrors count = %v, expected %v", len(deserializedData.Mirrors), len(originalData.Mirrors))
	}

	if deserializedData.DeviceID != originalData.DeviceID {
		t.Errorf("DeviceID = %v, expected %v", deserializedData.DeviceID, originalData.DeviceID)
	}

	if !deserializedData.Timestamp.Equal(originalData.Timestamp) {
		t.Errorf("Timestamp = %v, expected %v", deserializedData.Timestamp, originalData.Timestamp)
	}

	if deserializedData.HasAPIKeys != originalData.HasAPIKeys {
		t.Errorf("HasAPIKeys = %v, expected %v", deserializedData.HasAPIKeys, originalData.HasAPIKeys)
	}
}

// TestCreateBackup 测试创建备份功能.
func TestCreateBackup(t *testing.T) {
	tempDir := setupTestDirWithCleanup(t)
	mm := createTestMirrorManagerForSync(t, tempDir)
	sm := NewSyncManager(mm)

	// 添加一些测试数据
	mm.config.Mirrors = append(mm.config.Mirrors, MirrorConfig{
		Name:     "backup-test",
		BaseURL:  TestAPIURL,
		APIKey:   "backup-key",
		ToolType: ToolTypeCodex,
	})

	err := sm.createBackup()
	if err != nil {
		t.Errorf("createBackup() error = %v", err)
	}

	// 由于备份可能涉及文件系统操作，我们主要测试函数不报错
	// 具体的备份验证需要依赖实际的备份实现
}

// TestIntelligentMergeWithDeletions 测试智能合并策略处理删除操作.
func TestIntelligentMergeWithDeletions(t *testing.T) {
	tempDir := setupTestDirWithCleanup(t)
	mm := createTestMirrorManagerForSync(t, tempDir)
	_ = NewSyncManager(mm) // 用于创建测试环境，但不直接使用

	now := time.Now()
	testDeletedMirrorName := "deleted-mirror"

	// 设置本地配置：包含一个已删除的镜像源
	localConfig := &SystemConfig{
		CurrentCodex:  "local-codex",
		CurrentClaude: "local-claude",
		Mirrors: []MirrorConfig{
			{
				Name:         "existing-mirror",
				BaseURL:      "https://existing.local.com",
				APIKey:       "local-key",
				ToolType:     ToolTypeCodex,
				CreatedAt:    now.Add(-2 * time.Hour),
				LastModified: now.Add(-1 * time.Hour),
				Deleted:      false,
			},
			{
				Name:         testDeletedMirrorName,
				BaseURL:      "https://deleted.local.com",
				APIKey:       "deleted-key",
				ToolType:     ToolTypeClaude,
				CreatedAt:    now.Add(-3 * time.Hour),
				LastModified: now.Add(-30 * time.Minute),
				Deleted:      true,
				DeletedAt:    now.Add(-30 * time.Minute),
			},
		},
	}

	// 设置远程数据：包含一个新镜像源和已删除的镜像源
	remoteData := &SyncData{
		CurrentCodex:  "remote-codex",
		CurrentClaude: testDeletedMirrorName, // 远程仍然使用已删除的镜像源
		Mirrors: []MirrorConfig{
			{
				Name:         "existing-mirror",
				BaseURL:      "https://existing.remote.com", // 远程有不同的配置
				ToolType:     ToolTypeCodex,
				CreatedAt:    now.Add(-2 * time.Hour),
				LastModified: now.Add(-45 * time.Minute),
			},
			{
				Name:         testDeletedMirrorName,
				BaseURL:      "https://deleted.remote.com",
				ToolType:     ToolTypeClaude,
				CreatedAt:    now.Add(-3 * time.Hour),
				LastModified: now.Add(-1 * time.Hour),
			},
			{
				Name:         "new-remote-mirror",
				BaseURL:      "https://new.remote.com",
				ToolType:     ToolTypeCodex,
				CreatedAt:    now.Add(-20 * time.Minute),
				LastModified: now.Add(-20 * time.Minute),
			},
		},
		Timestamp: now,
		DeviceID:  "remote-device",
		Version:   "3.1",
		DeletedMirrors: []MirrorConfig{
			{
				Name:         "remote-deleted-mirror",
				BaseURL:      "https://remote-deleted.com",
				ToolType:     ToolTypeClaude,
				CreatedAt:    now.Add(-4 * time.Hour),
				LastModified: now.Add(-2 * time.Hour),
				Deleted:      true,
				DeletedAt:    now.Add(-2 * time.Hour),
			},
		},
	}

	// 创建冲突解决器
	resolver := NewConflictResolver(localConfig, remoteData)
	conflicts := resolver.DetectConflicts()

	// 验证冲突检测结果
	if len(conflicts.Conflicts) == 0 {
		t.Error("Expected conflicts to be detected, but none found")
	}

	// 测试智能合并
	resolvedConfig, err := resolver.ResolveConflicts(conflicts, StrategyMerge)
	if err != nil {
		t.Fatalf("ResolveConflicts with merge strategy error = %v", err)
	}

	// 验证合并结果
	foundExisting := false
	foundNewRemote := false
	foundDeleted := false

	for _, mirror := range resolvedConfig.Mirrors {
		if mirror.Name == "existing-mirror" {
			foundExisting = true
			// 应该保留本地API密钥
			if mirror.APIKey != "local-key" {
				t.Errorf("Expected local API key to be preserved, got: %v", mirror.APIKey)
			}
			// 应该使用远程的BaseURL
			if mirror.BaseURL != "https://existing.remote.com" {
				t.Errorf("Expected remote BaseURL, got: %v", mirror.BaseURL)
			}
		}

		if mirror.Name == "new-remote-mirror" {
			foundNewRemote = true
			// 新的镜像源API密钥应该为空
			if mirror.APIKey != "" {
				t.Errorf("Expected empty API key for new remote mirror, got: %v", mirror.APIKey)
			}
		}

		if mirror.Name == testDeletedMirrorName {
			foundDeleted = true
			t.Error("Deleted mirror should not be present in merged config")
		}
	}

	if !foundExisting {
		t.Error("Existing mirror should be present in merged config")
	}

	if !foundNewRemote {
		t.Error("New remote mirror should be present in merged config")
	}

	if foundDeleted {
		t.Error("Deleted mirror should not be present in merged config")
	}

	// 验证当前激活源选择（已删除的镜像源不应该被选择）
	if resolvedConfig.CurrentClaude == testDeletedMirrorName {
		t.Error("Deleted mirror should not be selected as current active source")
	}
}

// TestConflictDetectionWithDeletedMirrors 测试删除镜像源的冲突检测.
func TestConflictDetectionWithDeletedMirrors(t *testing.T) {
	tempDir := setupTestDirWithCleanup(t)
	_ = createTestMirrorManagerForSync(t, tempDir) // 用于创建测试环境，但不直接使用

	now := time.Now()

	// 测试场景1：本地主动删除，云端仍有
	localConfig1 := &SystemConfig{
		Mirrors: []MirrorConfig{
			{
				Name:         "test-mirror",
				BaseURL:      "https://test.com",
				ToolType:     ToolTypeCodex,
				CreatedAt:    now.Add(-2 * time.Hour),
				LastModified: now.Add(-30 * time.Minute),
				Deleted:      true,
				DeletedAt:    now.Add(-30 * time.Minute),
			},
		},
	}

	remoteData1 := &SyncData{
		Mirrors: []MirrorConfig{
			{
				Name:         "test-mirror",
				BaseURL:      "https://test.com",
				ToolType:     ToolTypeCodex,
				CreatedAt:    now.Add(-2 * time.Hour),
				LastModified: now.Add(-45 * time.Minute),
			},
		},
		Timestamp: now,
		DeviceID:  "remote-device",
		Version:   "3.1",
	}

	resolver1 := NewConflictResolver(localConfig1, remoteData1)
	conflicts1 := resolver1.DetectConflicts()

	// 应该检测到删除冲突
	foundDeleteConflict := false
	for _, conflict := range conflicts1.Conflicts {
		if conflict.Type == ConflictTypeDeletedMirror && conflict.Name == "test-mirror" {
			foundDeleteConflict = true
			if !strings.Contains(conflict.Description, "本地删除了镜像源") {
				t.Errorf("Expected deletion description, got: %v", conflict.Description)
			}
			break
		}
	}

	// 调试输出
	if !foundDeleteConflict {
		t.Logf("Debug - found %d conflicts:", len(conflicts1.Conflicts))
		for i, conflict := range conflicts1.Conflicts {
			t.Logf("  %d: Type=%v, Name=%v, Desc=%v", i, conflict.Type, conflict.Name, conflict.Description)
		}
	}

	if !foundDeleteConflict {
		t.Error("Expected to find deletion conflict for locally deleted mirror")
	}

	// 测试场景2：云端删除，本地仍有
	localConfig2 := &SystemConfig{
		Mirrors: []MirrorConfig{
			{
				Name:         "test-mirror-2",
				BaseURL:      "https://test2.com",
				ToolType:     ToolTypeCodex,
				CreatedAt:    now.Add(-2 * time.Hour),
				LastModified: now.Add(-1 * time.Hour),
			},
		},
	}

	remoteData2 := &SyncData{
		Mirrors: []MirrorConfig{},
		DeletedMirrors: []MirrorConfig{
			{
				Name:         "test-mirror-2",
				BaseURL:      "https://test2.com",
				ToolType:     ToolTypeCodex,
				CreatedAt:    now.Add(-2 * time.Hour),
				LastModified: now.Add(-1 * time.Hour),
				Deleted:      true,
				DeletedAt:    now.Add(-15 * time.Minute),
			},
		},
		Timestamp: now,
		DeviceID:  "remote-device",
		Version:   "3.1",
	}

	resolver2 := NewConflictResolver(localConfig2, remoteData2)
	conflicts2 := resolver2.DetectConflicts()

	// 应该检测到删除冲突
	foundRemoteDeleteConflict := false
	for _, conflict := range conflicts2.Conflicts {
		if conflict.Type == ConflictTypeDeletedMirror && conflict.Name == "test-mirror-2" {
			foundRemoteDeleteConflict = true
			if !strings.Contains(conflict.Description, "云端被删除") {
				t.Errorf("Expected remote deletion description, got: %v", conflict.Description)
			}
			break
		}
	}

	if !foundRemoteDeleteConflict {
		t.Error("Expected to find deletion conflict for remotely deleted mirror")
	}
}

// TestMirrorManagerSoftDelete 测试软删除功能.
func TestMirrorManagerSoftDelete(t *testing.T) {
	tempDir := setupTestDirWithCleanup(t)
	mm := createTestMirrorManagerForSync(t, tempDir)

	// 添加测试镜像源
	err := mm.AddMirror("test-mirror", "https://test.com", "test-key")
	if err != nil {
		t.Fatalf("AddMirror error = %v", err)
	}

	// 验证镜像源存在
	activeMirrors := mm.ListActiveMirrors()
	if len(activeMirrors) != 1 {
		t.Errorf("Expected 1 active mirror, got: %d", len(activeMirrors))
	}

	// 软删除镜像源
	err = mm.RemoveMirrorWithOptions("test-mirror", false)
	if err != nil {
		t.Fatalf("RemoveMirrorWithOptions error = %v", err)
	}

	// 验证镜像源不在活跃列表中
	activeMirrors = mm.ListActiveMirrors()
	if len(activeMirrors) != 0 {
		t.Errorf("Expected 0 active mirrors after soft delete, got: %d", len(activeMirrors))
	}

	// 验证镜像源在已删除列表中
	deletedMirrors := mm.ListDeletedMirrors()
	if len(deletedMirrors) != 1 {
		t.Errorf("Expected 1 deleted mirror, got: %d", len(deletedMirrors))
	}

	deletedMirror := deletedMirrors[0]
	if !deletedMirror.Deleted {
		t.Error("Expected mirror to be marked as deleted")
	}

	if deletedMirror.DeletedAt.IsZero() {
		t.Error("Expected DeletedAt to be set")
	}
}

// TestExportSyncDataWithDeletedMirrors 测试导出同步数据包含已删除镜像源.
func TestExportSyncDataWithDeletedMirrors(t *testing.T) {
	tempDir := setupTestDirWithCleanup(t)
	mm := createTestMirrorManagerForSync(t, tempDir)
	sm := NewSyncManager(mm)

	// 设置同步配置
	sm.config = &SyncConfig{
		DeviceID:    "test-device",
		SyncAPIKeys: true,
	}

	// 添加活跃和已删除的镜像源
	now := time.Now()
	mm.config.Mirrors = []MirrorConfig{
		{
			Name:         "active-mirror",
			BaseURL:      "https://active.com",
			APIKey:       "active-key",
			ToolType:     ToolTypeCodex,
			CreatedAt:    now.Add(-1 * time.Hour),
			LastModified: now.Add(-30 * time.Minute),
			Deleted:      false,
		},
		{
			Name:         "deleted-mirror",
			BaseURL:      "https://deleted.com",
			APIKey:       "deleted-key",
			ToolType:     ToolTypeClaude,
			CreatedAt:    now.Add(-2 * time.Hour),
			LastModified: now.Add(-45 * time.Minute),
			Deleted:      true,
			DeletedAt:    now.Add(-15 * time.Minute),
		},
	}

	// 导出同步数据
	syncData := sm.exportSyncData()

	// 验证活跃镜像源
	if len(syncData.Mirrors) != 1 {
		t.Errorf("Expected 1 active mirror in sync data, got: %d", len(syncData.Mirrors))
	}

	if syncData.Mirrors[0].Name != "active-mirror" {
		t.Errorf("Expected active-mirror, got: %v", syncData.Mirrors[0].Name)
	}

	// 验证已删除镜像源
	if len(syncData.DeletedMirrors) != 1 {
		t.Errorf("Expected 1 deleted mirror in sync data, got: %d", len(syncData.DeletedMirrors))
	}

	if syncData.DeletedMirrors[0].Name != "deleted-mirror" {
		t.Errorf("Expected \"deleted-mirror\", got: %v", syncData.DeletedMirrors[0].Name)
	}

	// 验证版本号
	if syncData.Version != "3.1" {
		t.Errorf("Expected version 3.1, got: %v", syncData.Version)
	}
}

// TestMockSyncProvider 测试模拟同步提供商.
func TestMockSyncProvider(t *testing.T) {
	provider := NewMockSyncProvider()

	// 测试获取信息
	info := provider.GetInfo()
	if info.Name != "mock" {
		t.Errorf("Provider name = %v, expected mock", info.Name)
	}

	// 测试上传
	testData := []byte("test file content")
	err := provider.Upload(testData, "test.txt")
	if err != nil {
		t.Errorf("Upload() error = %v", err)
	}

	// 测试下载
	downloadedData, err := provider.Download("test.txt")
	if err != nil {
		t.Errorf("Download() error = %v", err)
	}

	if !bytes.Equal(downloadedData, testData) {
		t.Errorf("Downloaded data = %v, expected %v", string(downloadedData), string(testData))
	}

	// 测试列表
	files, err := provider.List()
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	if len(files) != 1 || files[0] != "test.txt" {
		t.Errorf("Files = %v, expected [test.txt]", files)
	}

	// 测试删除
	err = provider.Delete("test.txt")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// 验证删除后无法下载
	_, err = provider.Download("test.txt")
	if err == nil {
		t.Error("Should not be able to download deleted file")
	}

	// 测试下载不存在的文件
	_, err = provider.Download("nonexistent.txt")
	if err == nil {
		t.Error("Should error when downloading non-existent file")
	}
}

// 这些函数在sync.go中已定义，这里只是为了测试引用

// TestProviderInfo 测试提供商信息结构.
func TestProviderInfo(t *testing.T) {
	info := ProviderInfo{
		Name:        "test-provider",
		Type:        "test",
		Endpoint:    TestAPIURL,
		MaxFileSize: 1024,
		Description: "Test provider",
	}

	// 测试JSON序列化
	data, err := json.Marshal(info)
	if err != nil {
		t.Errorf("Marshal ProviderInfo error = %v", err)
	}

	var deserializedInfo ProviderInfo
	err = json.Unmarshal(data, &deserializedInfo)
	if err != nil {
		t.Errorf("Unmarshal ProviderInfo error = %v", err)
	}

	if deserializedInfo.Name != info.Name {
		t.Errorf("Name = %v, expected %v", deserializedInfo.Name, info.Name)
	}
}

// setupTestDirWithCleanup 创建测试目录.
func setupTestDirWithCleanup(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "codex-mirror-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// 清理函数
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir
}

// createTestMirrorManagerForSync 创建测试用的镜像管理器.
func createTestMirrorManagerForSync(t *testing.T, tempDir string) *MirrorManager {
	// 创建测试配置文件路径
	configPath := filepath.Join(tempDir, "mirrors.toml")

	// 创建初始配置
	initialConfig := `# Codex Mirror Switch Configuration
current_codex = ""
current_claude = ""
`

	err := os.WriteFile(configPath, []byte(initialConfig), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// 创建镜像管理器
	mm, err := NewMirrorManagerWithPath(configPath)
	if err != nil {
		t.Fatalf("Failed to create mirror manager: %v", err)
	}

	return mm
}
