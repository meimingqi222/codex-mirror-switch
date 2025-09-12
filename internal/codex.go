package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// CodexConfigManager Codex配置管理器.
type CodexConfigManager struct {
	configPath string
	authPath   string
}

// NewCodexConfigManager 创建新的Codex配置管理器.
func NewCodexConfigManager() (*CodexConfigManager, error) {
	configPath, err := GetCodexConfigPath()
	if err != nil {
		return nil, fmt.Errorf("获取Codex配置路径失败: %v", err)
	}

	authPath, err := GetCodexAuthPath()
	if err != nil {
		return nil, fmt.Errorf("获取Codex认证路径失败: %v", err)
	}

	// 确保配置目录存在
	configDir := filepath.Dir(configPath)
	if err := EnsureDir(configDir); err != nil {
		return nil, fmt.Errorf("创建Codex配置目录失败: %v", err)
	}

	return &CodexConfigManager{
		configPath: configPath,
		authPath:   authPath,
	}, nil
}

// UpdateConfig 更新Codex配置文件.
// FixEnvKeyFormat 修复所有镜像源的env_key格式为CODEX_XXX_API_KEY.
func (ccm *CodexConfigManager) FixEnvKeyFormat() error {
	// 读取现有配置
	var config CodexConfig
	if _, err := os.Stat(ccm.configPath); err != nil {
		return nil // 配置文件不存在，无需修复
	}

	if _, err := toml.DecodeFile(ccm.configPath, &config); err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	if config.ModelProviders == nil {
		return nil // 没有镜像源配置
	}

	// 检查并修复每个镜像源的env_key格式
	updated := false
	for name, provider := range config.ModelProviders {
		expectedEnvKey := "CODEX_SWITCH_OPENAI_API_KEY" // Codex 固定使用专用的环境变量名
		if provider.EnvKey != expectedEnvKey {
			provider.EnvKey = expectedEnvKey
			config.ModelProviders[name] = provider
			updated = true
		}
	}

	// 如果有更新，保存配置文件
	if updated {
		if err := ccm.saveConfig(&config); err != nil {
			return fmt.Errorf("保存配置文件失败: %v", err)
		}
	}

	return nil
}

func (ccm *CodexConfigManager) UpdateConfig(mirror *MirrorConfig) error {
	// 尝试读取现有配置
	var config CodexConfig
	var rawConfig map[string]interface{}

	// 如果配置文件存在，先读取现有配置
	if _, err := os.Stat(ccm.configPath); err == nil {
		// 读取原始配置到map中以保留未知字段
		if _, err := toml.DecodeFile(ccm.configPath, &rawConfig); err != nil {
			return fmt.Errorf("读取现有配置文件失败: %v", err)
		}

		// 读取到结构体中
		if _, err := toml.DecodeFile(ccm.configPath, &config); err != nil {
			return fmt.Errorf("解析现有配置文件失败: %v", err)
		}
	} else {
		// 如果配置文件不存在，创建默认配置
		config = CodexConfig{
			ModelProvider:          "packycode",
			Model:                  "gpt-5",
			ModelReasoningEffort:   "high",
			DisableResponseStorage: true,
			ModelProviders:         make(map[string]ModelProviderConfig),
		}
		rawConfig = make(map[string]interface{})
	}

	// 确保ModelProviders存在
	if config.ModelProviders == nil {
		config.ModelProviders = make(map[string]ModelProviderConfig)
	}

	// 更新或添加镜像源配置
	providerConfig := ModelProviderConfig{
		Name:    mirror.Name,
		BaseURL: mirror.BaseURL,
		WireAPI: "responses",
		EnvKey:  mirror.EnvKey, // 直接使用镜像源中已设置的env_key
	}

	// 如果已存在配置，保留现有的wire_api和正确格式的env_key
	if existingProvider, exists := config.ModelProviders[mirror.Name]; exists {
		if existingProvider.WireAPI != "" {
			providerConfig.WireAPI = existingProvider.WireAPI
		}
		// 如果现有的env_key已经是正确的CODEX_前缀格式，保留它
		expectedEnvKey := "CODEX_SWITCH_OPENAI_API_KEY" // Codex 固定使用专用的环境变量名
		if existingProvider.EnvKey == expectedEnvKey {
			providerConfig.EnvKey = existingProvider.EnvKey
		}
	}

	config.ModelProviders[mirror.Name] = providerConfig

	// 设置model_provider为当前切换的镜像源名称
	config.ModelProvider = mirror.Name

	// 如果没有设置默认值，设置默认值
	if config.Model == "" {
		config.Model = "gpt-5"
	}
	if config.ModelReasoningEffort == "" {
		config.ModelReasoningEffort = "high"
	}

	// 强制写入禁用响应存储的配置
	config.DisableResponseStorage = true

	// 创建或更新config.toml文件
	file, err := os.Create(ccm.configPath)
	if err != nil {
		return fmt.Errorf("创建配置文件失败: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("警告: 关闭配置文件失败: %v\n", err)
		}
	}()

	// 手动构建TOML内容以避免嵌套的[model_providers]结构
	content := fmt.Sprintf(`model_provider = "%s"
model = "%s"
model_reasoning_effort = "%s"
disable_response_storage = %t

`,
		config.ModelProvider, config.Model, config.ModelReasoningEffort, config.DisableResponseStorage)

	// 添加每个模型提供商的配置
	for name, provider := range config.ModelProviders {
		content += fmt.Sprintf(`[model_providers.%s]
name = "%s"
base_url = "%s"
wire_api = "%s"
env_key = "%s"

`,
			name, provider.Name, provider.BaseURL, provider.WireAPI, provider.EnvKey)
	}

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// UpdateAuth 更新Codex认证文件.
func (ccm *CodexConfigManager) UpdateAuth(mirror *MirrorConfig) error {
	auth := CodexAuth{
		APIKey: mirror.APIKey,
	}

	// 创建或更新auth.json文件
	file, err := os.Create(ccm.authPath)
	if err != nil {
		return fmt.Errorf("创建认证文件失败: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("警告: 关闭认证文件失败: %v\n", err)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(auth); err != nil {
		return fmt.Errorf("写入认证文件失败: %v", err)
	}

	return nil
}

// SetEnvironmentVariable 设置环境变量.
func (ccm *CodexConfigManager) SetEnvironmentVariable(envKey, apiKey string) error {
	if envKey == "" {
		return fmt.Errorf("环境变量key不能为空")
	}

	// 直接使用传入的envKey，不再添加前缀（因为已经包含CODEX_前缀）
	// 在当前进程中设置环境变量
	if err := os.Setenv(envKey, apiKey); err != nil {
		return fmt.Errorf("设置环境变量 %s 失败: %v", envKey, err)
	}

	// 根据平台设置持久化环境变量
	platform := GetCurrentPlatform()
	switch platform {
	case PlatformWindows:
		if err := ccm.setWindowsUserEnvVar(envKey, apiKey); err != nil {
			return fmt.Errorf("设置Windows用户环境变量 %s 失败: %v", envKey, err)
		}
	case PlatformMac:
		if err := ccm.setMacUserEnvVar(envKey, apiKey); err != nil {
			return fmt.Errorf("设置macOS用户环境变量 %s 失败: %v", envKey, err)
		}
	case PlatformLinux:
		if err := ccm.setLinuxUserEnvVar(envKey, apiKey); err != nil {
			return fmt.Errorf("设置Linux用户环境变量 %s 失败: %v", envKey, err)
		}
	}

	return nil
}

// setWindowsUserEnvVar 在Windows中设置用户级环境变量.
func (ccm *CodexConfigManager) setWindowsUserEnvVar(envKey, apiKey string) error {
	// 使用setx命令设置用户级环境变量
	cmd := exec.Command("setx", envKey, apiKey)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("执行setx命令失败: %v, 输出: %s", err, string(output))
	}
	fmt.Printf("✓ 环境变量 %s 已设置\n", envKey)
	return nil
}

// setMacUserEnvVar 在macOS中设置用户级环境变量.
func (ccm *CodexConfigManager) setMacUserEnvVar(envKey, apiKey string) error {
	shellFiles := []string{".zshrc"} // macOS 默认使用 zsh
	return ccm.setUnixUserEnvVar(envKey, apiKey, shellFiles)
}

// setLinuxUserEnvVar 在Linux中设置用户级环境变量.
func (ccm *CodexConfigManager) setLinuxUserEnvVar(envKey, apiKey string) error {
	shellFiles := []string{".bashrc", ".profile"} // bash (最常见), 通用profile.
	return ccm.setUnixUserEnvVar(envKey, apiKey, shellFiles)
}

// setUnixUserEnvVar 在Unix系统（macOS和Linux）中设置用户级环境变量.
func (ccm *CodexConfigManager) setUnixUserEnvVar(envKey, apiKey string, shellFileNames []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %v", err)
	}

	// 构建完整的文件路径
	shellFiles := make([]string, len(shellFileNames))
	for i, name := range shellFileNames {
		shellFiles[i] = filepath.Join(homeDir, name)
	}

	envLine := fmt.Sprintf("export %s=%s", envKey, apiKey)
	updated := false

	for _, shellFile := range shellFiles {
		if err := ccm.updateShellProfile(shellFile, envKey, envLine); err != nil {
			fmt.Printf("警告: 更新 %s 失败: %v\n", shellFile, err)
			continue
		}
		updated = true
	}

	if !updated {
		return fmt.Errorf("无法更新任何shell配置文件")
	}

	fmt.Printf("✓ 环境变量 %s 已添加到shell配置文件\n", envKey)
	return nil
}

// updateShellProfile 更新shell配置文件，添加或更新环境变量.
func (ccm *CodexConfigManager) updateShellProfile(shellFile, envKey, envLine string) error {
	// 读取现有内容
	var existingContent []byte
	var err error
	if _, err = os.Stat(shellFile); err == nil {
		existingContent, err = os.ReadFile(shellFile)
		if err != nil {
			return fmt.Errorf("读取文件失败: %v", err)
		}
	}

	content := string(existingContent)
	lines := strings.Split(content, "\n")

	// 检查是否已存在该环境变量的设置
	envPattern := fmt.Sprintf("export %s=", envKey)
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), envPattern) {
			// 更新现有行
			lines[i] = envLine
			found = true
			break
		}
	}

	// 如果没找到，添加新行
	if !found {
		// 合并多个append操作
		lines = append(lines, "", "# Codex Mirror Switch - API Key.", envLine)
	}

	// 写回文件
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(shellFile, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}

// ApplyMirror 应用镜像源配置到Codex CLI.
func (ccm *CodexConfigManager) ApplyMirror(mirror *MirrorConfig) error {
	// 首先修复所有镜像源的env_key格式
	if err := ccm.FixEnvKeyFormat(); err != nil {
		return fmt.Errorf("修复env_key格式失败: %v", err)
	}

	// 更新配置文件
	if err := ccm.UpdateConfig(mirror); err != nil {
		return fmt.Errorf("更新Codex配置失败: %v", err)
	}

	// 更新认证文件
	if err := ccm.UpdateAuth(mirror); err != nil {
		return fmt.Errorf("更新Codex认证失败: %v", err)
	}

	// 设置环境变量（从配置中获取env_key）
	config, err := ccm.GetCurrentConfig()
	if err == nil && config.ModelProviders != nil {
		if provider, exists := config.ModelProviders[mirror.Name]; exists && provider.EnvKey != "" {
			if err := ccm.SetEnvironmentVariable(provider.EnvKey, mirror.APIKey); err != nil {
				return fmt.Errorf("设置环境变量失败: %v", err)
			}
		}
	}

	return nil
}

// GetCurrentConfig 获取当前Codex配置.
func (ccm *CodexConfigManager) GetCurrentConfig() (*CodexConfig, error) {
	if _, err := os.Stat(ccm.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Codex配置文件不存在")
	}

	var config CodexConfig
	if _, err := toml.DecodeFile(ccm.configPath, &config); err != nil {
		return nil, fmt.Errorf("读取Codex配置文件失败: %v", err)
	}

	return &config, nil
}

// GetCurrentBaseURL 获取当前使用的base_url.
func (ccm *CodexConfigManager) GetCurrentBaseURL(providerName string) (string, error) {
	config, err := ccm.GetCurrentConfig()
	if err != nil {
		return "", err
	}

	if config.ModelProviders == nil {
		return "", fmt.Errorf("未找到模型提供商配置")
	}

	provider, exists := config.ModelProviders[providerName]
	if !exists {
		return "", fmt.Errorf("未找到提供商 %s 的配置", providerName)
	}

	return provider.BaseURL, nil
}

// GetCurrentAuth 获取当前Codex认证.
func (ccm *CodexConfigManager) GetCurrentAuth() (*CodexAuth, error) {
	if _, err := os.Stat(ccm.authPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Codex认证文件不存在")
	}

	file, err := os.Open(ccm.authPath)
	if err != nil {
		return nil, fmt.Errorf("打开Codex认证文件失败: %v", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("警告: 关闭认证文件失败: %v\n", closeErr)
		}
	}()

	var auth CodexAuth
	if err := json.NewDecoder(file).Decode(&auth); err != nil {
		return nil, fmt.Errorf("读取Codex认证文件失败: %v", err)
	}

	return &auth, nil
}

// BackupConfig 备份当前配置.
func (ccm *CodexConfigManager) BackupConfig() error {
	configDir := filepath.Dir(ccm.configPath)
	backupDir := filepath.Join(configDir, "backup")
	if err := EnsureDir(backupDir); err != nil {
		return fmt.Errorf("创建备份目录失败: %v", err)
	}

	// 备份config.toml
	if _, err := os.Stat(ccm.configPath); err == nil {
		backupConfigPath := filepath.Join(backupDir, "config.toml.bak")
		if err := copyFile(ccm.configPath, backupConfigPath); err != nil {
			return fmt.Errorf("备份配置文件失败: %v", err)
		}
	}

	// 备份auth.json
	if _, err := os.Stat(ccm.authPath); err == nil {
		backupAuthPath := filepath.Join(backupDir, "auth.json.bak")
		if err := copyFile(ccm.authPath, backupAuthPath); err != nil {
			return fmt.Errorf("备份认证文件失败: %v", err)
		}
	}

	return nil
}

// copyFile 复制文件.
// saveConfig 保存配置到文件.
func (ccm *CodexConfigManager) saveConfig(config *CodexConfig) error {
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(ccm.configPath), 0o755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 直接写入配置文件
	file, err := os.Create(ccm.configPath)
	if err != nil {
		return fmt.Errorf("创建配置文件失败: %v", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("警告: 关闭配置文件失败: %v\n", closeErr)
		}
	}()

	// 编码并写入配置
	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("编码配置失败: %v", err)
	}

	return nil
}

// copyFile 复制文件.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := srcFile.Close(); closeErr != nil {
			fmt.Printf("警告: 关闭源文件失败: %v\n", closeErr)
		}
	}()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := dstFile.Close(); closeErr != nil {
			fmt.Printf("警告: 关闭目标文件失败: %v\n", closeErr)
		}
	}()

	_, err = dstFile.ReadFrom(srcFile)
	return err
}
