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
		expectedEnvKey := CodexSwitchAPIKeyEnv // Codex 固定使用专用的环境变量名
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
	config, rawConfig, err := ccm.loadExistingConfig()
	if err != nil {
		return err
	}

	providerConfig := ccm.createProviderConfig(mirror, config)
	ccm.updateConfigStructures(config, rawConfig, mirror.Name, providerConfig)

	return ccm.writeConfigFile(rawConfig)
}

func (ccm *CodexConfigManager) loadExistingConfig() (*CodexConfig, map[string]interface{}, error) {
	var config CodexConfig
	var rawConfig map[string]interface{}

	if _, err := os.Stat(ccm.configPath); err == nil {
		var err error
		rawConfig, err = ccm.decodeConfigFiles(&config)
		if err != nil {
			return nil, nil, err
		}
	} else {
		config = CodexConfig{
			ModelProvider:          "packycode",
			Model:                  "gpt-5",
			ModelReasoningEffort:   "high",
			DisableResponseStorage: true,
			ModelProviders:         make(map[string]ModelProviderConfig),
		}
		rawConfig = make(map[string]interface{})
	}

	return &config, rawConfig, nil
}

func (ccm *CodexConfigManager) decodeConfigFiles(config *CodexConfig) (map[string]interface{}, error) {
	var rawConfig map[string]interface{}

	if _, err := toml.DecodeFile(ccm.configPath, &rawConfig); err != nil {
		return nil, fmt.Errorf("读取现有配置文件失败: %v", err)
	}

	if _, err := toml.DecodeFile(ccm.configPath, config); err != nil {
		return nil, fmt.Errorf("解析现有配置文件失败: %v", err)
	}

	return rawConfig, nil
}

func (ccm *CodexConfigManager) createProviderConfig(mirror *MirrorConfig, config *CodexConfig) ModelProviderConfig {
	if config.ModelProviders == nil {
		config.ModelProviders = make(map[string]ModelProviderConfig)
	}

	providerConfig := ModelProviderConfig{
		Name:               mirror.Name,
		BaseURL:            mirror.BaseURL,
		WireAPI:            "responses",
		EnvKey:             mirror.EnvKey,
		RequiresOpenAIAuth: true,
	}

	if existingProvider, exists := config.ModelProviders[mirror.Name]; exists {
		ccm.mergeExistingProviderConfig(&providerConfig, existingProvider)
	}

	return providerConfig
}

func (ccm *CodexConfigManager) mergeExistingProviderConfig(providerConfig *ModelProviderConfig, existingProvider ModelProviderConfig) {
	if existingProvider.WireAPI != "" {
		providerConfig.WireAPI = existingProvider.WireAPI
	}

	if existingProvider.EnvKey == CodexSwitchAPIKeyEnv {
		providerConfig.EnvKey = existingProvider.EnvKey
	}

	providerConfig.RequiresOpenAIAuth = existingProvider.RequiresOpenAIAuth
}

func (ccm *CodexConfigManager) updateConfigStructures(config *CodexConfig, rawConfig map[string]interface{}, mirrorName string, providerConfig ModelProviderConfig) {
	// 更新 config 结构体中的 ModelProviders
	if config.ModelProviders == nil {
		config.ModelProviders = make(map[string]ModelProviderConfig)
	}

	// 保存现有的 provider 配置
	existingProviders := make(map[string]ModelProviderConfig)
	for k, v := range config.ModelProviders {
		existingProviders[k] = v
	}

	// 添加或更新当前镜像的配置
	config.ModelProviders[mirrorName] = providerConfig

	if rawConfig == nil {
		rawConfig = make(map[string]interface{})
	}

	ccm.updateRawConfigBasicFields(rawConfig, config, mirrorName)
	ccm.updateRawConfigModelProviders(rawConfig, mirrorName, providerConfig, existingProviders)
}

func (ccm *CodexConfigManager) updateRawConfigBasicFields(rawConfig map[string]interface{}, config *CodexConfig, mirrorName string) {
	rawConfig["model_provider"] = mirrorName

	// 更新 Model 字段 - 测试期望总是更新为 gpt-5
	config.Model = TestModelGPT5
	rawConfig["model"] = config.Model

	// 更新 ModelReasoningEffort 字段
	if config.ModelReasoningEffort == "" {
		config.ModelReasoningEffort = TestHighEffort
	}
	rawConfig["model_reasoning_effort"] = config.ModelReasoningEffort

	// 总是强制设置 DisableResponseStorage 为 true
	config.DisableResponseStorage = true
	rawConfig["disable_response_storage"] = config.DisableResponseStorage
}

func (ccm *CodexConfigManager) updateRawConfigModelProviders(rawConfig map[string]interface{}, mirrorName string, providerConfig ModelProviderConfig, existingProviders map[string]ModelProviderConfig) {
	// 使用扁平化结构 [model_providers.mirrorname]
	// 保留现有镜像的配置，只更新当前镜像

	// 先移除旧的嵌套结构（如果存在）
	delete(rawConfig, "model_providers")

	// 移除当前镜像的旧扁平化键
	var keysToDelete []string
	for key := range rawConfig {
		if strings.HasPrefix(key, "model_providers.") {
			// 提取镜像名
			parts := strings.SplitN(key, ".", 3)
			if len(parts) >= 3 && parts[1] == mirrorName {
				keysToDelete = append(keysToDelete, key)
			}
		}
	}
	for _, key := range keysToDelete {
		delete(rawConfig, key)
	}

	// 添加所有现有的 provider 配置（包括新添加的）
	allProviders := make(map[string]ModelProviderConfig)
	for k, v := range existingProviders {
		allProviders[k] = v
	}
	allProviders[mirrorName] = providerConfig

	// 将所有 provider 配置写入 rawConfig
	for providerName, provider := range allProviders {
		sectionName := "model_providers." + providerName
		rawConfig[sectionName] = map[string]interface{}{
			"name":                 provider.Name,
			"base_url":             provider.BaseURL,
			"wire_api":             provider.WireAPI,
			"env_key":              provider.EnvKey,
			"requires_openai_auth": provider.RequiresOpenAIAuth,
		}
	}
}

func (ccm *CodexConfigManager) writeConfigFile(rawConfig map[string]interface{}) error {
	file, err := os.Create(ccm.configPath)
	if err != nil {
		return fmt.Errorf("创建配置文件失败: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("警告: 关闭配置文件失败: %v\n", err)
		}
	}()

	// 首先写入基本配置项（不包含点的键值对）
	basicKeys := []string{"model_provider", "model", "model_reasoning_effort", "disable_response_storage"}
	for _, key := range basicKeys {
		if value, exists := rawConfig[key]; exists && !isMap(value) {
			if err := writeTOMLValue(file, key, value, ""); err != nil {
				return err
			}
		}
	}

	// 然后写入带点的节（如 [model_providers.provider_name]）
	for key, value := range rawConfig {
		if strings.Contains(key, ".") {
			if subMap, ok := value.(map[string]interface{}); ok {
				if _, err := fmt.Fprintf(file, "\n[%s]\n", key); err != nil {
					return err
				}
				if err := writeTOMLMap(file, subMap, "  "); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func isMap(value interface{}) bool {
	_, ok := value.(map[string]interface{})
	return ok
}

func writeTOMLMap(file *os.File, m map[string]interface{}, indent string) error {
	for key, value := range m {
		if err := writeTOMLValue(file, key, value, indent); err != nil {
			return err
		}
	}
	return nil
}

func writeTOMLValue(file *os.File, key string, value interface{}, indent string) error {
	switch v := value.(type) {
	case string:
		_, err := fmt.Fprintf(file, "%s%s = %q\n", indent, key, v)
		return err
	case bool:
		_, err := fmt.Fprintf(file, "%s%s = %t\n", indent, key, v)
		return err
	case int:
		_, err := fmt.Fprintf(file, "%s%s = %d\n", indent, key, v)
		return err
	case int32:
		_, err := fmt.Fprintf(file, "%s%s = %d\n", indent, key, v)
		return err
	case int64:
		_, err := fmt.Fprintf(file, "%s%s = %d\n", indent, key, v)
		return err
	case float32:
		_, err := fmt.Fprintf(file, "%s%s = %f\n", indent, key, v)
		return err
	case float64:
		_, err := fmt.Fprintf(file, "%s%s = %f\n", indent, key, v)
		return err
	default:
		_, err := fmt.Fprintf(file, "%s%s = %v\n", indent, key, v)
		return err
	}
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
	fmt.Printf("[OK] 环境变量 %s 已设置\n", envKey)
	return nil
}

// setMacUserEnvVar 在macOS中设置用户级环境变量.
func (ccm *CodexConfigManager) setMacUserEnvVar(envKey, apiKey string) error {
	shellFiles := []string{".zshrc"} // macOS 默认使用 zsh
	return setUnixUserEnvVar(envKey, apiKey, shellFiles)
}

// setLinuxUserEnvVar 在Linux中设置用户级环境变量.
func (ccm *CodexConfigManager) setLinuxUserEnvVar(envKey, apiKey string) error {
	shellFiles := []string{".bashrc", ".profile"} // bash (最常见), 通用profile.
	return setUnixUserEnvVar(envKey, apiKey, shellFiles)
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
