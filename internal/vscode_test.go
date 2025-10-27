package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestNewVSCodeConfigManager 测试创建VS Code配置管理器.
func TestNewVSCodeConfigManager(t *testing.T) {
	tempDir := setupTestDir(t)
	oldHome := setTempHome(t, tempDir)
	defer restoreHome(oldHome)

	vcm, err := NewVSCodeConfigManager()
	if err != nil {
		t.Fatalf("NewVSCodeConfigManager() error = %v", err)
	}

	if vcm == nil {
		t.Fatal("VSCodeConfigManager should not be nil")
	}

	// 验证设置路径
	platform := GetCurrentPlatform()
	var expectedPath string
	switch platform {
	case PlatformWindows:
		expectedPath = filepath.Join(tempDir, "AppData", "Roaming", "Code", "User", "settings.json")
	case PlatformMac:
		expectedPath = filepath.Join(tempDir, "Library", "Application Support", "Code", "User", "settings.json")
	case PlatformLinux:
		expectedPath = filepath.Join(tempDir, ".config", "Code", "User", "settings.json")
	}

	if vcm.settingsPath != expectedPath {
		t.Errorf("settingsPath = %v, expected %v", vcm.settingsPath, expectedPath)
	}

	// 验证配置目录是否创建
	configDir := filepath.Dir(vcm.settingsPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Errorf("Config directory %s should be created", configDir)
	}
}

// TestLoadSettings 测试加载VS Code设置.
func TestLoadSettings(t *testing.T) {
	tempDir := setupTestDir(t)
	vcm := createTestVSCodeConfigManager(t, tempDir)

	// 测试加载不存在的设置文件
	settings, err := vcm.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() should not error when file doesn't exist: %v", err)
	}

	if settings == nil {
		t.Fatal("Settings should not be nil")
	}

	if len(settings) != 0 {
		t.Errorf("Settings should be empty when file doesn't exist, got %d items", len(settings))
	}

	// 创建测试设置文件
	testSettings := map[string]interface{}{
		"editor.fontSize": 14,
		"editor.tabSize":  4,
		"chatgpt.apiBase": "https://api.test.com",
		"chatgpt.config": map[string]interface{}{
			"model":                  "gpt-4",
			"preferred_auth_method":  "apikey",
			"model_reasoning_effort": "medium",
		},
		"other.setting": "value",
	}

	err = vcm.SaveSettings(testSettings)
	if err != nil {
		t.Fatalf("Failed to save test settings: %v", err)
	}

	// 测试加载现有设置文件
	loadedSettings, err := vcm.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	// 验证基本设置
	if fontSize, ok := loadedSettings["editor.fontSize"]; !ok || fontSize != float64(14) {
		t.Errorf("editor.fontSize = %v, expected 14", fontSize)
	}

	if tabSize, ok := loadedSettings["editor.tabSize"]; !ok || tabSize != float64(4) {
		t.Errorf("editor.tabSize = %v, expected 4", tabSize)
	}

	// 验证ChatGPT设置
	if apiBase, ok := loadedSettings["chatgpt.apiBase"]; !ok || apiBase != "https://api.test.com" {
		t.Errorf("chatgpt.apiBase = %v, expected https://api.test.com", apiBase)
	}

	// 验证嵌套配置
	if config, ok := loadedSettings["chatgpt.config"]; ok {
		if configMap, ok := config.(map[string]interface{}); ok {
			if model, ok := configMap["model"]; !ok || model != "gpt-4" {
				t.Errorf("chatgpt.config.model = %v, expected gpt-4", model)
			}
		} else {
			t.Error("chatgpt.config should be a map")
		}
	} else {
		t.Error("chatgpt.config should exist")
	}
}

// TestSaveSettings 测试保存VS Code设置.
func TestSaveSettings(t *testing.T) {
	tempDir := setupTestDir(t)
	vcm := createTestVSCodeConfigManager(t, tempDir)

	testSettings := map[string]interface{}{
		"editor.fontSize": 16,
		"editor.theme":    "dark",
		"chatgpt.apiBase": "https://api.save.com",
		"chatgpt.config": map[string]interface{}{
			"model":                 "gpt-5",
			"preferred_auth_method": "apikey",
			"wire_api":              "responses",
		},
		"nested": map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": "deep_value",
			},
		},
	}

	err := vcm.SaveSettings(testSettings)
	if err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	// 验证文件是否存在
	if _, err := os.Stat(vcm.settingsPath); os.IsNotExist(err) {
		t.Error("Settings file should be created")
	}

	// 直接读取文件验证JSON格式
	file, err := os.Open(vcm.settingsPath)
	if err != nil {
		t.Fatalf("Failed to open settings file: %v", err)
	}
	defer file.Close()

	var savedSettings map[string]interface{}
	err = json.NewDecoder(file).Decode(&savedSettings)
	if err != nil {
		t.Fatalf("Failed to decode saved settings: %v", err)
	}

	// 验证保存的内容
	if fontSize, ok := savedSettings["editor.fontSize"]; !ok || fontSize != float64(16) {
		t.Errorf("Saved editor.fontSize = %v, expected 16", fontSize)
	}

	if theme, ok := savedSettings["editor.theme"]; !ok || theme != "dark" {
		t.Errorf("Saved editor.theme = %v, expected dark", theme)
	}

	if apiBase, ok := savedSettings["chatgpt.apiBase"]; !ok || apiBase != "https://api.save.com" {
		t.Errorf("Saved chatgpt.apiBase = %v, expected https://api.save.com", apiBase)
	}

	// 验证嵌套结构
	if nested, ok := savedSettings["nested"]; ok {
		if nestedMap, ok := nested.(map[string]interface{}); ok {
			if level1, ok := nestedMap["level1"]; ok {
				if level1Map, ok := level1.(map[string]interface{}); ok {
					if level2, ok := level1Map["level2"]; !ok || level2 != "deep_value" {
						t.Errorf("nested.level1.level2 = %v, expected deep_value", level2)
					}
				} else {
					t.Error("nested.level1 should be a map")
				}
			} else {
				t.Error("nested.level1 should exist")
			}
		} else {
			t.Error("nested should be a map")
		}
	} else {
		t.Error("nested should exist")
	}
}

// TestVSCodeApplyMirror 测试应用镜像源配置.
func TestVSCodeApplyMirror(t *testing.T) {
	tempDir := setupTestDir(t)
	vcm := createTestVSCodeConfigManager(t, tempDir)

	// 先创建一些现有设置
	existingSettings := map[string]interface{}{
		"editor.fontSize": 12,
		"editor.theme":    "light",
		"chatgpt.config": map[string]interface{}{
			"old_setting": "should_be_preserved",
			"apiKey":      "should_be_removed",
			"apiBaseUrl":  "should_be_removed",
			"model":       "gpt-3", // 应该被覆盖
		},
		"other.setting": "should_remain",
	}

	err := vcm.SaveSettings(existingSettings)
	if err != nil {
		t.Fatalf("Failed to save existing settings: %v", err)
	}

	// 应用镜像源配置
	testMirror := &MirrorConfig{
		Name:     "apply-test",
		BaseURL:  "https://api.apply.com",
		APIKey:   "apply-test-key",
		ToolType: ToolTypeCodex,
	}

	err = vcm.ApplyMirror(testMirror)
	if err != nil {
		t.Fatalf("ApplyMirror() error = %v", err)
	}

	// 验证应用结果
	settings, err := vcm.LoadSettings()
	if err != nil {
		t.Fatalf("Failed to load settings after apply: %v", err)
	}

	// 验证apiBase被更新
	if apiBase, ok := settings["chatgpt.apiBase"]; !ok || apiBase != "https://api.apply.com" {
		t.Errorf("chatgpt.apiBase = %v, expected https://api.apply.com", apiBase)
	}

	// 验证其他设置保持不变
	if fontSize, ok := settings["editor.fontSize"]; !ok || fontSize != float64(12) {
		t.Errorf("editor.fontSize = %v, expected 12 (should be preserved)", fontSize)
	}

	if otherSetting, ok := settings["other.setting"]; !ok || otherSetting != "should_remain" {
		t.Errorf("other.setting = %v, expected should_remain", otherSetting)
	}

	// 验证chatgpt.config被正确更新
	if config, ok := settings["chatgpt.config"]; ok {
		if configMap, ok := config.(map[string]interface{}); ok {
			// 验证新设置
			if authMethod, ok := configMap["preferred_auth_method"]; !ok || authMethod != "apikey" {
				t.Errorf("preferred_auth_method = %v, expected apikey", authMethod)
			}

			if model, ok := configMap["model"]; !ok || model != "gpt-5" {
				t.Errorf("model = %v, expected gpt-5", model)
			}

			if reasoningEffort, ok := configMap["model_reasoning_effort"]; !ok || reasoningEffort != "high" {
				t.Errorf("model_reasoning_effort = %v, expected high", reasoningEffort)
			}

			if wireAPI, ok := configMap["wire_api"]; !ok || wireAPI != "responses" {
				t.Errorf("wire_api = %v, expected responses", wireAPI)
			}

			// 验证旧设置被保留
			if oldSetting, ok := configMap["old_setting"]; !ok || oldSetting != "should_be_preserved" {
				t.Errorf("old_setting = %v, expected should_be_preserved", oldSetting)
			}

			// 验证不需要的设置被移除
			if _, ok := configMap["apiKey"]; ok {
				t.Error("apiKey should be removed")
			}

			if _, ok := configMap["apiBaseUrl"]; ok {
				t.Error("apiBaseUrl should be removed")
			}
		} else {
			t.Error("chatgpt.config should be a map")
		}
	} else {
		t.Error("chatgpt.config should exist")
	}
}

// TestVSCodeGetCurrentConfig 测试获取当前配置.
func TestVSCodeGetCurrentConfig(t *testing.T) {
	tempDir := setupTestDir(t)
	vcm := createTestVSCodeConfigManager(t, tempDir)

	// 测试空配置
	config, err := vcm.GetCurrentConfig()
	if err != nil {
		t.Fatalf("GetCurrentConfig() error = %v", err)
	}

	if len(config) != 0 {
		t.Errorf("Config should be empty when no settings exist, got %d items", len(config))
	}

	// 创建包含ChatGPT配置的设置
	testSettings := map[string]interface{}{
		"editor.fontSize": 14,
		"chatgpt.apiBase": "https://api.current.com",
		"chatgpt.config": map[string]interface{}{
			"model":                  "gpt-4",
			"preferred_auth_method":  "apikey",
			"model_reasoning_effort": "high",
			"wire_api":               "responses",
		},
		"other.setting": "value",
	}

	err = vcm.SaveSettings(testSettings)
	if err != nil {
		t.Fatalf("Failed to save test settings: %v", err)
	}

	// 获取当前配置
	config, err = vcm.GetCurrentConfig()
	if err != nil {
		t.Fatalf("GetCurrentConfig() error = %v", err)
	}

	// 验证返回的配置只包含ChatGPT相关设置
	if apiBase, ok := config["apiBase"]; !ok || apiBase != "https://api.current.com" {
		t.Errorf("apiBase = %v, expected https://api.current.com", apiBase)
	}

	if configData, ok := config["config"]; ok {
		if configMap, ok := configData.(map[string]interface{}); ok {
			if model, ok := configMap["model"]; !ok || model != "gpt-4" {
				t.Errorf("config.model = %v, expected gpt-4", model)
			}
		} else {
			t.Error("config should be a map")
		}
	} else {
		t.Error("config should exist")
	}

	// 验证非ChatGPT设置不被包含
	if _, ok := config["editor.fontSize"]; ok {
		t.Error("editor.fontSize should not be included in ChatGPT config")
	}

	if _, ok := config["other.setting"]; ok {
		t.Error("other.setting should not be included in ChatGPT config")
	}
}

// TestBackupSettings 测试备份设置.
func TestBackupSettings(t *testing.T) {
	tempDir := setupTestDir(t)
	vcm := createTestVSCodeConfigManager(t, tempDir)

	// 测试备份不存在的设置文件
	err := vcm.BackupSettings()
	if err != nil {
		t.Errorf("BackupSettings() should not error when settings file doesn't exist: %v", err)
	}

	// 创建测试设置文件
	testSettings := map[string]interface{}{
		"backup.test":     "backup_value",
		"editor.fontSize": 16,
	}

	err = vcm.SaveSettings(testSettings)
	if err != nil {
		t.Fatalf("Failed to save test settings: %v", err)
	}

	// 执行备份
	err = vcm.BackupSettings()
	if err != nil {
		t.Fatalf("BackupSettings() error = %v", err)
	}

	// 验证备份文件是否存在
	configDir := filepath.Dir(vcm.settingsPath)
	backupPath := filepath.Join(configDir, "backup", "settings.json.bak")

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file should exist")
	}

	// 验证备份文件内容
	backupFile, err := os.Open(backupPath)
	if err != nil {
		t.Fatalf("Failed to open backup file: %v", err)
	}
	defer backupFile.Close()

	var backupSettings map[string]interface{}
	err = json.NewDecoder(backupFile).Decode(&backupSettings)
	if err != nil {
		t.Fatalf("Failed to decode backup file: %v", err)
	}

	if backupTest, ok := backupSettings["backup.test"]; !ok || backupTest != "backup_value" {
		t.Errorf("Backup backup.test = %v, expected backup_value", backupTest)
	}

	if fontSize, ok := backupSettings["editor.fontSize"]; !ok || fontSize != float64(16) {
		t.Errorf("Backup editor.fontSize = %v, expected 16", fontSize)
	}
}

// TestRemoveChatGPTConfig 测试移除ChatGPT配置.
func TestRemoveChatGPTConfig(t *testing.T) {
	tempDir := setupTestDir(t)
	vcm := createTestVSCodeConfigManager(t, tempDir)

	// 创建包含ChatGPT配置的设置
	testSettings := map[string]interface{}{
		"editor.fontSize": 14,
		"editor.theme":    "dark",
		"chatgpt.apiBase": "https://api.remove.com",
		"chatgpt.config": map[string]interface{}{
			"model":                 "gpt-4",
			"preferred_auth_method": "apikey",
		},
		"other.extension.setting": "should_remain",
	}

	err := vcm.SaveSettings(testSettings)
	if err != nil {
		t.Fatalf("Failed to save test settings: %v", err)
	}

	// 移除ChatGPT配置
	err = vcm.RemoveChatGPTConfig()
	if err != nil {
		t.Fatalf("RemoveChatGPTConfig() error = %v", err)
	}

	// 验证ChatGPT配置被移除
	settings, err := vcm.LoadSettings()
	if err != nil {
		t.Fatalf("Failed to load settings after removal: %v", err)
	}

	// ChatGPT设置应该被移除
	if _, ok := settings["chatgpt.apiBase"]; ok {
		t.Error("chatgpt.apiBase should be removed")
	}

	if _, ok := settings["chatgpt.config"]; ok {
		t.Error("chatgpt.config should be removed")
	}

	// 其他设置应该保留
	if fontSize, ok := settings["editor.fontSize"]; !ok || fontSize != float64(14) {
		t.Errorf("editor.fontSize = %v, expected 14 (should be preserved)", fontSize)
	}

	if theme, ok := settings["editor.theme"]; !ok || theme != "dark" {
		t.Errorf("editor.theme = %v, expected dark (should be preserved)", theme)
	}

	if otherSetting, ok := settings["other.extension.setting"]; !ok || otherSetting != "should_remain" {
		t.Errorf("other.extension.setting = %v, expected should_remain", otherSetting)
	}
}

// TestLoadSettingsCorruptedFile 测试加载损坏的设置文件.
func TestLoadSettingsCorruptedFile(t *testing.T) {
	tempDir := setupTestDir(t)
	vcm := createTestVSCodeConfigManager(t, tempDir)

	// 创建损坏的JSON文件
	err := os.WriteFile(vcm.settingsPath, []byte("{ invalid json content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}

	// 尝试加载损坏的文件
	_, err = vcm.LoadSettings()
	if err == nil {
		t.Error("LoadSettings() should error when file is corrupted")
	}
}

// TestSettingsJSONFormatting 测试设置文件的JSON格式化.
func TestSettingsJSONFormatting(t *testing.T) {
	tempDir := setupTestDir(t)
	vcm := createTestVSCodeConfigManager(t, tempDir)

	testSettings := map[string]interface{}{
		"simple":  "value",
		"number":  42,
		"boolean": true,
		"array":   []interface{}{"item1", "item2", 3},
		"nested": map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": "deep_value",
			},
		},
	}

	err := vcm.SaveSettings(testSettings)
	if err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	// 读取原始文件内容检查格式化
	content, err := os.ReadFile(vcm.settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	contentStr := string(content)

	// 验证JSON是否格式化（包含换行和缩进）
	if !containsFormatting(contentStr) {
		t.Error("JSON should be formatted with indentation")
	}

	// 验证JSON是否有效
	var validationSettings map[string]interface{}
	err = json.Unmarshal(content, &validationSettings)
	if err != nil {
		t.Fatalf("Saved JSON should be valid: %v", err)
	}
}

// containsFormatting 检查JSON字符串是否包含格式化.
func containsFormatting(jsonStr string) bool {
	return len(jsonStr) > 10 && // 基本长度检查
		(jsonStr[1] == '\n' || jsonStr[2] == ' ') // 检查是否有缩进或换行
}

// createTestVSCodeConfigManager 创建用于测试的VS Code配置管理器.
func createTestVSCodeConfigManager(t *testing.T, tempDir string) *VSCodeConfigManager {
	oldHome := setTempHome(t, tempDir)
	t.Cleanup(func() { restoreHome(oldHome) })

	// 创建VS Code配置目录
	platform := GetCurrentPlatform()
	var configDir string
	switch platform {
	case PlatformWindows:
		configDir = filepath.Join(tempDir, "AppData", "Roaming", "Code", "User")
	case PlatformMac:
		configDir = filepath.Join(tempDir, "Library", "Application Support", "Code", "User")
	case PlatformLinux:
		configDir = filepath.Join(tempDir, ".config", "Code", "User")
	}

	if err := EnsureDir(configDir); err != nil {
		t.Fatalf("Failed to create VS Code config directory: %v", err)
	}

	settingsPath := filepath.Join(configDir, "settings.json")

	return &VSCodeConfigManager{
		settingsPath: settingsPath,
	}
}

// TestApplyMirrorEmptySettings 测试在空设置上应用镜像源.
func TestApplyMirrorEmptySettings(t *testing.T) {
	tempDir := setupTestDir(t)
	vcm := createTestVSCodeConfigManager(t, tempDir)

	testMirror := &MirrorConfig{
		Name:     "empty-test",
		BaseURL:  "https://api.empty.com",
		APIKey:   "empty-test-key",
		ToolType: ToolTypeCodex,
	}

	err := vcm.ApplyMirror(testMirror)
	if err != nil {
		t.Fatalf("ApplyMirror() on empty settings error = %v", err)
	}

	// 验证配置被正确创建
	settings, err := vcm.LoadSettings()
	if err != nil {
		t.Fatalf("Failed to load settings: %v", err)
	}

	if apiBase, ok := settings["chatgpt.apiBase"]; !ok || apiBase != "https://api.empty.com" {
		t.Errorf("chatgpt.apiBase = %v, expected https://api.empty.com", apiBase)
	}

	if config, ok := settings["chatgpt.config"]; ok {
		if configMap, ok := config.(map[string]interface{}); ok {
			if model, ok := configMap["model"]; !ok || model != "gpt-5" {
				t.Errorf("model = %v, expected gpt-5", model)
			}
		} else {
			t.Error("chatgpt.config should be a map")
		}
	} else {
		t.Error("chatgpt.config should be created")
	}
}
