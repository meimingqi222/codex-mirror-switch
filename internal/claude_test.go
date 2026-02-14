package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewClaudeConfigManager(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 保存原始 HOME 环境变量
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("USERPROFILE", originalUserProfile)
	}()

	// 设置临时目录为 HOME
	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)

	ccm, err := NewClaudeConfigManager()
	if err != nil {
		t.Fatalf("NewClaudeConfigManager failed: %v", err)
	}

	// 验证配置路径
	expectedPath := filepath.Join(tempDir, ".claude", "settings.json")
	if ccm.settingsPath != expectedPath {
		t.Errorf("settingsPath = %v, expected %v", ccm.settingsPath, expectedPath)
	}

	// 验证配置目录已创建
	configDir := filepath.Dir(ccm.settingsPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("Claude config directory should be created")
	}
}

func TestClaudeConfigManager_LoadSettings_Empty(t *testing.T) {
	tempDir := t.TempDir()

	ccm := &ClaudeConfigManager{
		settingsPath: filepath.Join(tempDir, ".claude", "settings.json"),
	}

	// 文件不存在时应返回空的 settings
	settings, err := ccm.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings failed: %v", err)
	}

	if settings.Env == nil {
		t.Error("Env should not be nil")
	}

	if len(settings.Env) != 0 {
		t.Errorf("Env should be empty, got %d entries", len(settings.Env))
	}
}

func TestClaudeConfigManager_LoadSettings_ExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	settingsPath := filepath.Join(configDir, "settings.json")

	// 创建测试配置文件
	testSettings := map[string]interface{}{
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL":   "https://api.test.com",
			"ANTHROPIC_AUTH_TOKEN": "test-token",
			"ANTHROPIC_MODEL":      "claude-3-opus",
		},
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(npm:*)"},
		},
		"customField": "customValue",
	}

	data, _ := json.MarshalIndent(testSettings, "", "  ")
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatalf("Failed to write test settings: %v", err)
	}

	ccm := &ClaudeConfigManager{
		settingsPath: settingsPath,
	}

	settings, err := ccm.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings failed: %v", err)
	}

	// 验证 env 字段
	if settings.Env["ANTHROPIC_BASE_URL"] != TestAPIURL {
		t.Errorf("ANTHROPIC_BASE_URL = %v, expected %s", settings.Env["ANTHROPIC_BASE_URL"], TestAPIURL)
	}
	if settings.Env["ANTHROPIC_AUTH_TOKEN"] != "test-token" {
		t.Errorf("ANTHROPIC_AUTH_TOKEN = %v, expected test-token", settings.Env["ANTHROPIC_AUTH_TOKEN"])
	}
	if settings.Env["ANTHROPIC_MODEL"] != "claude-3-opus" {
		t.Errorf("ANTHROPIC_MODEL = %v, expected claude-3-opus", settings.Env["ANTHROPIC_MODEL"])
	}

	// 验证 permissions 字段
	if settings.Permissions == nil {
		t.Error("Permissions should not be nil")
	}

	// 验证其他字段被保留
	if settings.OtherSettings["customField"] != "customValue" {
		t.Errorf("customField = %v, expected customValue", settings.OtherSettings["customField"])
	}
}

func TestClaudeConfigManager_SaveSettings(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	settingsPath := filepath.Join(configDir, "settings.json")
	ccm := &ClaudeConfigManager{
		settingsPath: settingsPath,
	}

	settings := &ClaudeSettings{
		Env: map[string]string{
			"ANTHROPIC_BASE_URL":   "https://api.new.com",
			"ANTHROPIC_AUTH_TOKEN": "new-token",
		},
		Permissions: map[string]interface{}{
			"allow": []interface{}{"Bash(go:*)"},
		},
		OtherSettings: map[string]interface{}{
			"customField": "customValue",
		},
	}

	if err := ccm.SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	// 验证文件已创建
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Error("Settings file should be created")
	}

	// 重新加载验证
	loadedSettings, err := ccm.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings after save failed: %v", err)
	}

	if loadedSettings.Env["ANTHROPIC_BASE_URL"] != "https://api.new.com" {
		t.Errorf("ANTHROPIC_BASE_URL = %v, expected https://api.new.com", loadedSettings.Env["ANTHROPIC_BASE_URL"])
	}
}

func TestClaudeConfigManager_ApplyMirror(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	settingsPath := filepath.Join(configDir, "settings.json")

	// 创建现有配置（模拟已有配置）
	existingSettings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(npm:*)"},
		},
		"customField": "shouldBePreserved",
	}
	data, _ := json.MarshalIndent(existingSettings, "", "  ")
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatalf("Failed to write existing settings: %v", err)
	}

	ccm := &ClaudeConfigManager{
		settingsPath: settingsPath,
	}

	mirror := &MirrorConfig{
		Name:      "test-mirror",
		BaseURL:   "https://api.mirror.com",
		APIKey:    "mirror-key",
		ModelName: "claude-3-sonnet",
		ToolType:  ToolTypeClaude,
	}

	if err := ccm.ApplyMirror(mirror); err != nil {
		t.Fatalf("ApplyMirror failed: %v", err)
	}

	// 验证配置已更新
	settings, err := ccm.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings after ApplyMirror failed: %v", err)
	}

	// 验证 env 字段
	if settings.Env["ANTHROPIC_BASE_URL"] != "https://api.mirror.com" {
		t.Errorf("ANTHROPIC_BASE_URL = %v, expected https://api.mirror.com", settings.Env["ANTHROPIC_BASE_URL"])
	}
	if settings.Env["ANTHROPIC_AUTH_TOKEN"] != "mirror-key" {
		t.Errorf("ANTHROPIC_AUTH_TOKEN = %v, expected mirror-key", settings.Env["ANTHROPIC_AUTH_TOKEN"])
	}
	if settings.Env["ANTHROPIC_MODEL"] != "claude-3-sonnet" {
		t.Errorf("ANTHROPIC_MODEL = %v, expected claude-3-sonnet", settings.Env["ANTHROPIC_MODEL"])
	}

	// 验证现有字段被保留
	if settings.OtherSettings["customField"] != "shouldBePreserved" {
		t.Errorf("customField = %v, expected shouldBePreserved", settings.OtherSettings["customField"])
	}

	// 验证 permissions 被保留
	if settings.Permissions == nil {
		t.Error("Permissions should be preserved")
	}
}

func TestClaudeConfigManager_ApplyMirror_ClearModel(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	settingsPath := filepath.Join(configDir, "settings.json")

	// 创建现有配置（包含 ANTHROPIC_MODEL）
	existingSettings := map[string]interface{}{
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL":   "https://old.api.com",
			"ANTHROPIC_AUTH_TOKEN": "old-token",
			"ANTHROPIC_MODEL":      "old-model",
		},
	}
	data, _ := json.MarshalIndent(existingSettings, "", "  ")
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatalf("Failed to write existing settings: %v", err)
	}

	ccm := &ClaudeConfigManager{
		settingsPath: settingsPath,
	}

	// 应用不包含模型名称的镜像
	mirror := &MirrorConfig{
		Name:      "test-mirror",
		BaseURL:   "https://api.new.com",
		APIKey:    "new-key",
		ModelName: "", // 空模型名称
		ToolType:  ToolTypeClaude,
	}

	if err := ccm.ApplyMirror(mirror); err != nil {
		t.Fatalf("ApplyMirror failed: %v", err)
	}

	// 验证 ANTHROPIC_MODEL 已被清除
	settings, err := ccm.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings after ApplyMirror failed: %v", err)
	}

	if _, exists := settings.Env["ANTHROPIC_MODEL"]; exists {
		t.Error("ANTHROPIC_MODEL should be cleared when ModelName is empty")
	}
}

func TestClaudeConfigManager_BackupSettings(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	settingsPath := filepath.Join(configDir, "settings.json")

	// 创建测试配置文件
	testContent := `{"env": {"TEST": "value"}}`
	if err := os.WriteFile(settingsPath, []byte(testContent), 0o644); err != nil {
		t.Fatalf("Failed to write test settings: %v", err)
	}

	ccm := &ClaudeConfigManager{
		settingsPath: settingsPath,
	}

	if err := ccm.BackupSettings(); err != nil {
		t.Fatalf("BackupSettings failed: %v", err)
	}

	// 验证备份文件存在
	backupPath := filepath.Join(configDir, "backup", "settings.json.bak")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file should exist")
	}

	// 验证备份内容
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if string(backupContent) != testContent {
		t.Errorf("Backup content = %v, expected %v", string(backupContent), testContent)
	}
}

func TestClaudeConfigManager_ApplyMirror_WithExtraEnv(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	settingsPath := filepath.Join(configDir, "settings.json")

	// 创建现有配置
	existingSettings := map[string]interface{}{
		"model": "opus",
		"env": map[string]interface{}{
			"EXISTING_VAR": "should-be-preserved",
		},
	}
	data, _ := json.MarshalIndent(existingSettings, "", "  ")
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatalf("Failed to write existing settings: %v", err)
	}

	ccm := &ClaudeConfigManager{
		settingsPath: settingsPath,
	}

	mirror := &MirrorConfig{
		Name:      "test-mirror",
		BaseURL:   "https://api.proxy.com",
		APIKey:    "test-key",
		ModelName: "claude-3-opus",
		ToolType:  ToolTypeClaude,
		ExtraEnv: map[string]string{
			"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "gemini-2.5-flash-lite",
			"ANTHROPIC_DEFAULT_SONNET_MODEL": "gemini-claude-sonnet-4-5-thinking",
			"ANTHROPIC_DEFAULT_OPUS_MODEL":   "gemini-claude-opus-4-5-thinking",
			"API_TIMEOUT_MS":                 "3000000",
		},
	}

	if err := ccm.ApplyMirror(mirror); err != nil {
		t.Fatalf("ApplyMirror failed: %v", err)
	}

	// 验证配置已更新
	settings, err := ccm.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings after ApplyMirror failed: %v", err)
	}

	// 验证基础 env 字段
	if settings.Env["ANTHROPIC_BASE_URL"] != "https://api.proxy.com" {
		t.Errorf("ANTHROPIC_BASE_URL = %v, expected https://api.proxy.com", settings.Env["ANTHROPIC_BASE_URL"])
	}
	if settings.Env["ANTHROPIC_AUTH_TOKEN"] != "test-key" {
		t.Errorf("ANTHROPIC_AUTH_TOKEN = %v, expected test-key", settings.Env["ANTHROPIC_AUTH_TOKEN"])
	}
	if settings.Env["ANTHROPIC_MODEL"] != "claude-3-opus" {
		t.Errorf("ANTHROPIC_MODEL = %v, expected claude-3-opus", settings.Env["ANTHROPIC_MODEL"])
	}

	// 验证额外环境变量
	if settings.Env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] != "gemini-2.5-flash-lite" {
		t.Errorf("ANTHROPIC_DEFAULT_HAIKU_MODEL = %v, expected gemini-2.5-flash-lite", settings.Env["ANTHROPIC_DEFAULT_HAIKU_MODEL"])
	}
	if settings.Env["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "gemini-claude-sonnet-4-5-thinking" {
		t.Errorf("ANTHROPIC_DEFAULT_SONNET_MODEL = %v, expected gemini-claude-sonnet-4-5-thinking", settings.Env["ANTHROPIC_DEFAULT_SONNET_MODEL"])
	}
	if settings.Env["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "gemini-claude-opus-4-5-thinking" {
		t.Errorf("ANTHROPIC_DEFAULT_OPUS_MODEL = %v, expected gemini-claude-opus-4-5-thinking", settings.Env["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
	if settings.Env["API_TIMEOUT_MS"] != "3000000" {
		t.Errorf("API_TIMEOUT_MS = %v, expected 3000000", settings.Env["API_TIMEOUT_MS"])
	}

	// 验证现有 env 变量被保留
	if settings.Env["EXISTING_VAR"] != "should-be-preserved" {
		t.Errorf("EXISTING_VAR = %v, expected should-be-preserved", settings.Env["EXISTING_VAR"])
	}

	// 验证其他设置被保留
	if settings.OtherSettings["model"] != "opus" {
		t.Errorf("model = %v, expected opus", settings.OtherSettings["model"])
	}
}
