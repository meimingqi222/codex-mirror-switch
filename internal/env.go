package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EnvManager 环境变量管理器.
type EnvManager struct{}

// NewEnvManager 创建新的环境变量管理器.
func NewEnvManager() *EnvManager {
	return &EnvManager{}
}

// SetClaudeEnvVars 设置 Claude Code 环境变量.
func (em *EnvManager) SetClaudeEnvVars(baseURL, authToken string) error {
	return em.SetClaudeEnvVarsWithModel(baseURL, authToken, "")
}

// SetClaudeEnvVarsWithModel 设置 Claude Code 环境变量（包括可选的模型名称）.
func (em *EnvManager) SetClaudeEnvVarsWithModel(baseURL, authToken, modelName string) error {
	// 设置 ANTHROPIC_BASE_URL (不显示刷新提示)
	if err := em.setEnvironmentVariableNoRefresh(AnthropicBaseURLEnv, baseURL); err != nil {
		return fmt.Errorf("设置 ANTHROPIC_BASE_URL 失败: %v", err)
	}

	// 设置 ANTHROPIC_AUTH_TOKEN (不显示刷新提示)
	if err := em.setEnvironmentVariableNoRefresh(AnthropicAuthTokenEnv, authToken); err != nil {
		return fmt.Errorf("设置 ANTHROPIC_AUTH_TOKEN 失败: %v", err)
	}

	// 设置 ANTHROPIC_MODEL (可选)
	if modelName != "" {
		if err := em.setEnvironmentVariableNoRefresh(AnthropicModelEnv, modelName); err != nil {
			return fmt.Errorf("设置 ANTHROPIC_MODEL 失败: %v", err)
		}
	} else {
		// 如果模型名称为空，尝试清除现有的 ANTHROPIC_MODEL 环境变量
		em.unsetEnvironmentVariable(AnthropicModelEnv)
	}

	// 一次性显示刷新提示
	return em.showRefreshInstructions()
}

// SetCodexEnvVar 设置 Codex 环境变量.
func (em *EnvManager) SetCodexEnvVar(envKey, apiKey string) error {
	if envKey == "" {
		return fmt.Errorf("环境变量 key 不能为空")
	}

	return em.setEnvironmentVariable(envKey, apiKey)
}

// setEnvironmentVariable 设置环境变量（跨平台）.
func (em *EnvManager) setEnvironmentVariable(envKey, value string) error {
	if err := em.setEnvironmentVariableNoRefresh(envKey, value); err != nil {
		return err
	}
	return em.showRefreshInstructions()
}

// setEnvironmentVariableNoRefresh 设置环境变量（不显示刷新提示）.
func (em *EnvManager) setEnvironmentVariableNoRefresh(envKey, value string) error {
	// 在当前进程中设置环境变量
	if err := os.Setenv(envKey, value); err != nil {
		return fmt.Errorf("设置环境变量 %s 失败: %v", envKey, err)
	}

	// 根据平台设置持久化环境变量
	platform := GetCurrentPlatform()
	var err error
	switch platform {
	case PlatformWindows:
		err = em.setWindowsUserEnvVarNoRefresh(envKey, value)
	case PlatformMac:
		shellFiles := []string{".zshrc"} // macOS 默认使用 zsh
		err = setUnixUserEnvVar(envKey, value, shellFiles)
	case PlatformLinux:
		shellFiles := []string{".bashrc", ".profile"} // bash (最常见), 通用 profile
		err = setUnixUserEnvVar(envKey, value, shellFiles)
	}

	if err != nil {
		return fmt.Errorf("设置 %s 用户环境变量 %s 失败: %v", platform, envKey, err)
	}

	return nil
}

// setWindowsUserEnvVarNoRefresh 在 Windows 中设置用户级环境变量（不显示刷新提示）.
func (em *EnvManager) setWindowsUserEnvVarNoRefresh(envKey, value string) error {
	// 使用 setx 命令设置用户级环境变量
	cmd := exec.Command("setx", envKey, value)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("执行 setx 命令失败: %v, 输出: %s", err, string(output))
	}
	fmt.Printf("[OK] 环境变量 %s 已设置\n", envKey)
	return nil
}

// setUnixUserEnvVar 在 Unix 系统（macOS 和 Linux）中设置用户级环境变量.
func setUnixUserEnvVar(envKey, value string, shellFileNames []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %v", err)
	}

	// 构建完整的文件路径
	shellFiles := make([]string, len(shellFileNames))
	for i, name := range shellFileNames {
		shellFiles[i] = filepath.Join(homeDir, name)
	}

	envLine := fmt.Sprintf("export %s=%s", envKey, value)
	updated := false

	for _, shellFile := range shellFiles {
		if err := updateShellProfile(shellFile, envKey, envLine); err != nil {
			fmt.Printf("警告: 更新 %s 失败: %v\n", shellFile, err)
			continue
		}
		updated = true
	}

	if !updated {
		return fmt.Errorf("无法更新任何 shell 配置文件")
	}

	fmt.Printf("[OK] 环境变量 %s 已添加到 shell 配置文件\n", envKey)
	return nil
}

// updateShellProfile 更新 shell 配置文件，添加或更新环境变量.
func updateShellProfile(shellFile, envKey, envLine string) error {
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

	// 对于所有 Codex 相关的环境变量，先清理所有相关的旧环境变量
	if strings.HasPrefix(envKey, "CODEX_") {
		lines = cleanupOldCodexEnvVars(lines)
	}

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
		lines = append(lines, "", "# Codex Mirror Switch - API Key", envLine)
	}

	// 清理多余的空行和连续的注释
	lines = cleanupExtraLines(lines)

	// 写回文件
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(shellFile, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}

// cleanupOldCodexEnvVars 清理旧的 Codex 相关环境变量.
func cleanupOldCodexEnvVars(lines []string) []string {
	var cleanedLines []string
	i := 0

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// 只清理由本工具写入的标记段落：
		// 1. 注释行必须是精确的 "# Codex Mirror Switch - API Key" 或带句号版本；
		// 2. 紧随其后的下一行必须是以 "export CODEX_" 开头且包含 "_API_KEY=" 的环境变量行。
		// 满足这两个条件时，一并删除注释行和变量行；否则两行都保留。
		if (trimmed == "# Codex Mirror Switch - API Key" || trimmed == "# Codex Mirror Switch - API Key.") &&
			i+1 < len(lines) {
			nextLine := lines[i+1]
			nextTrimmed := strings.TrimSpace(nextLine)

			if strings.HasPrefix(nextTrimmed, "export CODEX_") && strings.Contains(nextTrimmed, "_API_KEY=") {
				// 删除该注释行和紧随其后的 CODEX_*_API_KEY 行
				i += 2
				continue
			}
		}

		// 其他行原样保留
		cleanedLines = append(cleanedLines, line)
		i++
	}

	return cleanedLines
}

// cleanupExtraLines 清理多余的空行和连续的注释.
func cleanupExtraLines(lines []string) []string {
	var cleanedLines []string
	prevEmpty := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isEmpty := trimmed == ""

		// 跳过连续的空行
		if isEmpty && prevEmpty {
			continue
		}

		// 跳过孤立的 "# Codex Mirror Switch - API Key" 注释行
		if trimmed == "# Codex Mirror Switch - API Key" || trimmed == "# Codex Mirror Switch - API Key." {
			// 检查下一行是否是要保留的环境变量
			continue
		}

		cleanedLines = append(cleanedLines, line)
		prevEmpty = isEmpty
	}

	// 移除末尾的空行
	for len(cleanedLines) > 0 && strings.TrimSpace(cleanedLines[len(cleanedLines)-1]) == "" {
		cleanedLines = cleanedLines[:len(cleanedLines)-1]
	}

	return cleanedLines
}

// CleanupOldCodexEnvVars 导出版本的清理函数，用于测试.
func CleanupOldCodexEnvVars(lines []string) []string {
	return cleanupOldCodexEnvVars(lines)
}

// unsetEnvironmentVariable 清除环境变量(适用于可选变量).
func (em *EnvManager) unsetEnvironmentVariable(envKey string) {
	// 从当前进程中移除环境变量
	_ = os.Unsetenv(envKey)

	// 从持久化存储中移除环境变量定义
	platform := GetCurrentPlatform()

	switch platform {
	case PlatformWindows:
		// Windows 上使用 reg delete 从注册表删除用户环境变量
		// setx 设置空字符串不等于删除，必须使用 reg delete
		cmd := exec.Command("reg", "delete", "HKCU\\Environment", "/v", envKey, "/f")
		output, err := cmd.CombinedOutput()
		if err != nil {
			// 如果变量不存在，reg delete 会返回错误，这是预期行为
			if !strings.Contains(string(output), "找不到") && !strings.Contains(string(output), "not found") {
				fmt.Printf("警告: 清除环境变量 %s 失败: %v\n", envKey, err)
			}
		} else {
			fmt.Printf("[OK] 环境变量 %s 已清除\n", envKey)
			// 通知系统环境变量已更改（可选，需要广播 WM_SETTINGCHANGE）
		}
		return
	case PlatformMac:
		shellFiles := []string{".zshrc"}
		em.unsetUnixEnvVar(envKey, shellFiles)
	case PlatformLinux:
		shellFiles := []string{".bashrc", ".profile"}
		em.unsetUnixEnvVar(envKey, shellFiles)
	}
}

// unsetUnixEnvVar 在Unix系统中从shell配置文件中移除环境变量.
func (em *EnvManager) unsetUnixEnvVar(envKey string, shellFiles []string) {
	// 从所有 shell 配置文件中移除环境变量
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// 如果无法获取主目录，就直接跳过
		return
	}

	updated := false
	for _, shellFileName := range shellFiles {
		shellFile := filepath.Join(homeDir, shellFileName)
		if _, err := os.Stat(shellFile); os.IsNotExist(err) {
			continue
		}

		// 读取文件内容
		content, err := os.ReadFile(shellFile)
		if err != nil {
			// 如果读取失败，跳过这个文件
			continue
		}

		// 分行处理
		lines := strings.Split(string(content), "\n")
		var newLines []string

		envPattern := fmt.Sprintf("export %s=", envKey)
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			// 只删除以环境变量开头的行，避免误删
			if !strings.HasPrefix(trimmedLine, envPattern) {
				newLines = append(newLines, line)
			}
		}

		// 写回文件
		if err := os.WriteFile(shellFile, []byte(strings.Join(newLines, "\n")), 0o644); err != nil {
			// 如果写入失败，跳过这个文件
			continue
		}
		updated = true
	}

	if updated {
		fmt.Printf("[OK] 环境变量 %s 已从 shell 配置文件中清除\n", envKey)
	}
}

// showRefreshInstructions 显示环境变量刷新指导.
func (em *EnvManager) showRefreshInstructions() error {
	platform := GetCurrentPlatform()
	if platform == PlatformWindows {
		fmt.Println("\n📝 环境变量已设置")
		fmt.Println("🔄 请重新启动终端或注销重新登录以应用更改")
		return nil
	}

	// macOS 和 Linux
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %v", err)
	}

	var shellFiles []string
	if platform == PlatformMac {
		shellFiles = []string{".zshrc"}
	} else {
		shellFiles = []string{".bashrc", ".profile"}
	}

	// 显示刷新提示信息
	fmt.Println("\n📝 环境变量已写入配置文件")
	fmt.Println("\n💡 要使环境变量在当前终端生效，请执行以下命令之一:")

	for _, shellFileName := range shellFiles {
		shellFile := filepath.Join(homeDir, shellFileName)
		if _, err := os.Stat(shellFile); err == nil {
			fmt.Printf("   source %s\n", shellFile)
			break // 只显示第一个存在的文件
		}
	}

	fmt.Println("\n🔄 或者重新启动终端/打开新的终端窗口")
	fmt.Println("\n⚡ 提示: 新打开的终端窗口会自动应用环境变量更改")

	return nil
}
