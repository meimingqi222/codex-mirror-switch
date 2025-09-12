package internal

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// MirrorManager 镜像源管理器.
type MirrorManager struct {
	configPath string
	config     *SystemConfig
}

// NewMirrorManager 创建新的镜像源管理器.
func NewMirrorManager() (*MirrorManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户主目录失败: %v", err)
	}

	configDir := filepath.Join(homeDir, ".codex-mirror")
	if err := EnsureDir(configDir); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %v", err)
	}

	configPath := filepath.Join(configDir, "mirrors.toml")
	mm := &MirrorManager{
		configPath: configPath,
		config:     &SystemConfig{},
	}

	// 尝试加载现有配置
	if err := mm.loadConfig(); err != nil {
		// 如果配置文件不存在，检查是否有已存在的环境变量
		mm.discoverFromEnvironment()
	}

	return mm, nil
}

// loadConfig 加载配置文件.
func (mm *MirrorManager) loadConfig() error {
	if _, err := os.Stat(mm.configPath); os.IsNotExist(err) {
		return err
	}

	_, err := toml.DecodeFile(mm.configPath, mm.config)
	return err
}

// saveConfig 保存配置文件.
func (mm *MirrorManager) saveConfig() error {
	file, err := os.Create(mm.configPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("警告: 关闭配置文件失败: %v\n", err)
		}
	}()

	return toml.NewEncoder(file).Encode(mm.config)
}

// initDefaultConfig 初始化默认配置.
func (mm *MirrorManager) initDefaultConfig() {
	mm.config = &SystemConfig{
		CurrentMirror: DefaultMirrorName,
		CurrentCodex:  DefaultMirrorName,
		Mirrors: []MirrorConfig{
			{
				Name:     DefaultMirrorName,
				BaseURL:  "https://api.openai.com",
				APIKey:   "",
				ToolType: ToolTypeCodex,
			},
		},
	}
}

// AddMirror 添加镜像源.
func (mm *MirrorManager) AddMirror(name, baseURL, apiKey string) error {
	return mm.AddMirrorWithType(name, baseURL, apiKey, ToolTypeCodex) // 默认为 codex 类型
}

// AddMirrorWithType 添加指定类型的镜像源.
func (mm *MirrorManager) AddMirrorWithType(name, baseURL, apiKey string, toolType ToolType) error {
	return mm.AddMirrorWithModel(name, baseURL, apiKey, toolType, "")
}

// AddMirrorWithModel 添加指定类型和模型名称的镜像源.
func (mm *MirrorManager) AddMirrorWithModel(name, baseURL, apiKey string, toolType ToolType, modelName string) error {
	// 检查镜像源是否已存在
	for _, mirror := range mm.config.Mirrors {
		if mirror.Name == name {
			return fmt.Errorf("镜像源 '%s' 已存在", name)
		}
	}

	// 添加新镜像源
	newMirror := MirrorConfig{
		Name:      name,
		BaseURL:   baseURL,
		APIKey:    apiKey,
		ToolType:  toolType,
		ModelName: modelName,
	}

	// 根据工具类型设置环境变量key
	switch toolType {
	case ToolTypeCodex:
		newMirror.EnvKey = CodexSwitchAPIKeyEnv // Codex 固定使用专用的环境变量名
	case ToolTypeClaude:
		newMirror.EnvKey = "ANTHROPIC_AUTH_TOKEN" // Claude 使用固定的环境变量名
	}

	mm.config.Mirrors = append(mm.config.Mirrors, newMirror)

	// 如果是第一个该类型的配置，设置为当前激活的配置
	if toolType == ToolTypeCodex && mm.config.CurrentCodex == "" {
		mm.config.CurrentCodex = name
	} else if toolType == ToolTypeClaude && mm.config.CurrentClaude == "" {
		mm.config.CurrentClaude = name
	}

	return mm.saveConfig()
}

// RemoveMirror 删除镜像源.
func (mm *MirrorManager) RemoveMirror(name string) error {
	if name == DefaultMirrorName {
		return fmt.Errorf("不能删除官方镜像源")
	}

	for i, mirror := range mm.config.Mirrors {
		if mirror.Name != name {
			continue
		}

		// 如果删除的是当前使用的镜像源，切换到官方镜像源
		if mm.config.CurrentMirror == name {
			mm.config.CurrentMirror = DefaultMirrorName
		}
		if mm.config.CurrentCodex == name {
			mm.config.CurrentCodex = DefaultMirrorName
		}
		if mm.config.CurrentClaude == name {
			mm.config.CurrentClaude = ""
		}

		// 删除镜像源
		mm.config.Mirrors = append(mm.config.Mirrors[:i], mm.config.Mirrors[i+1:]...)
		return mm.saveConfig()
	}

	return fmt.Errorf("镜像源 '%s' 不存在", name)
}

// ListMirrors 列出所有镜像源.
func (mm *MirrorManager) ListMirrors() []MirrorConfig {
	return mm.config.Mirrors
}

// GetCurrentMirror 获取当前镜像源.
func (mm *MirrorManager) GetCurrentMirror() (*MirrorConfig, error) {
	for _, mirror := range mm.config.Mirrors {
		if mirror.Name == mm.config.CurrentMirror {
			return &mirror, nil
		}
	}
	return nil, fmt.Errorf("当前镜像源 '%s' 不存在", mm.config.CurrentMirror)
}

// GetCurrentCodexMirror 获取当前激活的 Codex 镜像源.
func (mm *MirrorManager) GetCurrentCodexMirror() (*MirrorConfig, error) {
	for _, mirror := range mm.config.Mirrors {
		if mirror.Name == mm.config.CurrentCodex && mirror.ToolType == ToolTypeCodex {
			return &mirror, nil
		}
	}
	return nil, fmt.Errorf("当前 Codex 镜像源 '%s' 不存在", mm.config.CurrentCodex)
}

// GetCurrentClaudeMirror 获取当前激活的 Claude 镜像源.
func (mm *MirrorManager) GetCurrentClaudeMirror() (*MirrorConfig, error) {
	for _, mirror := range mm.config.Mirrors {
		if mirror.Name == mm.config.CurrentClaude && mirror.ToolType == ToolTypeClaude {
			return &mirror, nil
		}
	}
	return nil, fmt.Errorf("当前 Claude 镜像源 '%s' 不存在", mm.config.CurrentClaude)
}

// GetMirrorByName 根据名称获取镜像源配置.
func (mm *MirrorManager) GetMirrorByName(name string) (*MirrorConfig, error) {
	for _, mirror := range mm.config.Mirrors {
		if mirror.Name == name {
			return &mirror, nil
		}
	}
	return nil, fmt.Errorf("镜像源 '%s' 不存在", name)
}

// SwitchMirror 切换镜像源.
func (mm *MirrorManager) SwitchMirror(name string) error {
	// 检查镜像源是否存在
	for _, mirror := range mm.config.Mirrors {
		if mirror.Name == name {
			mm.config.CurrentMirror = name

			// 根据工具类型设置当前激活的配置
			switch mirror.ToolType {
			case ToolTypeCodex:
				mm.config.CurrentCodex = name
			case ToolTypeClaude:
				mm.config.CurrentClaude = name
			}

			return mm.saveConfig()
		}
	}

	return fmt.Errorf("镜像源 '%s' 不存在", name)
}

// UpdateMirror 更新镜像源.
func (mm *MirrorManager) UpdateMirror(name, baseURL, apiKey string) error {
	for i, mirror := range mm.config.Mirrors {
		if mirror.Name == name {
			if baseURL != "" {
				mm.config.Mirrors[i].BaseURL = baseURL
			}
			if apiKey != "" {
				mm.config.Mirrors[i].APIKey = apiKey
			}
			return mm.saveConfig()
		}
	}

	return fmt.Errorf("镜像源 '%s' 不存在", name)
}

// FixEnvKeyFormat 修复所有镜像源的env_key格式.
func (mm *MirrorManager) FixEnvKeyFormat() error {
	updated := false

	// 检查并修复每个镜像源的env_key格式
	for i, mirror := range mm.config.Mirrors {
		var expectedEnvKey string
		switch mirror.ToolType {
		case ToolTypeCodex:
			expectedEnvKey = CodexSwitchAPIKeyEnv // Codex 固定使用专用的环境变量名
		case ToolTypeClaude:
			expectedEnvKey = "ANTHROPIC_AUTH_TOKEN"
		default:
			continue // 跳过未知类型
		}

		// 如果env_key为空或者格式不正确，都需要修复
		if mirror.EnvKey == "" || mirror.EnvKey != expectedEnvKey {
			fmt.Printf("修复镜像源 '%s' 的env_key: '%s' -> '%s'\n", mirror.Name, mirror.EnvKey, expectedEnvKey)
			mm.config.Mirrors[i].EnvKey = expectedEnvKey
			updated = true
		}
	}

	// 如果有更新，保存配置文件
	if updated {
		fmt.Println("保存更新后的mirrors.toml配置...")
		return mm.saveConfig()
	}

	return nil
}

// discoverFromEnvironment 从环境变量和配置文件中发现并初始化镜像源配置.
func (mm *MirrorManager) discoverFromEnvironment() {
	// 首先初始化默认配置作为基础
	mm.initDefaultConfig()

	discoveredMirrors := make(map[string]MirrorConfig)

	// 1. 从 ~/.codex/config.toml 中发现 Codex 配置
	mm.discoverCodexFromConfig(discoveredMirrors)

	// 2. 从环境变量中发现 Claude 配置
	mm.discoverClaudeFromEnv(discoveredMirrors)

	// 3. 从环境变量中发现 Codex 配置（作为补充）
	mm.discoverCodexFromEnv(discoveredMirrors)

	// 将发现的镜像源添加到配置中
	for name, mirror := range discoveredMirrors {
		// 检查是否已经存在于默认配置中
		exists := false
		for _, existingMirror := range mm.config.Mirrors {
			if existingMirror.Name == name {
				exists = true
				break
			}
		}

		if !exists {
			mm.config.Mirrors = append(mm.config.Mirrors, mirror)

			// 设置当前激活的配置
			switch mirror.ToolType {
			case ToolTypeCodex:
				if mm.config.CurrentCodex == DefaultMirrorName {
					mm.config.CurrentCodex = name
				}
			case ToolTypeClaude:
				if mm.config.CurrentClaude == "" {
					mm.config.CurrentClaude = name
				}
			}

			// 设置通用当前镜像源（兼容旧版本）
			if mm.config.CurrentMirror == DefaultMirrorName && len(mm.config.Mirrors) > 1 {
				mm.config.CurrentMirror = name
			}
		}
	}

	// 保存发现的配置
	if len(discoveredMirrors) > 0 {
		if err := mm.saveConfig(); err != nil {
			fmt.Printf("保存配置失败: %v\n", err)
		}
	}
}

// discoverCodexFromConfig 从 ~/.codex/config.toml 文件中发现 Codex 配置.
func (mm *MirrorManager) discoverCodexFromConfig(discoveredMirrors map[string]MirrorConfig) {
	// 获取 Codex 配置文件路径
	configPath, err := GetCodexConfigPath()
	if err != nil {
		return // 无法获取路径，跳过
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return // 配置文件不存在，跳过
	}

	// 读取配置文件
	var config CodexConfig
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return // 解析失败，跳过
	}

	// 获取认证文件中的 API 密钥
	authPath, err := GetCodexAuthPath()
	if err != nil {
		return // 无法获取认证路径，跳过
	}

	var auth CodexAuth
	if _, err := os.Stat(authPath); err == nil {
		// 认证文件存在，读取 API 密钥
		if file, err := os.Open(authPath); err == nil {
			defer func() {
				if closeErr := file.Close(); closeErr != nil {
					fmt.Printf("关闭文件失败: %v\n", closeErr)
				}
			}()
			_ = json.NewDecoder(file).Decode(&auth)
		}
	}

	// 遍历所有模型提供商配置
	if config.ModelProviders != nil {
		for name, provider := range config.ModelProviders {
			// 跳过空配置
			if provider.BaseURL == "" {
				continue
			}

			// 智能获取 API 密钥
			apiKey := mm.getApiKeyForProvider(auth)

			// 创建镜像源配置
			mirror := MirrorConfig{
				Name:     name,
				BaseURL:  provider.BaseURL,
				APIKey:   apiKey,
				EnvKey:   provider.EnvKey,
				ToolType: ToolTypeCodex,
			}

			discoveredMirrors[name] = mirror
		}
	}
}

// getApiKeyForProvider 智能获取提供商的 API 密钥.
func (mm *MirrorManager) getApiKeyForProvider(auth CodexAuth) string {
	// Codex 固定使用 CODEX_SWITCH_OPENAI_API_KEY 环境变量
	envKey := CodexSwitchAPIKeyEnv

	// 1. 首先尝试从环境变量获取
	if apiKey := os.Getenv(envKey); apiKey != "" {
		return apiKey
	}

	// 2. 尝试从认证文件获取
	if auth.APIKey != "" {
		return auth.APIKey
	}

	// 3. 如果都没有找到，返回空字符串
	return ""
}

// discoverClaudeFromEnv 从环境变量中发现 Claude 配置.
func (mm *MirrorManager) discoverClaudeFromEnv(discoveredMirrors map[string]MirrorConfig) {
	// 检查 Claude 相关的环境变量
	if anthropicBaseURL := os.Getenv("ANTHROPIC_BASE_URL"); anthropicBaseURL != "" {
		if authToken := os.Getenv("ANTHROPIC_AUTH_TOKEN"); authToken != "" {
			// 从 URL 提取镜像源名称
			mirrorName := extractMirrorNameFromURL(anthropicBaseURL, "claude")

			// 发现 Claude 配置
			discoveredMirrors[mirrorName] = MirrorConfig{
				Name:     mirrorName,
				BaseURL:  anthropicBaseURL,
				APIKey:   authToken,
				EnvKey:   "ANTHROPIC_AUTH_TOKEN",
				ToolType: ToolTypeClaude,
			}
		}
	}

	// 检查旧的 ANTHROPIC_API_KEY（兼容性）
	if anthropicAPIKey := os.Getenv("ANTHROPIC_API_KEY"); anthropicAPIKey != "" {
		if _, exists := discoveredMirrors["claude"]; !exists {
			discoveredMirrors["claude"] = MirrorConfig{
				Name:     "claude",
				BaseURL:  "https://api.anthropic.com",
				APIKey:   anthropicAPIKey,
				EnvKey:   "ANTHROPIC_AUTH_TOKEN",
				ToolType: ToolTypeClaude,
			}
		}
	}
}

// discoverCodexFromEnv 从环境变量中发现 Codex 配置（作为补充）.
func (mm *MirrorManager) discoverCodexFromEnv(discoveredMirrors map[string]MirrorConfig) {
	// 扫描所有环境变量，寻找可能相关的API密钥
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}

		key, value := pair[0], pair[1]

		// 跳过空值
		if value == "" {
			continue
		}

		// 检查是否匹配任何API密钥模式
		var mirrorName string
		var baseURL string
		var toolType ToolType

		switch {
		case key == "OPENAI_API_KEY":
			mirrorName = "openai"
			baseURL = "https://api.openai.com"
			toolType = ToolTypeCodex
		case strings.HasPrefix(key, "CODEX_") && strings.HasSuffix(key, "_API_KEY"):
			// 提取镜像源名称: CODEX_<NAME>_API_KEY -> <name>
			namePart := strings.TrimPrefix(key, "CODEX_")
			namePart = strings.TrimSuffix(namePart, "_API_KEY")
			if namePart != "" {
				mirrorName = strings.ToLower(namePart)
				baseURL = "https://api.example.com" // 通用URL，用户可以后续修改
				toolType = ToolTypeCodex
			}
		}

		if mirrorName != "" {
			// 检查是否已经存在该镜像源（避免重复）
			if _, exists := discoveredMirrors[mirrorName]; !exists {
				discoveredMirrors[mirrorName] = MirrorConfig{
					Name:     mirrorName,
					BaseURL:  baseURL,
					APIKey:   value,
					EnvKey:   key,
					ToolType: toolType,
				}
			}
		}
	}
}

// SanitizeEnvVarName 将镜像源名称转换为合法的环境变量名称部分.
func SanitizeEnvVarName(name string) string {
	// 将连字符替换为下划线，移除其他特殊字符
	sanitized := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-' || r == ' ':
			return '_'
		default:
			return -1 // 删除其他字符
		}
	}, name)

	// 确保不以数字开头
	if sanitized != "" && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "MIRROR_" + sanitized
	}

	// 如果为空，使用默认名称
	if sanitized == "" {
		sanitized = "MIRROR"
	}

	return strings.ToUpper(sanitized)
}

// extractMirrorNameFromURL 从 URL 中提取镜像源名称.
func extractMirrorNameFromURL(urlStr, defaultName string) string {
	// 解析 URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return defaultName
	}

	// 获取主机名
	host := u.Hostname()
	if host == "" {
		return defaultName
	}

	// 移除端口
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	// 分割域名部分
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return defaultName
	}

	// 提取主域名部分
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		// 跳过常见的 TLD
		if part == "com" || part == "org" || part == "net" || part == "cn" ||
			part == "io" || part == "ai" || part == "dev" || part == "app" {
			continue
		}
		// 找到主域名
		if i > 0 {
			return parts[i-1] + "-" + part
		}
		return part
	}

	return defaultName
}
