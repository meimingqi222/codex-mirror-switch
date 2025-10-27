package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupTestDir 创建临时测试目录.
func setupTestDir(t *testing.T) string {
	tempDir := t.TempDir()
	return tempDir
}

// TestNewMirrorManager 测试创建镜像源管理器.
func TestNewMirrorManager(t *testing.T) {
	// 设置临时主目录
	tempDir := setupTestDir(t)
	oldHome := os.Getenv("HOME")
	if oldHome == "" {
		oldHome = os.Getenv("USERPROFILE") // Windows
	}

	// 使用临时目录作为home目录
	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)
	defer func() {
		if oldHome != "" {
			os.Setenv("HOME", oldHome)
			os.Setenv("USERPROFILE", oldHome)
		}
	}()

	mm, err := NewMirrorManager()
	if err != nil {
		t.Fatalf("创建镜像源管理器失败: %v", err)
	}

	if mm == nil {
		t.Fatal("镜像源管理器为空")
	}

	// 检查配置目录是否创建
	configDir := filepath.Join(tempDir, ".codex-mirror")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Errorf("配置目录 %s 未创建", configDir)
	}
}

// TestAddMirror 测试添加镜像源.
func TestAddMirror(t *testing.T) {
	tempDir := setupTestDir(t)
	mm := createTestMirrorManager(t, tempDir)

	tests := []struct {
		name        string
		mirrorName  string
		baseURL     string
		apiKey      string
		expectError bool
	}{
		{
			name:        "添加有效镜像源",
			mirrorName:  "test-mirror",
			baseURL:     "https://api.test.com",
			apiKey:      "test-api-key",
			expectError: false,
		},
		{
			name:        "添加重复镜像源",
			mirrorName:  "test-mirror",
			baseURL:     "https://api.test2.com",
			apiKey:      "test-api-key2",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mm.AddMirror(tt.mirrorName, tt.baseURL, tt.apiKey)
			if (err != nil) != tt.expectError {
				t.Errorf("AddMirror() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}

	// 验证镜像源是否正确添加
	mirrors := mm.ListMirrors()
	found := false
	for _, mirror := range mirrors {
		if mirror.Name == "test-mirror" {
			found = true
			if mirror.BaseURL != "https://api.test.com" {
				t.Errorf("期望 BaseURL 为 'https://api.test.com', 实际为 %s", mirror.BaseURL)
			}
			if mirror.APIKey != "test-api-key" {
				t.Errorf("期望 APIKey 为 'test-api-key', 实际为 %s", mirror.APIKey)
			}
			if mirror.ToolType != ToolTypeCodex {
				t.Errorf("期望 ToolType 为 %s, 实际为 %s", ToolTypeCodex, mirror.ToolType)
			}
		}
	}
	if !found {
		t.Error("添加的镜像源未在列表中找到")
	}
}

// TestAddMirrorWithType 测试添加指定类型的镜像源.
func TestAddMirrorWithType(t *testing.T) {
	tempDir := setupTestDir(t)
	mm := createTestMirrorManager(t, tempDir)

	tests := []struct {
		name       string
		mirrorName string
		baseURL    string
		apiKey     string
		toolType   ToolType
	}{
		{
			name:       "添加Codex镜像源",
			mirrorName: "codex-mirror",
			baseURL:    "https://api.openai.com",
			apiKey:     "sk-test-key",
			toolType:   ToolTypeCodex,
		},
		{
			name:       "添加Claude镜像源",
			mirrorName: "claude-mirror",
			baseURL:    "https://api.anthropic.com",
			apiKey:     "claude-test-key",
			toolType:   ToolTypeClaude,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mm.AddMirrorWithType(tt.mirrorName, tt.baseURL, tt.apiKey, tt.toolType)
			if err != nil {
				t.Errorf("AddMirrorWithType() error = %v", err)
			}

			// 验证镜像源类型
			mirror, err := mm.GetMirrorByName(tt.mirrorName)
			if err != nil {
				t.Errorf("获取镜像源失败: %v", err)
			}
			if mirror.ToolType != tt.toolType {
				t.Errorf("期望 ToolType 为 %s, 实际为 %s", tt.toolType, mirror.ToolType)
			}

			// 验证环境变量key设置
			expectedEnvKey := ""
			switch tt.toolType {
			case ToolTypeCodex:
				expectedEnvKey = CodexSwitchAPIKeyEnv
			case ToolTypeClaude:
				expectedEnvKey = "ANTHROPIC_AUTH_TOKEN"
			}
			if mirror.EnvKey != expectedEnvKey {
				t.Errorf("期望 EnvKey 为 %s, 实际为 %s", expectedEnvKey, mirror.EnvKey)
			}
		})
	}
}

// TestRemoveMirror 测试删除镜像源.
func TestRemoveMirror(t *testing.T) {
	tempDir := setupTestDir(t)
	mm := createTestMirrorManager(t, tempDir)

	// 先添加一个测试镜像源
	err := mm.AddMirror("test-remove", "https://api.test.com", "test-key")
	if err != nil {
		t.Fatalf("添加测试镜像源失败: %v", err)
	}

	tests := []struct {
		name        string
		mirrorName  string
		expectError bool
	}{
		{
			name:        "删除存在的镜像源",
			mirrorName:  "test-remove",
			expectError: false,
		},
		{
			name:        "删除官方镜像源",
			mirrorName:  DefaultMirrorName,
			expectError: true,
		},
		{
			name:        "删除不存在的镜像源",
			mirrorName:  "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mm.RemoveMirror(tt.mirrorName)
			if (err != nil) != tt.expectError {
				t.Errorf("RemoveMirror() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}

	// 验证镜像源是否被删除
	mirrors := mm.ListMirrors()
	for _, mirror := range mirrors {
		if mirror.Name == "test-remove" {
			t.Error("镜像源应该已被删除")
		}
	}
}

// TestSwitchMirror 测试切换镜像源.
func TestSwitchMirror(t *testing.T) {
	tempDir := setupTestDir(t)
	mm := createTestMirrorManager(t, tempDir)

	// 添加测试镜像源
	err := mm.AddMirror("test-switch", "https://api.test.com", "test-key")
	if err != nil {
		t.Fatalf("添加测试镜像源失败: %v", err)
	}

	tests := []struct {
		name        string
		mirrorName  string
		expectError bool
	}{
		{
			name:        "切换到存在的镜像源",
			mirrorName:  "test-switch",
			expectError: false,
		},
		{
			name:        "切换到不存在的镜像源",
			mirrorName:  "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mm.SwitchMirror(tt.mirrorName)
			if (err != nil) != tt.expectError {
				t.Errorf("SwitchMirror() error = %v, expectError %v", err, tt.expectError)
			}

			// 如果切换成功，验证当前镜像源
			if !tt.expectError {
				if mm.config.CurrentMirror != tt.mirrorName {
					t.Errorf("期望当前镜像源为 %s, 实际为 %s", tt.mirrorName, mm.config.CurrentMirror)
				}
			}
		})
	}
}

// TestGetCurrentMirror 测试获取当前镜像源.
func TestGetCurrentMirror(t *testing.T) {
	tempDir := setupTestDir(t)
	mm := createTestMirrorManager(t, tempDir)

	// 初始状态下应该有默认镜像源
	currentMirror, err := mm.GetCurrentMirror()
	if err != nil {
		t.Errorf("获取当前镜像源失败: %v", err)
	}
	if currentMirror.Name != DefaultMirrorName {
		t.Errorf("期望当前镜像源名称为 %s, 实际为 %s", DefaultMirrorName, currentMirror.Name)
	}

	// 添加并切换到新镜像源
	err = mm.AddMirror("test-current", "https://api.test.com", "test-key")
	if err != nil {
		t.Fatalf("添加测试镜像源失败: %v", err)
	}

	err = mm.SwitchMirror("test-current")
	if err != nil {
		t.Fatalf("切换镜像源失败: %v", err)
	}

	currentMirror, err = mm.GetCurrentMirror()
	if err != nil {
		t.Errorf("获取当前镜像源失败: %v", err)
	}
	if currentMirror.Name != "test-current" {
		t.Errorf("期望当前镜像源名称为 test-current, 实际为 %s", currentMirror.Name)
	}
}

// TestUpdateMirror 测试更新镜像源.
func TestUpdateMirror(t *testing.T) {
	tempDir := setupTestDir(t)
	mm := createTestMirrorManager(t, tempDir)

	// 添加测试镜像源
	err := mm.AddMirror("test-update", "https://api.test.com", "old-key")
	if err != nil {
		t.Fatalf("添加测试镜像源失败: %v", err)
	}

	tests := []struct {
		name        string
		mirrorName  string
		newBaseURL  string
		newAPIKey   string
		expectError bool
	}{
		{
			name:        "更新存在的镜像源",
			mirrorName:  "test-update",
			newBaseURL:  "https://api.newtest.com",
			newAPIKey:   "new-key",
			expectError: false,
		},
		{
			name:        "更新不存在的镜像源",
			mirrorName:  "nonexistent",
			newBaseURL:  "https://api.test.com",
			newAPIKey:   "test-key",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mm.UpdateMirror(tt.mirrorName, tt.newBaseURL, tt.newAPIKey)
			if (err != nil) != tt.expectError {
				t.Errorf("UpdateMirror() error = %v, expectError %v", err, tt.expectError)
			}

			// 如果更新成功，验证更新结果
			if !tt.expectError {
				mirror, err := mm.GetMirrorByName(tt.mirrorName)
				if err != nil {
					t.Errorf("获取镜像源失败: %v", err)
				}
				if mirror.BaseURL != tt.newBaseURL {
					t.Errorf("期望 BaseURL 为 %s, 实际为 %s", tt.newBaseURL, mirror.BaseURL)
				}
				if mirror.APIKey != tt.newAPIKey {
					t.Errorf("期望 APIKey 为 %s, 实际为 %s", tt.newAPIKey, mirror.APIKey)
				}
			}
		})
	}
}

// TestSanitizeEnvVarName 测试环境变量名称清理函数.
func TestSanitizeEnvVarName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "NORMAL"},
		{"with-dash", "WITH_DASH"},
		{"with space", "WITH_SPACE"},
		{"with123numbers", "WITH123NUMBERS"},
		{"123start", "MIRROR_123START"},
		{"", "MIRROR"},
		{"special@#$chars", "SPECIALCHARS"},
		{"mixed-case_Test", "MIXED_CASE_TEST"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeEnvVarName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeEnvVarName(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

// TestExtractMirrorNameFromURL 测试从URL提取镜像源名称.
func TestExtractMirrorNameFromURL(t *testing.T) {
	tests := []struct {
		url         string
		defaultName string
		expected    string
	}{
		{"https://api.openai.com", "default", "api-openai"},
		{"https://api.anthropic.com", "default", "api-anthropic"},
		{"https://custom.example.com/api", "default", "custom-example"},
		{"http://localhost:3000", "default", "localhost"},
		{"invalid-url", "default", "default"},
		{"", "default", "default"},
		{"https://sub.domain.com", "default", "sub-domain"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := extractMirrorNameFromURL(tt.url, tt.defaultName)
			if result != tt.expected {
				t.Errorf("extractMirrorNameFromURL(%s, %s) = %s, expected %s", tt.url, tt.defaultName, result, tt.expected)
			}
		})
	}
}

// TestMirrorFixEnvKeyFormat 测试修复环境变量key格式.
func TestMirrorFixEnvKeyFormat(t *testing.T) {
	tempDir := setupTestDir(t)
	mm := createTestMirrorManager(t, tempDir)

	// 添加一些镜像源（有些可能有错误的env_key）
	mm.config.Mirrors = []MirrorConfig{
		{
			Name:     "codex-test",
			BaseURL:  "https://api.test.com",
			APIKey:   "test-key",
			EnvKey:   "", // 空env_key，需要修复
			ToolType: ToolTypeCodex,
		},
		{
			Name:     "claude-test",
			BaseURL:  "https://api.anthropic.com",
			APIKey:   "claude-key",
			EnvKey:   "OLD_WRONG_KEY", // 错误的env_key，需要修复
			ToolType: ToolTypeClaude,
		},
		{
			Name:     "correct-one",
			BaseURL:  "https://api.example.com",
			APIKey:   "correct-key",
			EnvKey:   CodexSwitchAPIKeyEnv, // 正确的env_key
			ToolType: ToolTypeCodex,
		},
	}

	err := mm.FixEnvKeyFormat()
	if err != nil {
		t.Errorf("FixEnvKeyFormat() error = %v", err)
	}

	// 验证修复结果
	for _, mirror := range mm.config.Mirrors {
		switch mirror.ToolType {
		case ToolTypeCodex:
			if mirror.EnvKey != CodexSwitchAPIKeyEnv {
				t.Errorf("镜像源 %s 的 EnvKey 修复失败，期望 %s，实际 %s", mirror.Name, CodexSwitchAPIKeyEnv, mirror.EnvKey)
			}
		case ToolTypeClaude:
			if mirror.EnvKey != "ANTHROPIC_AUTH_TOKEN" {
				t.Errorf("镜像源 %s 的 EnvKey 修复失败，期望 ANTHROPIC_AUTH_TOKEN，实际 %s", mirror.Name, mirror.EnvKey)
			}
		}
	}
}

// createTestMirrorManager 创建用于测试的镜像源管理器.
func createTestMirrorManager(t *testing.T, tempDir string) *MirrorManager {
	// 设置临时环境变量
	oldHome := os.Getenv("HOME")
	if oldHome == "" {
		oldHome = os.Getenv("USERPROFILE") // Windows
	}

	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)

	t.Cleanup(func() {
		if oldHome != "" {
			os.Setenv("HOME", oldHome)
			os.Setenv("USERPROFILE", oldHome)
		}
	})

	// 创建配置目录
	configDir := filepath.Join(tempDir, ".codex-mirror")
	if err := EnsureDir(configDir); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}

	// 手动创建MirrorManager，避免环境发现干扰测试
	configPath := filepath.Join(configDir, "mirrors.toml")
	mm := &MirrorManager{
		configPath: configPath,
		config: &SystemConfig{
			CurrentMirror: DefaultMirrorName,
			CurrentCodex:  DefaultMirrorName,
			Mirrors: []MirrorConfig{
				{
					Name:     DefaultMirrorName,
					BaseURL:  "https://api.openai.com",
					APIKey:   "",
					ToolType: ToolTypeCodex,
					EnvKey:   CodexSwitchAPIKeyEnv,
				},
			},
		},
	}

	// 保存初始配置
	if err := mm.saveConfig(); err != nil {
		t.Fatalf("保存初始配置失败: %v", err)
	}

	return mm
}

// TestSystemConfigPersistence 测试系统配置持久化.
func TestSystemConfigPersistence(t *testing.T) {
	tempDir := setupTestDir(t)

	// 创建第一个管理器并添加配置
	mm1 := createTestMirrorManager(t, tempDir)
	err := mm1.AddMirror("persistence-test", "https://api.test.com", "test-key")
	if err != nil {
		t.Fatalf("添加测试镜像源失败: %v", err)
	}

	// 添加sync配置
	syncConfig := &SyncConfig{
		Enabled:      true,
		Provider:     "gist",
		Endpoint:     "https://api.github.com",
		Token:        "test-token",
		AutoSync:     true,
		SyncInterval: 30,
		DeviceID:     "test-device-id",
		LastSync:     time.Now(),
		SyncAPIKeys:  true,
	}
	mm1.config.Sync = syncConfig
	err = mm1.saveConfig()
	if err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}

	// 创建第二个管理器，应该加载第一个的配置
	mm2 := createTestMirrorManager(t, tempDir)
	err = mm2.loadConfig()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置是否正确加载
	mirror, err := mm2.GetMirrorByName("persistence-test")
	if err != nil {
		t.Errorf("配置持久化失败，找不到镜像源: %v", err)
	}
	if mirror.BaseURL != "https://api.test.com" {
		t.Errorf("配置持久化失败，期望 BaseURL 为 'https://api.test.com'，实际为 %s", mirror.BaseURL)
	}

	// 验证sync配置
	if mm2.config.Sync == nil {
		t.Error("Sync配置未正确持久化")
	} else {
		if mm2.config.Sync.Provider != "gist" {
			t.Errorf("Sync配置持久化失败，期望 Provider 为 'gist'，实际为 %s", mm2.config.Sync.Provider)
		}
		if mm2.config.Sync.DeviceID != "test-device-id" {
			t.Errorf("Sync配置持久化失败，期望 DeviceID 为 'test-device-id'，实际为 %s", mm2.config.Sync.DeviceID)
		}
	}
}
