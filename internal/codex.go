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

// CodexConfigManager Codex配置管理器
type CodexConfigManager struct {
	configPath string
	authPath   string
}

// NewCodexConfigManager 创建新的Codex配置管理器
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

// UpdateConfig 更新Codex配置文件
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
			ModelProvider:       "packycode",
			Model:              "gpt-5",
			ModelReasoningEffort: "high",
			ModelProviders:     make(map[string]ModelProviderConfig),
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
		EnvKey:  mirror.Name,
	}
	
	// 如果是packycode提供商，保留现有的wire_api和env_key
	if existingProvider, exists := config.ModelProviders[mirror.Name]; exists {
		if existingProvider.WireAPI != "" {
			providerConfig.WireAPI = existingProvider.WireAPI
		}
		if existingProvider.EnvKey != "" {
			providerConfig.EnvKey = existingProvider.EnvKey
		}
	}
	
	config.ModelProviders[mirror.Name] = providerConfig
	
	// 如果没有设置默认值，设置默认值
	if config.ModelProvider == "" {
		config.ModelProvider = "packycode"
	}
	if config.Model == "" {
		config.Model = "gpt-5"
	}
	if config.ModelReasoningEffort == "" {
		config.ModelReasoningEffort = "high"
	}
	
	// 创建或更新config.toml文件
	file, err := os.Create(ccm.configPath)
	if err != nil {
		return fmt.Errorf("创建配置文件失败: %v", err)
	}
	defer file.Close()

	// 手动构建TOML内容以避免嵌套的[model_providers]结构
	content := fmt.Sprintf(`model_provider = "%s"
model = "%s"
model_reasoning_effort = "%s"

`, 
		config.ModelProvider, config.Model, config.ModelReasoningEffort)
	
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

// UpdateAuth 更新Codex认证文件
func (ccm *CodexConfigManager) UpdateAuth(mirror *MirrorConfig) error {
	auth := CodexAuth{
		APIKey: mirror.APIKey,
	}

	// 创建或更新auth.json文件
	file, err := os.Create(ccm.authPath)
	if err != nil {
		return fmt.Errorf("创建认证文件失败: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(auth); err != nil {
		return fmt.Errorf("写入认证文件失败: %v", err)
	}

	return nil
}

// SetEnvironmentVariable 设置环境变量
func (ccm *CodexConfigManager) SetEnvironmentVariable(envKey, apiKey string) error {
	if envKey == "" {
		return fmt.Errorf("环境变量key不能为空")
	}
	
	// 构造带前缀的环境变量名，避免冲突
	fullEnvKey := fmt.Sprintf("CODEX_%s_API_KEY", strings.ToUpper(envKey))
	
	// 在当前进程中设置环境变量
	if err := os.Setenv(fullEnvKey, apiKey); err != nil {
		return fmt.Errorf("设置环境变量 %s 失败: %v", fullEnvKey, err)
	}
	
	// 在Windows中设置用户级环境变量（持久化）
	if err := ccm.setWindowsUserEnvVar(fullEnvKey, apiKey); err != nil {
		return fmt.Errorf("设置Windows用户环境变量 %s 失败: %v", fullEnvKey, err)
	}
	
	return nil
}

// setWindowsUserEnvVar 在Windows中设置用户级环境变量
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

// ApplyMirror 应用镜像源配置到Codex CLI
func (ccm *CodexConfigManager) ApplyMirror(mirror *MirrorConfig) error {
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

// GetCurrentConfig 获取当前Codex配置
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

// GetCurrentBaseURL 获取当前使用的base_url
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

// GetCurrentAuth 获取当前Codex认证
func (ccm *CodexConfigManager) GetCurrentAuth() (*CodexAuth, error) {
	if _, err := os.Stat(ccm.authPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Codex认证文件不存在")
	}

	file, err := os.Open(ccm.authPath)
	if err != nil {
		return nil, fmt.Errorf("打开Codex认证文件失败: %v", err)
	}
	defer file.Close()

	var auth CodexAuth
	if err := json.NewDecoder(file).Decode(&auth); err != nil {
		return nil, fmt.Errorf("读取Codex认证文件失败: %v", err)
	}

	return &auth, nil
}

// BackupConfig 备份当前配置
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

// copyFile 复制文件
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = dstFile.ReadFrom(srcFile)
	return err
}