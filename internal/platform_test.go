package internal

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestGetCurrentPlatform 测试获取当前平台函数.
func TestGetCurrentPlatform(t *testing.T) {
	platform := GetCurrentPlatform()

	// 根据实际运行环境检查平台
	expectedPlatform := PlatformLinux // 默认值
	switch runtime.GOOS {
	case "windows":
		expectedPlatform = PlatformWindows
	case "darwin":
		expectedPlatform = PlatformMac
	case "linux":
		expectedPlatform = PlatformLinux
	}

	if platform != expectedPlatform {
		t.Errorf("GetCurrentPlatform() = %v, expected %v for GOOS=%s", platform, expectedPlatform, runtime.GOOS)
	}

	// 确保返回的是有效的平台类型
	validPlatforms := []Platform{PlatformWindows, PlatformMac, PlatformLinux}
	isValid := false
	for _, validPlatform := range validPlatforms {
		if platform == validPlatform {
			isValid = true
			break
		}
	}
	if !isValid {
		t.Errorf("GetCurrentPlatform() returned invalid platform: %v", platform)
	}
}

// TestGetPathConfig 测试获取路径配置.
func TestGetPathConfig(t *testing.T) {
	// 设置临时home目录
	tempDir := t.TempDir()
	oldHome := setTempHome(t, tempDir)
	defer restoreHome(oldHome)

	config, err := GetPathConfig()
	if err != nil {
		t.Fatalf("GetPathConfig() error = %v", err)
	}

	if config == nil {
		t.Fatal("PathConfig is nil")
	}

	// 检查基本字段
	if config.HomeDir != tempDir {
		t.Errorf("HomeDir = %v, expected %v", config.HomeDir, tempDir)
	}

	// 检查Codex配置目录路径
	expectedCodexDir := filepath.Join(tempDir, ".codex")
	if config.CodexConfigDir != expectedCodexDir {
		t.Errorf("CodexConfigDir = %v, expected %v", config.CodexConfigDir, expectedCodexDir)
	}

	// VS Code配置目录路径会根据平台不同而不同
	platform := GetCurrentPlatform()
	var expectedVSCodeDir string
	switch platform {
	case PlatformWindows:
		expectedVSCodeDir = filepath.Join(tempDir, "AppData", "Roaming", "Code", "User")
	case PlatformMac:
		expectedVSCodeDir = filepath.Join(tempDir, "Library", "Application Support", "Code", "User")
	case PlatformLinux:
		expectedVSCodeDir = filepath.Join(tempDir, ".config", "Code", "User")
	}

	if config.VSCodeConfigDir != expectedVSCodeDir {
		t.Errorf("VSCodeConfigDir = %v, expected %v for platform %v", config.VSCodeConfigDir, expectedVSCodeDir, platform)
	}
}

// TestGetPathConfigPlatformSpecific 测试不同平台的路径配置.
func TestGetPathConfigPlatformSpecific(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := setTempHome(t, tempDir)
	defer restoreHome(oldHome)

	tests := []struct {
		name              string
		platform          Platform
		expectedCodexDir  string
		expectedVSCodeDir string
	}{
		{
			name:              "Windows路径",
			platform:          PlatformWindows,
			expectedCodexDir:  filepath.Join(tempDir, ".codex"),
			expectedVSCodeDir: filepath.Join(tempDir, "AppData", "Roaming", "Code", "User"),
		},
		{
			name:              "Mac路径",
			platform:          PlatformMac,
			expectedCodexDir:  filepath.Join(tempDir, ".codex"),
			expectedVSCodeDir: filepath.Join(tempDir, "Library", "Application Support", "Code", "User"),
		},
		{
			name:              "Linux路径",
			platform:          PlatformLinux,
			expectedCodexDir:  filepath.Join(tempDir, ".codex"),
			expectedVSCodeDir: filepath.Join(tempDir, ".config", "Code", "User"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 注意：这里我们无法真正模拟不同的平台，
			// 但我们可以验证当前平台的行为是否正确
			config, err := GetPathConfig()
			if err != nil {
				t.Fatalf("GetPathConfig() error = %v", err)
			}

			currentPlatform := GetCurrentPlatform()
			if currentPlatform == tt.platform {
				if config.CodexConfigDir != tt.expectedCodexDir {
					t.Errorf("CodexConfigDir = %v, expected %v", config.CodexConfigDir, tt.expectedCodexDir)
				}
				if config.VSCodeConfigDir != tt.expectedVSCodeDir {
					t.Errorf("VSCodeConfigDir = %v, expected %v", config.VSCodeConfigDir, tt.expectedVSCodeDir)
				}
			}
		})
	}
}

// TestEnsureDir 测试目录创建函数.
func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		dir       string
		shouldErr bool
	}{
		{
			name:      "创建新目录",
			dir:       filepath.Join(tempDir, "test-new-dir"),
			shouldErr: false,
		},
		{
			name:      "现有目录",
			dir:       tempDir, // 已存在的临时目录
			shouldErr: false,
		},
		{
			name:      "嵌套目录",
			dir:       filepath.Join(tempDir, "nested", "deep", "directory"),
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureDir(tt.dir)
			if (err != nil) != tt.shouldErr {
				t.Errorf("EnsureDir() error = %v, shouldErr %v", err, tt.shouldErr)
			}

			// 如果没有错误，检查目录是否真的存在
			if !tt.shouldErr {
				if _, err := os.Stat(tt.dir); os.IsNotExist(err) {
					t.Errorf("Directory %s was not created", tt.dir)
				}
			}
		})
	}
}

// TestEnsureDirPermissions 测试目录权限.
func TestEnsureDirPermissions(t *testing.T) {
	// 在Windows上跳过权限测试
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "permission-test")

	err := EnsureDir(testDir)
	if err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}

	info, err := os.Stat(testDir)
	if err != nil {
		t.Fatalf("stat error = %v", err)
	}

	expectedPerm := os.FileMode(0o755)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("Directory permission = %o, expected %o", info.Mode().Perm(), expectedPerm)
	}
}

// TestGetCodexConfigPath 测试获取Codex配置文件路径.
func TestGetCodexConfigPath(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := setTempHome(t, tempDir)
	defer restoreHome(oldHome)

	path, err := GetCodexConfigPath()
	if err != nil {
		t.Fatalf("GetCodexConfigPath() error = %v", err)
	}

	expectedPath := filepath.Join(tempDir, ".codex", "config.toml")
	if path != expectedPath {
		t.Errorf("GetCodexConfigPath() = %v, expected %v", path, expectedPath)
	}

	// 验证路径格式是否正确
	if !filepath.IsAbs(path) {
		t.Errorf("GetCodexConfigPath() returned relative path: %v", path)
	}

	if filepath.Ext(path) != ".toml" {
		t.Errorf("GetCodexConfigPath() should return .toml file: %v", path)
	}
}

// TestGetCodexAuthPath 测试获取Codex认证文件路径.
func TestGetCodexAuthPath(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := setTempHome(t, tempDir)
	defer restoreHome(oldHome)

	path, err := GetCodexAuthPath()
	if err != nil {
		t.Fatalf("GetCodexAuthPath() error = %v", err)
	}

	expectedPath := filepath.Join(tempDir, ".codex", "auth.json")
	if path != expectedPath {
		t.Errorf("GetCodexAuthPath() = %v, expected %v", path, expectedPath)
	}

	// 验证路径格式是否正确
	if !filepath.IsAbs(path) {
		t.Errorf("GetCodexAuthPath() returned relative path: %v", path)
	}

	if filepath.Ext(path) != ".json" {
		t.Errorf("GetCodexAuthPath() should return .json file: %v", path)
	}
}

// TestGetVSCodeSettingsPath 测试获取VS Code设置文件路径.
func TestGetVSCodeSettingsPath(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := setTempHome(t, tempDir)
	defer restoreHome(oldHome)

	path, err := GetVSCodeSettingsPath()
	if err != nil {
		t.Fatalf("GetVSCodeSettingsPath() error = %v", err)
	}

	// 根据当前平台验证路径
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

	if path != expectedPath {
		t.Errorf("GetVSCodeSettingsPath() = %v, expected %v for platform %v", path, expectedPath, platform)
	}

	// 验证路径格式是否正确
	if !filepath.IsAbs(path) {
		t.Errorf("GetVSCodeSettingsPath() returned relative path: %v", path)
	}

	if filepath.Ext(path) != ".json" {
		t.Errorf("GetVSCodeSettingsPath() should return .json file: %v", path)
	}
}

// TestPathConfigErrorHandling 测试路径配置错误处理.
func TestPathConfigErrorHandling(t *testing.T) {
	// 保存原始环境变量
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")

	// 清除所有可能的home目录环境变量
	os.Unsetenv("HOME")
	os.Unsetenv("USERPROFILE")

	defer func() {
		// 恢复环境变量
		if oldHome != "" {
			os.Setenv("HOME", oldHome)
		}
		if oldUserProfile != "" {
			os.Setenv("USERPROFILE", oldUserProfile)
		}
	}()

	_, err := GetPathConfig()
	if err == nil {
		// 某些系统可能仍然能够确定用户目录，这是可以的
		t.Log("GetPathConfig() succeeded even without HOME environment variable")
	} else {
		t.Logf("GetPathConfig() correctly returned error when HOME is not set: %v", err)
	}
}

// TestPlatformConstants 测试平台常量定义.
func TestPlatformConstants(t *testing.T) {
	tests := []struct {
		platform Platform
		expected string
	}{
		{PlatformWindows, "windows"},
		{PlatformMac, "mac"},
		{PlatformLinux, "linux"},
	}

	for _, tt := range tests {
		t.Run(string(tt.platform), func(t *testing.T) {
			if string(tt.platform) != tt.expected {
				t.Errorf("Platform constant %v = %v, expected %v", tt.platform, string(tt.platform), tt.expected)
			}
		})
	}
}

// TestEnvironmentConstants 测试环境变量常量.
func TestEnvironmentConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"CodexSwitchAPIKeyEnv", CodexSwitchAPIKeyEnv, "CODEX_SWITCH_OPENAI_API_KEY"},
		{"AnthropicBaseURLEnv", AnthropicBaseURLEnv, "ANTHROPIC_BASE_URL"},
		{"AnthropicAuthTokenEnv", AnthropicAuthTokenEnv, "ANTHROPIC_AUTH_TOKEN"},
		{"AnthropicModelEnv", AnthropicModelEnv, "ANTHROPIC_MODEL"},
		{"DefaultMirrorName", DefaultMirrorName, "official"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %v, expected %v", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// Helper functions for testing

// setTempHome 设置临时home目录并返回原始值.
func setTempHome(_ *testing.T, tempDir string) map[string]string {
	oldValues := make(map[string]string)

	// 保存原始值
	oldValues["HOME"] = os.Getenv("HOME")
	oldValues["USERPROFILE"] = os.Getenv("USERPROFILE")

	// 设置新值
	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)

	return oldValues
}

// restoreHome 恢复原始home目录环境变量.
func restoreHome(oldValues map[string]string) {
	for key, value := range oldValues {
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
}

// TestFilepathIntegration 测试文件路径集成功能.
func TestFilepathIntegration(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := setTempHome(t, tempDir)
	defer restoreHome(oldHome)

	// 测试所有路径函数返回的路径都在同一个基础目录下
	codexPath, err := GetCodexConfigPath()
	if err != nil {
		t.Fatalf("GetCodexConfigPath() error = %v", err)
	}

	authPath, err := GetCodexAuthPath()
	if err != nil {
		t.Fatalf("GetCodexAuthPath() error = %v", err)
	}

	vscodePath, err := GetVSCodeSettingsPath()
	if err != nil {
		t.Fatalf("GetVSCodeSettingsPath() error = %v", err)
	}

	// 验证所有路径都以tempDir开始
	paths := []struct {
		name string
		path string
	}{
		{"Codex config", codexPath},
		{"Codex auth", authPath},
		{"VSCode settings", vscodePath},
	}

	for _, p := range paths {
		if !filepath.IsAbs(p.path) {
			t.Errorf("%s path is not absolute: %v", p.name, p.path)
		}

		// 确保路径在tempDir之下
		relPath, err := filepath.Rel(tempDir, p.path)
		if err != nil || filepath.IsAbs(relPath) || relPath == ".." || len(relPath) >= 2 && relPath[:3] == ".."+string(filepath.Separator) {
			t.Errorf("%s path %v is not under temp directory %v", p.name, p.path, tempDir)
		}
	}

	// 验证Codex相关路径在同一个目录下
	codexDir := filepath.Dir(codexPath)
	authDir := filepath.Dir(authPath)
	if codexDir != authDir {
		t.Errorf("Codex config and auth files should be in the same directory: %v vs %v", codexDir, authDir)
	}
}
