package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ClaudeSettings Claude Code settings.json 结构.
type ClaudeSettings struct {
	Env           map[string]string      `json:"env,omitempty"`
	Permissions   map[string]interface{} `json:"permissions,omitempty"`
	OtherSettings map[string]interface{} `json:"-"`
}

// ClaudeConfigManager Claude Code 配置管理器.
type ClaudeConfigManager struct {
	settingsPath string
}

// NewClaudeConfigManager 创建新的 Claude Code 配置管理器.
func NewClaudeConfigManager() (*ClaudeConfigManager, error) {
	settingsPath, err := GetClaudeSettingsPath()
	if err != nil {
		return nil, fmt.Errorf("获取Claude配置路径失败: %v", err)
	}

	// 确保配置目录存在
	configDir := filepath.Dir(settingsPath)
	if err := EnsureDir(configDir); err != nil {
		return nil, fmt.Errorf("创建Claude配置目录失败: %v", err)
	}

	return &ClaudeConfigManager{
		settingsPath: settingsPath,
	}, nil
}

// GetClaudeSettingsPath 获取 Claude Code 设置文件路径.
func GetClaudeSettingsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".claude", "settings.json"), nil
}

// LoadSettings 加载 Claude Code 设置.
func (ccm *ClaudeConfigManager) LoadSettings() (*ClaudeSettings, error) {
	settings := &ClaudeSettings{
		Env:           make(map[string]string),
		OtherSettings: make(map[string]interface{}),
	}

	// 如果文件不存在，返回空的设置
	if _, err := os.Stat(ccm.settingsPath); os.IsNotExist(err) {
		return settings, nil
	}

	data, err := os.ReadFile(ccm.settingsPath)
	if err != nil {
		return nil, fmt.Errorf("读取Claude配置文件失败: %v", err)
	}

	// 先解析到通用 map 以保留所有字段
	var rawSettings map[string]interface{}
	if err := json.Unmarshal(data, &rawSettings); err != nil {
		return nil, fmt.Errorf("解析Claude配置文件失败: %v", err)
	}

	// 提取 env 字段
	if envRaw, ok := rawSettings["env"]; ok {
		if envMap, ok := envRaw.(map[string]interface{}); ok {
			for k, v := range envMap {
				if strVal, ok := v.(string); ok {
					settings.Env[k] = strVal
				}
			}
		}
		delete(rawSettings, "env")
	}

	// 提取 permissions 字段
	if permRaw, ok := rawSettings["permissions"]; ok {
		if permMap, ok := permRaw.(map[string]interface{}); ok {
			settings.Permissions = permMap
		}
		delete(rawSettings, "permissions")
	}

	// 保留其他所有字段
	settings.OtherSettings = rawSettings

	return settings, nil
}

// SaveSettings 保存 Claude Code 设置.
func (ccm *ClaudeConfigManager) SaveSettings(settings *ClaudeSettings) error {
	// 构建要写入的 map
	outputMap := make(map[string]interface{})

	// 复制其他设置
	for k, v := range settings.OtherSettings {
		outputMap[k] = v
	}

	// 添加 env 字段（如果有内容）
	if len(settings.Env) > 0 {
		outputMap["env"] = settings.Env
	}

	// 添加 permissions 字段（如果有内容）
	if len(settings.Permissions) > 0 {
		outputMap["permissions"] = settings.Permissions
	}

	// 使用原子写入
	configDir := filepath.Dir(ccm.settingsPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	tmpFile, err := os.CreateTemp(configDir, "settings-*.json")
	if err != nil {
		return fmt.Errorf("创建临时配置文件失败: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(outputMap); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("写入临时配置文件失败: %v", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("同步临时配置文件失败: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("关闭临时配置文件失败: %v", err)
	}

	if err := os.Rename(tmpPath, ccm.settingsPath); err != nil {
		return fmt.Errorf("替换配置文件失败: %v", err)
	}

	return nil
}

// ApplyMirror 应用镜像源配置到 Claude Code settings.json.
func (ccm *ClaudeConfigManager) ApplyMirror(mirror *MirrorConfig) error {
	return ccm.ApplyMirrorWithCleanup(mirror, nil)
}

// ApplyMirrorWithCleanup 应用镜像源配置，并清理旧镜像的额外环境变量.
func (ccm *ClaudeConfigManager) ApplyMirrorWithCleanup(mirror *MirrorConfig, oldExtraEnv map[string]string) error {
	settings, err := ccm.LoadSettings()
	if err != nil {
		return err
	}

	// 确保 env map 存在
	if settings.Env == nil {
		settings.Env = make(map[string]string)
	}

	// 清理旧镜像的额外环境变量（只清理不在新配置中的）
	for key := range oldExtraEnv {
		if _, existsInNew := mirror.ExtraEnv[key]; !existsInNew {
			delete(settings.Env, key)
		}
	}

	// 设置 Claude 相关环境变量
	settings.Env[AnthropicBaseURLEnv] = mirror.BaseURL
	settings.Env[AnthropicAuthTokenEnv] = mirror.APIKey

	// 设置或清除模型名称
	if mirror.ModelName != "" {
		settings.Env[AnthropicModelEnv] = mirror.ModelName
	} else {
		delete(settings.Env, AnthropicModelEnv)
	}

	// 应用额外的环境变量配置 (如 ANTHROPIC_DEFAULT_HAIKU_MODEL 等)
	for key, value := range mirror.ExtraEnv {
		if value != "" {
			settings.Env[key] = value
		} else {
			delete(settings.Env, key)
		}
	}

	return ccm.SaveSettings(settings)
}

// GetCurrentEnv 获取当前配置的环境变量.
func (ccm *ClaudeConfigManager) GetCurrentEnv() (map[string]string, error) {
	settings, err := ccm.LoadSettings()
	if err != nil {
		return nil, err
	}
	return settings.Env, nil
}

// BackupSettings 备份当前设置.
func (ccm *ClaudeConfigManager) BackupSettings() error {
	if _, err := os.Stat(ccm.settingsPath); os.IsNotExist(err) {
		return nil // 文件不存在，无需备份
	}

	configDir := filepath.Dir(ccm.settingsPath)
	backupDir := filepath.Join(configDir, "backup")
	if err := EnsureDir(backupDir); err != nil {
		return fmt.Errorf("创建备份目录失败: %v", err)
	}

	backupPath := filepath.Join(backupDir, "settings.json.bak")
	if err := copyFile(ccm.settingsPath, backupPath); err != nil {
		return fmt.Errorf("备份Claude设置文件失败: %v", err)
	}

	return nil
}

// GetSettingsPath 获取设置文件路径.
func (ccm *ClaudeConfigManager) GetSettingsPath() string {
	return ccm.settingsPath
}
