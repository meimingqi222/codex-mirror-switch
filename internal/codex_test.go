package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

// TestNewCodexConfigManager 测试创建Codex配置管理器.
func TestNewCodexConfigManager(t *testing.T) {
	tempDir := setupTestDir(t)
	oldHome := setTempHome(t, tempDir)
	defer restoreHome(oldHome)

	ccm, err := NewCodexConfigManager()
	if err != nil {
		t.Fatalf("NewCodexConfigManager() error = %v", err)
	}

	if ccm == nil {
		t.Fatal("CodexConfigManager should not be nil")
	}

	// 验证路径设置
	expectedConfigPath := filepath.Join(tempDir, ".codex", "config.toml")
	if ccm.configPath != expectedConfigPath {
		t.Errorf("configPath = %v, expected %v", ccm.configPath, expectedConfigPath)
	}

	expectedAuthPath := filepath.Join(tempDir, ".codex", "auth.json")
	if ccm.authPath != expectedAuthPath {
		t.Errorf("authPath = %v, expected %v", ccm.authPath, expectedAuthPath)
	}

	// 验证配置目录是否创建
	configDir := filepath.Dir(ccm.configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Errorf("Config directory %s should be created", configDir)
	}
}

// TestUpdateConfig 测试更新Codex配置文件.
func TestUpdateConfig(t *testing.T) {
	tempDir := setupTestDir(t)
	ccm := createTestCodexConfigManager(t, tempDir)

	testMirror := &MirrorConfig{
		Name:     TestProviderName,
		BaseURL:  TestAPIURL,
		APIKey:   "test-api-key",
		EnvKey:   CodexSwitchAPIKeyEnv,
		ToolType: ToolTypeCodex,
	}

	err := ccm.UpdateConfig(testMirror)
	if err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}

	// 验证配置文件是否存在
	if _, err := os.Stat(ccm.configPath); os.IsNotExist(err) {
		t.Error("Config file should be created")
	}

	// 读取并验证配置内容
	var config CodexConfig
	_, err = toml.DecodeFile(ccm.configPath, &config)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	// 验证基本配置
	if config.ModelProvider != TestProviderName {
		t.Errorf("ModelProvider = %v, expected test-provider", config.ModelProvider)
	}

	if config.Model != TestModelGPT5 {
		t.Errorf("Model = %v, expected gpt-5", config.Model)
	}

	if config.ModelReasoningEffort != TestHighEffort {
		t.Errorf("ModelReasoningEffort = %v, expected high", config.ModelReasoningEffort)
	}

	if !config.DisableResponseStorage {
		t.Error("DisableResponseStorage should be true")
	}

	// 验证模型提供商配置
	if config.ModelProviders == nil {
		t.Fatal("ModelProviders should not be nil")
	}

	provider, exists := config.ModelProviders[TestProviderName]
	if !exists {
		t.Fatal("test-provider should exist in ModelProviders")
	}

	if provider.Name != TestProviderName {
		t.Errorf("Provider Name = %v, expected test-provider", provider.Name)
	}

	if provider.BaseURL != "https://api.test.com" {
		t.Errorf("Provider BaseURL = %v, expected %s", provider.BaseURL, TestAPIURL)
	}

	if provider.EnvKey != CodexSwitchAPIKeyEnv {
		t.Errorf("Provider EnvKey = %v, expected %v", provider.EnvKey, CodexSwitchAPIKeyEnv)
	}

	if provider.WireAPI != TestResponsesDir {
		t.Errorf("Provider WireAPI = %v, expected responses", provider.WireAPI)
	}
}

// TestUpdateConfigExisting 测试更新现有配置文件.
func TestUpdateConfigExisting(t *testing.T) {
	tempDir := setupTestDir(t)
	ccm := createTestCodexConfigManager(t, tempDir)

	// 创建初始配置文件
	initialConfig := CodexConfig{
		ModelProvider:          "initial-provider",
		Model:                  TestModelGPT4,
		ModelReasoningEffort:   "medium",
		DisableResponseStorage: false,
		ModelProviders: map[string]ModelProviderConfig{
			"initial-provider": {
				Name:    "initial-provider",
				BaseURL: "https://api.initial.com",
				WireAPI: "custom-wire",
				EnvKey:  "INITIAL_API_KEY",
			},
		},
	}

	// 保存初始配置
	err := ccm.saveConfig(&initialConfig)
	if err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// 更新配置
	testMirror := &MirrorConfig{
		Name:     "new-provider",
		BaseURL:  "https://api.new.com",
		APIKey:   "new-api-key",
		EnvKey:   CodexSwitchAPIKeyEnv,
		ToolType: ToolTypeCodex,
	}

	err = ccm.UpdateConfig(testMirror)
	if err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}

	// 读取更新后的配置
	var updatedConfig CodexConfig
	_, err = toml.DecodeFile(ccm.configPath, &updatedConfig)
	if err != nil {
		t.Fatalf("Failed to read updated config: %v", err)
	}

	// 验证当前提供商被更新
	if updatedConfig.ModelProvider != "new-provider" {
		t.Errorf("ModelProvider = %v, expected new-provider", updatedConfig.ModelProvider)
	}

	// 验证旧的提供商配置保留
	if _, exists := updatedConfig.ModelProviders["initial-provider"]; !exists {
		t.Error("initial-provider should be preserved")
	}

	// 验证新的提供商配置添加
	newProvider, exists := updatedConfig.ModelProviders["new-provider"]
	if !exists {
		t.Fatal("new-provider should be added")
	}

	if newProvider.BaseURL != "https://api.new.com" {
		t.Errorf("New provider BaseURL = %v, expected https://api.new.com", newProvider.BaseURL)
	}

	// 验证默认值被设置
	if updatedConfig.Model != TestModelGPT5 {
		t.Errorf("Model should be updated to gpt-5, got %v", updatedConfig.Model)
	}

	if !updatedConfig.DisableResponseStorage {
		t.Error("DisableResponseStorage should be forced to true")
	}
}

// TestUpdateAuth 测试更新认证文件.
func TestUpdateAuth(t *testing.T) {
	tempDir := setupTestDir(t)
	ccm := createTestCodexConfigManager(t, tempDir)

	testMirror := &MirrorConfig{
		Name:     TestProviderName,
		BaseURL:  TestAPIURL,
		APIKey:   "test-api-key-12345",
		ToolType: ToolTypeCodex,
	}

	err := ccm.UpdateAuth(testMirror)
	if err != nil {
		t.Fatalf("UpdateAuth() error = %v", err)
	}

	// 验证认证文件是否存在
	if _, err := os.Stat(ccm.authPath); os.IsNotExist(err) {
		t.Error("Auth file should be created")
	}

	// 读取并验证认证文件内容
	file, err := os.Open(ccm.authPath)
	if err != nil {
		t.Fatalf("Failed to open auth file: %v", err)
	}
	defer file.Close()

	var auth CodexAuth
	err = json.NewDecoder(file).Decode(&auth)
	if err != nil {
		t.Fatalf("Failed to decode auth file: %v", err)
	}

	if auth.APIKey != "test-api-key-12345" {
		t.Errorf("APIKey = %v, expected test-api-key-12345", auth.APIKey)
	}
}

// TestGetCurrentConfig 测试获取当前配置.
func TestGetCurrentConfig(t *testing.T) {
	tempDir := setupTestDir(t)
	ccm := createTestCodexConfigManager(t, tempDir)

	// 测试配置文件不存在的情况
	_, err := ccm.GetCurrentConfig()
	if err == nil {
		t.Error("Should error when config file does not exist")
	}

	// 创建测试配置文件
	testConfig := CodexConfig{
		ModelProvider:          TestProviderName,
		Model:                  TestModelGPT4,
		ModelReasoningEffort:   TestHighEffort,
		DisableResponseStorage: true,
		ModelProviders: map[string]ModelProviderConfig{
			TestProviderName: {
				Name:    TestProviderName,
				BaseURL: TestAPIURL,
				WireAPI: TestResponsesDir,
				EnvKey:  CodexSwitchAPIKeyEnv,
			},
		},
	}

	err = ccm.saveConfig(&testConfig)
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// 获取当前配置
	config, err := ccm.GetCurrentConfig()
	if err != nil {
		t.Fatalf("GetCurrentConfig() error = %v", err)
	}

	if config.ModelProvider != TestProviderName {
		t.Errorf("ModelProvider = %v, expected test-provider", config.ModelProvider)
	}

	if config.Model != TestModelGPT4 {
		t.Errorf("Model = %v, expected gpt-4", config.Model)
	}
}

// TestGetCurrentAuth 测试获取当前认证.
func TestGetCurrentAuth(t *testing.T) {
	tempDir := setupTestDir(t)
	ccm := createTestCodexConfigManager(t, tempDir)

	// 测试认证文件不存在的情况
	_, err := ccm.GetCurrentAuth()
	if err == nil {
		t.Error("Should error when auth file does not exist")
	}

	// 创建测试认证文件
	testAuth := CodexAuth{
		APIKey: "test-auth-key-67890",
	}

	file, err := os.Create(ccm.authPath)
	if err != nil {
		t.Fatalf("Failed to create auth file: %v", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(testAuth)
	file.Close()
	if err != nil {
		t.Fatalf("Failed to encode auth file: %v", err)
	}

	// 获取当前认证
	auth, err := ccm.GetCurrentAuth()
	if err != nil {
		t.Fatalf("GetCurrentAuth() error = %v", err)
	}

	if auth.APIKey != "test-auth-key-67890" {
		t.Errorf("APIKey = %v, expected test-auth-key-67890", auth.APIKey)
	}
}

// TestGetCurrentBaseURL 测试获取当前Base URL.
func TestGetCurrentBaseURL(t *testing.T) {
	tempDir := setupTestDir(t)
	ccm := createTestCodexConfigManager(t, tempDir)

	// 创建测试配置
	testConfig := CodexConfig{
		ModelProvider: TestProviderName,
		ModelProviders: map[string]ModelProviderConfig{
			TestProviderName: {
				Name:    TestProviderName,
				BaseURL: TestAPIURL,
				WireAPI: TestResponsesDir,
				EnvKey:  CodexSwitchAPIKeyEnv,
			},
			"other-provider": {
				Name:    "other-provider",
				BaseURL: "https://api.other.com",
				WireAPI: TestResponsesDir,
				EnvKey:  CodexSwitchAPIKeyEnv,
			},
		},
	}

	err := ccm.saveConfig(&testConfig)
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	tests := []struct {
		name         string
		providerName string
		expectedURL  string
		expectError  bool
	}{
		{
			name:         "存在的提供商",
			providerName: TestProviderName,
			expectedURL:  TestAPIURL,
			expectError:  false,
		},
		{
			name:         "其他存在的提供商",
			providerName: "other-provider",
			expectedURL:  "https://api.other.com",
			expectError:  false,
		},
		{
			name:         "不存在的提供商",
			providerName: "nonexistent-provider",
			expectedURL:  "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, err := ccm.GetCurrentBaseURL(tt.providerName)
			if (err != nil) != tt.expectError {
				t.Errorf("GetCurrentBaseURL() error = %v, expectError %v", err, tt.expectError)
			}

			if !tt.expectError && baseURL != tt.expectedURL {
				t.Errorf("BaseURL = %v, expected %v", baseURL, tt.expectedURL)
			}
		})
	}
}

// TestApplyMirror 测试应用镜像源配置.
func TestApplyMirror(t *testing.T) {
	tempDir := setupTestDir(t)
	ccm := createTestCodexConfigManager(t, tempDir)

	testMirror := &MirrorConfig{
		Name:     "apply-test",
		BaseURL:  "https://api.apply.com",
		APIKey:   "apply-test-key",
		EnvKey:   CodexSwitchAPIKeyEnv,
		ToolType: ToolTypeCodex,
	}

	// 由于我们无法模拟SetEnvironmentVariable方法，这里我们跳过环境变量设置的验证
	// 在实际测试中，ApplyMirror会调用SetEnvironmentVariable，但我们主要测试配置文件的更新

	err := ccm.ApplyMirror(testMirror)
	if err != nil {
		t.Fatalf("ApplyMirror() error = %v", err)
	}

	// 验证配置文件是否更新
	config, err := ccm.GetCurrentConfig()
	if err != nil {
		t.Fatalf("Failed to get config after apply: %v", err)
	}

	if config.ModelProvider != "apply-test" {
		t.Errorf("ModelProvider = %v, expected apply-test", config.ModelProvider)
	}

	provider, exists := config.ModelProviders["apply-test"]
	if !exists {
		t.Fatal("apply-test provider should exist")
	}

	if provider.BaseURL != "https://api.apply.com" {
		t.Errorf("Provider BaseURL = %v, expected https://api.apply.com", provider.BaseURL)
	}

	// 验证认证文件是否更新
	auth, err := ccm.GetCurrentAuth()
	if err != nil {
		t.Fatalf("Failed to get auth after apply: %v", err)
	}

	if auth.APIKey != "apply-test-key" {
		t.Errorf("Auth APIKey = %v, expected apply-test-key", auth.APIKey)
	}
}

// TestFixEnvKeyFormat 测试修复环境变量key格式.
func TestFixEnvKeyFormat(t *testing.T) {
	tempDir := setupTestDir(t)
	ccm := createTestCodexConfigManager(t, tempDir)

	// 创建有错误env_key格式的配置
	testConfig := CodexConfig{
		ModelProvider: TestProviderName,
		ModelProviders: map[string]ModelProviderConfig{
			"provider1": {
				Name:    "provider1",
				BaseURL: "https://api.test1.com",
				WireAPI: TestResponsesDir,
				EnvKey:  "WRONG_ENV_KEY", // 错误的格式
			},
			"provider2": {
				Name:    "provider2",
				BaseURL: "https://api.test2.com",
				WireAPI: TestResponsesDir,
				EnvKey:  CodexSwitchAPIKeyEnv, // 正确的格式
			},
			"provider3": {
				Name:    "provider3",
				BaseURL: "https://api.test3.com",
				WireAPI: TestResponsesDir,
				EnvKey:  "", // 空的env_key
			},
		},
	}

	err := ccm.saveConfig(&testConfig)
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// 执行修复
	err = ccm.FixEnvKeyFormat()
	if err != nil {
		t.Fatalf("FixEnvKeyFormat() error = %v", err)
	}

	// 验证修复结果
	config, err := ccm.GetCurrentConfig()
	if err != nil {
		t.Fatalf("Failed to get config after fix: %v", err)
	}

	// 检查所有提供商的env_key是否都被修复为正确格式
	for name, provider := range config.ModelProviders {
		if provider.EnvKey != CodexSwitchAPIKeyEnv {
			t.Errorf("Provider %s EnvKey = %v, expected %v", name, provider.EnvKey, CodexSwitchAPIKeyEnv)
		}
	}
}

// TestBackupConfig 测试备份配置.
func TestBackupConfig(t *testing.T) {
	tempDir := setupTestDir(t)
	ccm := createTestCodexConfigManager(t, tempDir)

	// 创建测试配置和认证文件
	testConfig := CodexConfig{
		ModelProvider: "backup-test",
		Model:         TestModelGPT4,
	}

	err := ccm.saveConfig(&testConfig)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	testAuth := CodexAuth{
		APIKey: "backup-test-key",
	}

	authFile, err := os.Create(ccm.authPath)
	if err != nil {
		t.Fatalf("Failed to create test auth file: %v", err)
	}
	json.NewEncoder(authFile).Encode(testAuth)
	authFile.Close()

	// 执行备份
	err = ccm.BackupConfig()
	if err != nil {
		t.Fatalf("BackupConfig() error = %v", err)
	}

	// 验证备份文件是否存在
	configDir := filepath.Dir(ccm.configPath)
	backupConfigPath := filepath.Join(configDir, "backup", "config.toml.bak")
	backupAuthPath := filepath.Join(configDir, "backup", "auth.json.bak")

	if _, err := os.Stat(backupConfigPath); os.IsNotExist(err) {
		t.Error("Backup config file should exist")
	}

	if _, err := os.Stat(backupAuthPath); os.IsNotExist(err) {
		t.Error("Backup auth file should exist")
	}

	// 验证备份内容
	var backupConfig CodexConfig
	_, err = toml.DecodeFile(backupConfigPath, &backupConfig)
	if err != nil {
		t.Fatalf("Failed to read backup config: %v", err)
	}

	if backupConfig.ModelProvider != "backup-test" {
		t.Errorf("Backup config ModelProvider = %v, expected backup-test", backupConfig.ModelProvider)
	}
}

// TestCopyFile 测试文件复制功能.
func TestCopyFile(t *testing.T) {
	tempDir := setupTestDir(t)

	// 创建源文件
	srcPath := filepath.Join(tempDir, "source.txt")
	srcContent := "This is test content for copy file test"
	err := os.WriteFile(srcPath, []byte(srcContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// 复制文件
	dstPath := filepath.Join(tempDir, "destination.txt")
	err = copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	// 验证目标文件存在
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Error("Destination file should exist")
	}

	// 验证文件内容
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != srcContent {
		t.Errorf("Destination content = %v, expected %v", string(dstContent), srcContent)
	}
}

// TestSetEnvironmentVariableValidation 测试环境变量设置的参数验证.
func TestSetEnvironmentVariableValidation(t *testing.T) {
	tempDir := setupTestDir(t)
	ccm := createTestCodexConfigManager(t, tempDir)

	// 测试空环境变量key
	err := ccm.SetEnvironmentVariable("", "test-value")
	if err == nil {
		t.Error("Should error when env key is empty")
	}

	// 测试有效参数但不执行实际的系统调用
	// 我们只能测试参数验证部分，不能测试实际的系统调用
}

// createTestCodexConfigManager 创建用于测试的Codex配置管理器.
func createTestCodexConfigManager(t *testing.T, tempDir string) *CodexConfigManager {
	oldHome := setTempHome(t, tempDir)
	t.Cleanup(func() { restoreHome(oldHome) })

	// 创建配置目录
	configDir := filepath.Join(tempDir, ".codex")
	if err := EnsureDir(configDir); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	authPath := filepath.Join(configDir, "auth.json")

	return &CodexConfigManager{
		configPath: configPath,
		authPath:   authPath,
	}
}

// TestCodexConfigStructures 测试配置结构体的JSON/TOML兼容性.
func TestCodexConfigStructures(t *testing.T) {
	// 测试CodexConfig结构体
	config := CodexConfig{
		ModelProvider:          TestProviderName,
		Model:                  TestModelGPT4,
		ModelReasoningEffort:   TestHighEffort,
		DisableResponseStorage: true,
		ModelProviders: map[string]ModelProviderConfig{
			"provider1": {
				Name:    "provider1",
				BaseURL: TestAPIURL,
				WireAPI: TestResponsesDir,
				EnvKey:  CodexSwitchAPIKeyEnv,
			},
		},
	}

	// 测试TOML编码
	tempDir := setupTestDir(t)
	configPath := filepath.Join(tempDir, "test-config.toml")

	file, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	encoder := toml.NewEncoder(file)
	err = encoder.Encode(config)
	file.Close()
	if err != nil {
		t.Fatalf("Failed to encode config to TOML: %v", err)
	}

	// 测试TOML解码
	var decodedConfig CodexConfig
	_, err = toml.DecodeFile(configPath, &decodedConfig)
	if err != nil {
		t.Fatalf("Failed to decode TOML config: %v", err)
	}

	if decodedConfig.ModelProvider != config.ModelProvider {
		t.Errorf("Decoded ModelProvider = %v, expected %v", decodedConfig.ModelProvider, config.ModelProvider)
	}

	// 测试CodexAuth结构体
	auth := CodexAuth{
		APIKey: "test-api-key-json",
	}

	// 测试JSON编码
	authPath := filepath.Join(tempDir, "test-auth.json")
	authFile, err := os.Create(authPath)
	if err != nil {
		t.Fatalf("Failed to create auth file: %v", err)
	}

	jsonEncoder := json.NewEncoder(authFile)
	err = jsonEncoder.Encode(auth)
	authFile.Close()
	if err != nil {
		t.Fatalf("Failed to encode auth to JSON: %v", err)
	}

	// 测试JSON解码
	authReadFile, err := os.Open(authPath)
	if err != nil {
		t.Fatalf("Failed to open auth file: %v", err)
	}
	defer authReadFile.Close()

	var decodedAuth CodexAuth
	err = json.NewDecoder(authReadFile).Decode(&decodedAuth)
	if err != nil {
		t.Fatalf("Failed to decode JSON auth: %v", err)
	}

	if decodedAuth.APIKey != auth.APIKey {
		t.Errorf("Decoded APIKey = %v, expected %v", decodedAuth.APIKey, auth.APIKey)
	}
}
