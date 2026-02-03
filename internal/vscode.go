package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VSCodeConfigManager VS Code配置管理器.
type VSCodeConfigManager struct {
	settingsPath string
}

// NewVSCodeConfigManager 创建新的VS Code配置管理器.
func NewVSCodeConfigManager() (*VSCodeConfigManager, error) {
	settingsPath, err := GetVSCodeSettingsPath()
	if err != nil {
		return nil, fmt.Errorf("获取VS Code设置路径失败: %v", err)
	}

	// 确保VS Code配置目录存在
	configDir := filepath.Dir(settingsPath)
	if err := EnsureDir(configDir); err != nil {
		return nil, fmt.Errorf("创建VS Code配置目录失败: %v", err)
	}

	return &VSCodeConfigManager{
		settingsPath: settingsPath,
	}, nil
}

// LoadSettings 加载VS Code设置.
func (vcm *VSCodeConfigManager) LoadSettings() (map[string]interface{}, error) {
	settings := make(map[string]interface{})

	// 如果设置文件不存在，返回空设置
	if _, err := os.Stat(vcm.settingsPath); os.IsNotExist(err) {
		return settings, nil
	}

	file, err := os.Open(vcm.settingsPath)
	if err != nil {
		return nil, fmt.Errorf("打开VS Code设置文件失败: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("警告: 关闭VS Code设置文件失败: %v\n", err)
		}
	}()

	// 读取文件内容
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(file); err != nil {
		return nil, fmt.Errorf("读取VS Code设置文件失败: %v", err)
	}

	// 移除JSONC注释（支持 // 和 /* */）
	cleanedJSON := RemoveJSONComments(buf.String())

	// 解析JSON
	if err := json.Unmarshal([]byte(cleanedJSON), &settings); err != nil {
		return nil, fmt.Errorf("解析VS Code设置文件失败: %v", err)
	}

	return settings, nil
}

// RemoveJSONComments 移除JSONC注释（// 和 /* */）
func RemoveJSONComments(jsonStr string) string {
	var result strings.Builder
	runes := []rune(jsonStr)
	n := len(runes)

	inString := false
	escapeNext := false
	inSingleLineComment := false
	inMultiLineComment := false

	for i := 0; i < n; i++ {
		c := runes[i]

		// 处理转义字符
		if escapeNext {
			result.WriteRune(c)
			escapeNext = false
			continue
		}

		// 字符串内部的转义
		if inString && c == '\\' {
			result.WriteRune(c)
			escapeNext = true
			continue
		}

		// 处理字符串字面量
		if c == '"' && !inSingleLineComment && !inMultiLineComment {
			inString = !inString
			result.WriteRune(c)
			continue
		}

		// 在字符串内时，直接输出字符
		if inString {
			result.WriteRune(c)
			continue
		}

		// 处理单行注释 //
		if i+1 < n && c == '/' && runes[i+1] == '/' && !inMultiLineComment {
			inSingleLineComment = true
			continue
		}

		// 处理多行注释 /*
		if i+1 < n && c == '/' && runes[i+1] == '*' && !inSingleLineComment {
			inMultiLineComment = true
			i++ // 跳过 *
			continue
		}

		// 单行注释结束（遇到换行）
		if inSingleLineComment && c == '\n' {
			inSingleLineComment = false
			result.WriteRune(c)
			continue
		}

		// 多行注释结束
		if inMultiLineComment && i+1 < n && c == '*' && runes[i+1] == '/' {
			inMultiLineComment = false
			i++ // 跳过 /
			continue
		}

		// 在注释内时跳过
		if inSingleLineComment || inMultiLineComment {
			continue
		}

		// 其他字符直接输出
		result.WriteRune(c)
	}

	resultStr := strings.TrimSpace(result.String())
	if resultStr == "" {
		return "{}"
	}

	return resultStr
}

// SaveSettings 保存VS Code设置（使用原子写入）.
func (vcm *VSCodeConfigManager) SaveSettings(settings map[string]interface{}) error {
	// 使用原子写入：先写入临时文件，再通过重命名替换原文件
	configDir := filepath.Dir(vcm.settingsPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("创建VS Code配置目录失败: %v", err)
	}

	tmpFile, err := os.CreateTemp(configDir, "settings-*.json")
	if err != nil {
		return fmt.Errorf("创建临时设置文件失败: %v", err)
	}
	tmpPath := tmpFile.Name()

	// 确保临时文件被清理
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	// 写入 JSON 数据
	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(settings); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("写入临时设置文件失败: %v", err)
	}

	// 确保数据刷入磁盘
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("同步临时设置文件失败: %v", err)
	}

	// 关闭临时文件句柄，避免在 Windows 上影响重命名
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("关闭临时设置文件失败: %v", err)
	}

	// 原子替换目标文件
	if err := os.Rename(tmpPath, vcm.settingsPath); err != nil {
		return fmt.Errorf("替换VS Code设置文件失败: %v", err)
	}

	return nil
}

// ApplyMirror 应用镜像源配置到VS Code.
func (vcm *VSCodeConfigManager) ApplyMirror(mirror *MirrorConfig) error {
	// 加载现有设置
	settings, err := vcm.LoadSettings()
	if err != nil {
		return fmt.Errorf("加载VS Code设置失败: %v", err)
	}

	// 更新chatgpt.apiBase
	settings["chatgpt.apiBase"] = mirror.BaseURL

	// 更新chatgpt.config，只保留基本配置，不设置baseurl和key
	chatgptConfig := make(map[string]interface{})
	if existingConfig, exists := settings["chatgpt.config"]; exists {
		if configMap, ok := existingConfig.(map[string]interface{}); ok {
			chatgptConfig = configMap
		}
	}

	// 设置基本配置项
	chatgptConfig["preferred_auth_method"] = "apikey"
	if mirror.ModelName != "" {
		chatgptConfig["model"] = mirror.ModelName
	} else {
		chatgptConfig["model"] = DefaultModelGPT4
	}
	chatgptConfig["model_reasoning_effort"] = "medium"
	chatgptConfig["wire_api"] = "messages"

	// 移除不必要的baseurl和key设置
	delete(chatgptConfig, "apiKey")
	delete(chatgptConfig, "apiBaseUrl")

	settings["chatgpt.config"] = chatgptConfig

	// 保存设置
	if err := vcm.SaveSettings(settings); err != nil {
		return fmt.Errorf("保存VS Code设置失败: %v", err)
	}

	return nil
}

// GetCurrentConfig 获取当前VS Code中的ChatGPT配置.
func (vcm *VSCodeConfigManager) GetCurrentConfig() (map[string]interface{}, error) {
	settings, err := vcm.LoadSettings()
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})

	// 获取chatgpt.apiBase
	if apiBase, exists := settings["chatgpt.apiBase"]; exists {
		result["apiBase"] = apiBase
	}

	// 获取chatgpt.config
	if config, exists := settings["chatgpt.config"]; exists {
		result["config"] = config
	}

	return result, nil
}

// BackupSettings 备份当前设置.
func (vcm *VSCodeConfigManager) BackupSettings() error {
	if _, err := os.Stat(vcm.settingsPath); os.IsNotExist(err) {
		// 设置文件不存在，无需备份
		return nil
	}

	configDir := filepath.Dir(vcm.settingsPath)
	backupDir := filepath.Join(configDir, "backup")
	if err := EnsureDir(backupDir); err != nil {
		return fmt.Errorf("创建备份目录失败: %v", err)
	}

	backupPath := filepath.Join(backupDir, "settings.json.bak")
	if err := copyFile(vcm.settingsPath, backupPath); err != nil {
		return fmt.Errorf("备份VS Code设置文件失败: %v", err)
	}

	return nil
}

// RemoveChatGPTConfig 移除ChatGPT相关配置.
func (vcm *VSCodeConfigManager) RemoveChatGPTConfig() error {
	settings, err := vcm.LoadSettings()
	if err != nil {
		return fmt.Errorf("加载VS Code设置失败: %v", err)
	}

	// 删除chatgpt相关配置
	delete(settings, "chatgpt.apiBase")
	delete(settings, "chatgpt.config")

	// 保存设置
	if err := vcm.SaveSettings(settings); err != nil {
		return fmt.Errorf("保存VS Code设置失败: %v", err)
	}

	return nil
}

// GetSettingsPath 获取设置文件路径.
func (vcm *VSCodeConfigManager) GetSettingsPath() string {
	return vcm.settingsPath
}
