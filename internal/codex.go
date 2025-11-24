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

// UpdateConfig 更新Codex配置文件，根据提供的镜像配置更新或添加相应的模型提供商配置。
func (ccm *CodexConfigManager) UpdateConfig(mirror *MirrorConfig) error {
	config, rawConfig, err := ccm.loadExistingConfig()
	if err != nil {
		return err
	}

	providerConfig := ccm.createProviderConfig(mirror, config)
	ccm.updateConfigStructures(config, rawConfig, mirror, providerConfig)

	return ccm.writeConfigFile(rawConfig)
}

// loadExistingConfig 加载现有的Codex配置文件.
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

// decodeConfigFiles 解码Codex配置文件.
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

// createProviderConfig 根据提供的镜像配置创建模型提供商配置.
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

// mergeExistingProviderConfig 合并现有的模型提供商配置.
func (ccm *CodexConfigManager) mergeExistingProviderConfig(providerConfig *ModelProviderConfig, existingProvider ModelProviderConfig) {
	if existingProvider.WireAPI != "" {
		providerConfig.WireAPI = existingProvider.WireAPI
	}

	if existingProvider.EnvKey == CodexSwitchAPIKeyEnv {
		providerConfig.EnvKey = existingProvider.EnvKey
	}

	providerConfig.RequiresOpenAIAuth = existingProvider.RequiresOpenAIAuth
}

// updateConfigStructures 更新配置结构体和原始配置.
func (ccm *CodexConfigManager) updateConfigStructures(config *CodexConfig, rawConfig map[string]interface{}, mirror *MirrorConfig, providerConfig ModelProviderConfig) {
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
	config.ModelProviders[mirror.Name] = providerConfig

	if rawConfig == nil {
		rawConfig = make(map[string]interface{})
	}

	ccm.updateRawConfigBasicFields(rawConfig, config, mirror)
	ccm.updateRawConfigModelProviders(rawConfig, mirror.Name, providerConfig, existingProviders)
}

// updateRawConfigBasicFields 更新原始配置中的基础字段.
func (ccm *CodexConfigManager) updateRawConfigBasicFields(rawConfig map[string]interface{}, config *CodexConfig, mirror *MirrorConfig) {
	rawConfig["model_provider"] = mirror.Name

	// 更新 Model 字段 - 使用 mirror 中的 ModelName，如果没有则使用默认值
	if mirror.ModelName != "" {
		config.Model = mirror.ModelName
	} else {
		config.Model = TestModelGPT5
	}
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

// updateRawConfigModelProviders 更新原始配置中的模型提供商配置.
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

// writeConfigFile 将配置写入文件（保留所有原始字段）.
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

	// 分类并写入配置
	return ccm.writeConfigSections(file, rawConfig)
}

// writeConfigSections 分类并写入配置的各个部分.
func (ccm *CodexConfigManager) writeConfigSections(file *os.File, rawConfig map[string]interface{}) error {
	// 分离不同类型的键
	basicKeys, dottedKeys, topLevelMaps := classifyConfigKeys(rawConfig)

	// 1. 写入基本配置项
	if err := writeBasicKeys(file, rawConfig, basicKeys); err != nil {
		return err
	}

	// 2. 写入带点的节
	if err := ccm.writeDottedKeys(file, rawConfig, dottedKeys); err != nil {
		return err
	}

	// 3. 写入顶级map
	return writeTopLevelMaps(file, rawConfig, topLevelMaps)
}

// classifyConfigKeys 将配置键分类为不同类型.
func classifyConfigKeys(rawConfig map[string]interface{}) (basicKeys, dottedKeys, topLevelMaps map[string]bool) {
	basicKeys = make(map[string]bool)
	dottedKeys = make(map[string]bool)
	topLevelMaps = make(map[string]bool)

	for key, value := range rawConfig {
		switch {
		case strings.Contains(key, "."):
			dottedKeys[key] = true
		case isMap(value):
			topLevelMaps[key] = true
		default:
			basicKeys[key] = true
		}
	}
	return
}

// writeBasicKeys 写入基本配置项（不包含点的简单值）.
func writeBasicKeys(file *os.File, rawConfig map[string]interface{}, basicKeys map[string]bool) error {
	for key, value := range rawConfig {
		if basicKeys[key] {
			if err := writeTOMLValue(file, key, value, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeDottedKeys 写入带点的节（保留所有原始的带点的键）.
func (ccm *CodexConfigManager) writeDottedKeys(file *os.File, rawConfig map[string]interface{}, dottedKeys map[string]bool) error {
	for key, value := range rawConfig {
		if dottedKeys[key] {
			if subMap, ok := value.(map[string]interface{}); ok {
				if err := writeDottedSection(file, key, subMap); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// writeDottedSection 写入一个带点的节，处理其简单值和嵌套map.
func writeDottedSection(file *os.File, key string, subMap map[string]interface{}) error {
	// 分离嵌套map和简单值
	nestedMaps := make(map[string]map[string]interface{})
	simpleValues := make(map[string]interface{})

	for k, v := range subMap {
		if nestedMap, isMap := v.(map[string]interface{}); isMap {
			nestedMaps[k] = nestedMap
		} else {
			simpleValues[k] = v
		}
	}

	// 先写入当前节的简单值（如果有）
	if len(simpleValues) > 0 {
		if _, err := fmt.Fprintf(file, "\n[%s]\n", key); err != nil {
			return err
		}
		for k, v := range simpleValues {
			if err := writeTOMLValue(file, k, v, "  "); err != nil {
				return err
			}
		}
	}

	// 递归处理嵌套的map（作为独立的节）
	for k, nestedMap := range nestedMaps {
		nestedKey := key + "." + k
		if _, err := fmt.Fprintf(file, "\n[%s]\n", nestedKey); err != nil {
			return err
		}
		if err := writeTOMLMap(file, nestedMap, "  "); err != nil {
			return err
		}
	}
	return nil
}

// writeTopLevelMaps 写入顶级map.
func writeTopLevelMaps(file *os.File, rawConfig map[string]interface{}, topLevelMaps map[string]bool) error {
	for key, value := range rawConfig {
		if topLevelMaps[key] {
			if subMap, ok := value.(map[string]interface{}); ok {
				if err := writeTopLevelMapAsSections(file, key, subMap); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// writeTopLevelMapAsSections 将顶级map写入为带点的节.
// 例如: projects map 转换为 [projects."/path"] 节.
func writeTopLevelMapAsSections(file *os.File, prefix string, m map[string]interface{}) error {
	for key, value := range m {
		fullKey := prefix + "." + key
		if subMap, ok := value.(map[string]interface{}); ok {
			// 分离嵌套map和简单值
			nestedMaps := make(map[string]map[string]interface{})
			simpleValues := make(map[string]interface{})

			for k, v := range subMap {
				if nestedMap, isMap := v.(map[string]interface{}); isMap {
					nestedMaps[k] = nestedMap
				} else {
					simpleValues[k] = v
				}
			}

			// 先写入当前节的简单值（如果有）
			if len(simpleValues) > 0 {
				if _, err := fmt.Fprintf(file, "\n[%s]\n", fullKey); err != nil {
					return err
				}
				// 只写入简单值，不包括嵌套map
				for k, v := range simpleValues {
					if err := writeTOMLValue(file, k, v, "  "); err != nil {
						return err
					}
				}
			}

			// 递归处理嵌套的map（作为独立的节）
			for k, nestedMap := range nestedMaps {
				nestedKey := fullKey + "." + k

				// 再次分离嵌套map中的简单值和嵌套map
				nestedNestedMaps := make(map[string]map[string]interface{})
				nestedSimpleValues := make(map[string]interface{})

				for nk, nv := range nestedMap {
					if nnMap, isMap := nv.(map[string]interface{}); isMap {
						nestedNestedMaps[nk] = nnMap
					} else {
						nestedSimpleValues[nk] = nv
					}
				}

				// 写入嵌套节的简单值
				if _, err := fmt.Fprintf(file, "\n[%s]\n", nestedKey); err != nil {
					return err
				}
				for nk, nv := range nestedSimpleValues {
					if err := writeTOMLValue(file, nk, nv, "  "); err != nil {
						return err
					}
				}

				// 递归处理更深层的嵌套
				for nnk, nnMap := range nestedNestedMaps {
					deepKey := nestedKey + "." + nnk
					if _, err := fmt.Fprintf(file, "\n[%s]\n", deepKey); err != nil {
						return err
					}
					if err := writeTOMLMap(file, nnMap, "  "); err != nil {
						return err
					}
				}
			}
		} else {
			// 不应该发生，跳过
			continue
		}
	}
	return nil
}

// isMap 判断给定的值是否为 map[string]interface{}.
func isMap(value interface{}) bool {
	_, ok := value.(map[string]interface{})
	return ok
}

// writeTOMLMap 将 map[string]interface{} 写入 TOML 文件（标准格式）.
func writeTOMLMap(file *os.File, m map[string]interface{}, indent string) error {
	// 标准格式：每个键值对单独一行
	// 跳过map类型的值，这些应该由调用者处理为独立的节
	for key, value := range m {
		if _, isMap := value.(map[string]interface{}); isMap {
			// 跳过map类型，让调用者处理为独立节
			continue
		}
		if err := writeTOMLValue(file, key, value, indent); err != nil {
			return err
		}
	}
	return nil
}

// shouldUseInlineTable 判断是否应该使用内联表格式.
func shouldUseInlineTable(m map[string]interface{}) bool {
	// 如果map只包含简单类型（字符串、数字、布尔值、数组），使用内联表
	for _, value := range m {
		switch value.(type) {
		case string, int, int32, int64, float32, float64, bool, []interface{}:
			// 简单类型，适合内联表
			continue
		case map[string]interface{}:
			// 嵌套map，不适合内联表
			return false
		default:
			return false
		}
	}
	return true
}

// writeInlineTableValue 写入内联表的值.
func writeInlineTableValue(file *os.File, key string, value interface{}) error {
	switch v := value.(type) {
	case string:
		_, err := fmt.Fprintf(file, "%s = %q", key, v)
		return err
	case bool:
		_, err := fmt.Fprintf(file, "%s = %t", key, v)
		return err
	case int, int32, int64:
		_, err := fmt.Fprintf(file, "%s = %d", key, v)
		return err
	case float32, float64:
		_, err := fmt.Fprintf(file, "%s = %f", key, v)
		return err
	case []interface{}:
		// 内联数组
		if _, err := fmt.Fprintf(file, "%s = [", key); err != nil {
			return err
		}
		for i, item := range v {
			if i > 0 {
				if _, err := fmt.Fprintf(file, ", "); err != nil {
					return err
				}
			}
			if s, ok := item.(string); ok {
				if _, err := fmt.Fprintf(file, "%q", s); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(file, "%v", item); err != nil {
					return err
				}
			}
		}
		_, err := fmt.Fprintf(file, "]")
		return err
	default:
		_, err := fmt.Fprintf(file, "%s = %v", key, v)
		return err
	}
}

// writeTOMLValue 将单个键值对写入 TOML 文件.
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
	case []interface{}:
		// 处理数组
		return writeTOMLArray(file, key, v, indent)
	case map[string]interface{}:
		// map 值：检查是否应该使用内联表
		if shouldUseInlineTable(v) {
			// 内联表格式：key = { field1 = val1, field2 = val2 }
			if _, err := fmt.Fprintf(file, "%s%s = ", indent, key); err != nil {
				return err
			}
			if err := writeInlineTable(v, file); err != nil {
				return err
			}
			_, err := fmt.Fprintf(file, "\n")
			return err
		}
		// 复杂map，不应该直接作为值
		return fmt.Errorf("复杂map值不支持直接写入: %s", key)
	default:
		_, err := fmt.Fprintf(file, "%s%s = %v\n", indent, key, v)
		return err
	}
}

// writeInlineTable 写入内联表格式: { key1 = val1, key2 = val2 }.
func writeInlineTable(m map[string]interface{}, file *os.File) error {
	if _, err := fmt.Fprintf(file, "{"); err != nil {
		return err
	}
	first := true
	for key, value := range m {
		if !first {
			if _, err := fmt.Fprintf(file, ", "); err != nil {
				return err
			}
		}
		first = false
		if err := writeInlineTableValue(file, key, value); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(file, "}")
	return err
}

// writeTOMLArray 将数组写入 TOML 格式.
func writeTOMLArray(file *os.File, key string, arr []interface{}, indent string) error {
	if _, err := fmt.Fprintf(file, "%s%s = [", indent, key); err != nil {
		return err
	}

	for i, value := range arr {
		if i > 0 {
			if _, err := fmt.Fprintf(file, ", "); err != nil {
				return err
			}
		}

		switch v := value.(type) {
		case string:
			if _, err := fmt.Fprintf(file, "%q", v); err != nil {
				return err
			}
		case int, int32, int64, float32, float64, bool:
			if _, err := fmt.Fprintf(file, "%v", v); err != nil {
				return err
			}
		default:
			if _, err := fmt.Fprintf(file, "%v", v); err != nil {
				return err
			}
		}
	}

	_, err := fmt.Fprintf(file, "]\n")
	return err
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
