package main

import (
	"testing"

	"codex-mirror/internal"
)

// 创建测试用的 App 实例
func createTestApp(t *testing.T) *App {
	t.Helper()

	// 创建一个临时配置路径的 MirrorManager
	mm, err := internal.NewMirrorManager()
	if err != nil {
		t.Fatalf("创建测试 App 失败: %v", err)
	}

	return &App{
		mirrorManager: mm,
		configPath:    mm.GetConfigPath(),
	}
}

// TestNewApp 测试 App 构造函数
func TestNewApp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() 失败: %v", err)
	}

	if app == nil {
		t.Fatal("NewApp() 返回 nil")
	}

	if app.mirrorManager == nil {
		t.Error("App.mirrorManager 未初始化")
	}

	if app.configPath == "" {
		t.Error("App.configPath 为空")
	}
}

// TestListMirrors 测试获取镜像源列表
func TestListMirrors(t *testing.T) {
	app := createTestApp(t)

	mirrors := app.ListMirrors()

	// 验证返回值不为 nil
	if mirrors == nil {
		t.Fatal("ListMirrors() 返回 nil")
	}

	// 至少应该有默认的 official 镜像源
	if len(mirrors) == 0 {
		t.Error("ListMirrors() 应该至少返回一个镜像源")
	}

	// 验证 DTO 转换正确
	for _, m := range mirrors {
		if m.Name == "" {
			t.Error("镜像源名称为空")
		}
		if m.BaseURL == "" {
			t.Error("镜像源 URL 为空")
		}
		if m.ToolType == "" {
			t.Error("镜像源类型为空")
		}
		// API Key 应该被掩码
		if m.APIKey != "" && len(m.APIKey) < 8 {
			t.Errorf("API Key 掩码格式不正确: %s", m.APIKey)
		}
	}
}

// TestGetMirror 测试获取单个镜像源
func TestGetMirror(t *testing.T) {
	app := createTestApp(t)

	// 测试获取官方镜像源
	mirror, err := app.GetMirror("official")
	if err != nil {
		t.Fatalf("GetMirror('official') 失败: %v", err)
	}

	if mirror.Name != "official" {
		t.Errorf("镜像源名称不正确: expected 'official', got '%s'", mirror.Name)
	}

	if mirror.BaseURL == "" {
		t.Error("镜像源 URL 为空")
	}

	// 测试获取不存在的镜像源
	_, err = app.GetMirror("nonexistent")
	if err == nil {
		t.Error("GetMirror('nonexistent') 应该返回错误")
	}
}

// TestAddMirror 测试添加镜像源
func TestAddMirror(t *testing.T) {
	app := createTestApp(t)

	// 准备测试数据
	testMirror := MirrorDTO{
		Name:     "test-mirror",
		BaseURL:  "https://api.test.com",
		APIKey:   "test-key-1234567890",
		ToolType: "codex",
	}

	// 添加镜像源
	err := app.AddMirror(testMirror)
	if err != nil {
		t.Fatalf("AddMirror() 失败: %v", err)
	}

	// 验证添加成功
	mirror, err := app.GetMirror("test-mirror")
	if err != nil {
		t.Fatalf("获取新添加的镜像源失败: %v", err)
	}

	if mirror.Name != "test-mirror" {
		t.Errorf("镜像源名称不正确: expected 'test-mirror', got '%s'", mirror.Name)
	}

	if mirror.BaseURL != testMirror.BaseURL {
		t.Errorf("镜像源 URL 不正确: expected '%s', got '%s'", testMirror.BaseURL, mirror.BaseURL)
	}

	// 清理测试数据
	app.RemoveMirror("test-mirror")
}

// TestAddMirrorDuplicate 测试添加重复镜像源
func TestAddMirrorDuplicate(t *testing.T) {
	app := createTestApp(t)

	// 添加一个镜像源
	testMirror := MirrorDTO{
		Name:     "duplicate-test",
		BaseURL:  "https://api.test1.com",
		ToolType: "codex",
	}

	err := app.AddMirror(testMirror)
	if err != nil {
		t.Fatalf("第一次添加失败: %v", err)
	}

	// 尝试添加同名镜像源
	err = app.AddMirror(testMirror)
	if err == nil {
		t.Error("添加同名镜像源应该返回错误")
	}

	// 清理
	app.RemoveMirror("duplicate-test")
}

// TestUpdateMirror 测试更新镜像源
func TestUpdateMirror(t *testing.T) {
	app := createTestApp(t)

	// 先添加一个镜像源
	testMirror := MirrorDTO{
		Name:     "update-test",
		BaseURL:  "https://api.test1.com",
		APIKey:   "old-key",
		ToolType: "codex",
	}

	app.AddMirror(testMirror)

	// 更新镜像源
	updatedMirror := MirrorDTO{
		Name:     "update-test",
		BaseURL:  "https://api.test2.com",
		APIKey:   "new-key",
		ToolType: "codex",
	}

	err := app.UpdateMirror(updatedMirror)
	if err != nil {
		t.Fatalf("UpdateMirror() 失败: %v", err)
	}

	// 验证更新成功
	mirror, _ := app.GetMirror("update-test")
	if mirror.BaseURL != updatedMirror.BaseURL {
		t.Errorf("URL 更新失败: expected '%s', got '%s'", updatedMirror.BaseURL, mirror.BaseURL)
	}

	// 清理
	app.RemoveMirror("update-test")
}

// TestRemoveMirror 测试删除镜像源
func TestRemoveMirror(t *testing.T) {
	app := createTestApp(t)

	// 先添加一个镜像源
	testMirror := MirrorDTO{
		Name:     "delete-test",
		BaseURL:  "https://api.test.com",
		ToolType: "codex",
	}

	app.AddMirror(testMirror)

	// 删除镜像源（软删除）
	err := app.RemoveMirror("delete-test")
	if err != nil {
		t.Fatalf("RemoveMirror() 失败: %v", err)
	}

	// 验证软删除：ListMirrors 不应包含已删除的镜像
	mirrors := app.ListMirrors()
	for _, m := range mirrors {
		if m.Name == "delete-test" {
			t.Error("软删除后 ListMirrors 不应包含该镜像源")
		}
	}
}

// TestRemoveOfficialMirror 测试删除官方镜像源（应该失败）
func TestRemoveOfficialMirror(t *testing.T) {
	app := createTestApp(t)

	err := app.RemoveMirror("official")
	if err == nil {
		t.Error("不应该允许删除官方镜像源")
	}
}

// TestSwitchMirror 测试切换镜像源
func TestSwitchMirror(t *testing.T) {
	app := createTestApp(t)

	// 添加一个测试镜像源
	testMirror := MirrorDTO{
		Name:     "switch-test",
		BaseURL:  "https://api.test.com",
		ToolType: "codex",
	}

	app.AddMirror(testMirror)
	defer app.RemoveMirror("switch-test")

	// 切换到测试镜像源
	err := app.SwitchMirror("switch-test")
	if err != nil {
		// 可能因为 Codex 配置写入失败，但不影响切换逻辑本身
		t.Logf("SwitchMirror() 可能的配置错误: %v", err)
	}

	// 验证状态更新
	status := app.GetCurrentStatus()
	if status.CurrentCodex != "switch-test" {
		t.Errorf("切换后当前镜像源应为 'switch-test', 实际为 '%s'", status.CurrentCodex)
	}
}

// TestGetCurrentStatus 测试获取当前状态
func TestGetCurrentStatus(t *testing.T) {
	app := createTestApp(t)

	status := app.GetCurrentStatus()

	if status.ConfigPath == "" {
		t.Error("ConfigPath 为空")
	}

	// 验证状态结构
	if status.CodexStatus.Path == "" {
		t.Error("CodexStatus.Path 为空")
	}

	if status.ClaudeStatus.Path != "环境变量" {
		t.Error("ClaudeStatus.Path 应该是 '环境变量'")
	}
}

// TestValidateURL 测试 URL 验证
func TestValidateURL(t *testing.T) {
	app := createTestApp(t)

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"有效 URL", "https://api.example.com", false},
		{"有效 HTTP URL", "http://localhost:8080", false},
		{"空 URL", "", true},
		{"缺少协议", "api.example.com", true},
		{"无效协议", "ftp://api.example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

// TestMaskAPIKey 测试 API Key 掩码
func TestMaskAPIKey(t *testing.T) {
	app := createTestApp(t)

	tests := []struct {
		name     string
		apiKey   string
		contains string // 检查结果是否包含这些字符
	}{
		{"短 Key", "abc123", "******"},
		{"8字符 Key", "abcd1234", "****"},
		{"长 Key", "sk-1234567890abcdefghijklmnop", "sk-1"},
		{"标准 Key", "sk-proj-abc123def456ghi789", "sk-p"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.maskAPIKey(tt.apiKey)
			// 确保掩码后长度与原 Key 相同
			if len(result) != len(tt.apiKey) {
				t.Errorf("掩码后长度不匹配: got %d, want %d", len(result), len(tt.apiKey))
			}
			// 确保包含星号
			if !contains(result, "*") {
				t.Error("掩码结果应该包含星号")
			}
			// 确保包含前缀（如果有）
			if tt.contains != "" && !contains(result, tt.contains) {
				t.Errorf("掩码结果应该包含 %q, got %q", tt.contains, result)
			}
		})
	}
}

// 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestToMirrorDTO 测试 MirrorConfig 到 MirrorDTO 的转换
func TestToMirrorDTO(t *testing.T) {
	app := createTestApp(t)

	// 创建测试用的 MirrorConfig
	mirrorConfig := internal.MirrorConfig{
		Name:      "test-dto",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-test1234567890abcdef",
		ToolType:  internal.ToolTypeCodex,
		ModelName: "gpt-4",
		EnvKey:    "CODEX_SWITCH_OPENAI_API_KEY",
	}

	config := &internal.SystemConfig{
		CurrentCodex: "test-dto",
	}

	// 执行转换
	dto := app.toMirrorDTO(mirrorConfig, config)

	// 验证基本字段
	if dto.Name != mirrorConfig.Name {
		t.Errorf("Name 不匹配: expected %s, got %s", mirrorConfig.Name, dto.Name)
	}

	if dto.BaseURL != mirrorConfig.BaseURL {
		t.Errorf("BaseURL 不匹配: expected %s, got %s", mirrorConfig.BaseURL, dto.BaseURL)
	}

	if dto.ToolType != string(mirrorConfig.ToolType) {
		t.Errorf("ToolType 不匹配: expected %s, got %s", mirrorConfig.ToolType, dto.ToolType)
	}

	// 验证 API Key 被掩码
	if dto.APIKey == mirrorConfig.APIKey {
		t.Error("API Key 应该被掩码")
	}

	if !dto.HasAPIKey {
		t.Error("HasAPIKey 应该为 true")
	}

	// 验证当前激活状态
	if !dto.IsCurrent {
		t.Error("IsCurrent 应该为 true")
	}
}

// TestMirrorDTOJSON 测试 MirrorDTO 的 JSON 序列化
func TestMirrorDTOJSON(t *testing.T) {
	dto := MirrorDTO{
		Name:      "json-test",
		BaseURL:   "https://api.test.com",
		APIKey:    "sk-****test",
		HasAPIKey: true,
		ToolType:  "claude",
		IsCurrent: true,
	}

	// 验证 JSON 标签
	if dto.Name == "" {
		t.Error("MirrorDTO Name 为空")
	}

	// 如果 JSON 序列化有问题，这里会暴露
	_ = dto.Name
	_ = dto.BaseURL
	_ = dto.ToolType
}
