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
	// 设置 ANTHROPIC_BASE_URL
	if err := em.setEnvironmentVariable(AnthropicBaseURLEnv, baseURL); err != nil {
		return fmt.Errorf("设置 ANTHROPIC_BASE_URL 失败: %v", err)
	}

	// 设置 ANTHROPIC_AUTH_TOKEN
	if err := em.setEnvironmentVariable(AnthropicAuthTokenEnv, authToken); err != nil {
		return fmt.Errorf("设置 ANTHROPIC_AUTH_TOKEN 失败: %v", err)
	}

	return nil
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
	// 在当前进程中设置环境变量
	if err := os.Setenv(envKey, value); err != nil {
		return fmt.Errorf("设置环境变量 %s 失败: %v", envKey, err)
	}

	// 根据平台设置持久化环境变量
	platform := GetCurrentPlatform()
	var err error
	switch platform {
	case PlatformWindows:
		err = em.setWindowsUserEnvVar(envKey, value)
	case PlatformMac:
		err = em.setMacUserEnvVar(envKey, value)
	case PlatformLinux:
		err = em.setLinuxUserEnvVar(envKey, value)
	}

	if err != nil {
		return fmt.Errorf("设置 %s 用户环境变量 %s 失败: %v", platform, envKey, err)
	}

	return nil
}

// setWindowsUserEnvVar 在 Windows 中设置用户级环境变量.
func (em *EnvManager) setWindowsUserEnvVar(envKey, value string) error {
	// 使用 setx 命令设置用户级环境变量
	cmd := exec.Command("setx", envKey, value)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("执行 setx 命令失败: %v, 输出: %s", err, string(output))
	}
	fmt.Printf("✓ 环境变量 %s 已设置\n", envKey)
	return nil
}

// setMacUserEnvVar 在 macOS 中设置用户级环境变量.
func (em *EnvManager) setMacUserEnvVar(envKey, value string) error {
	shellFiles := []string{".zshrc"} // macOS 默认使用 zsh
	return setUnixUserEnvVar(envKey, value, shellFiles)
}

// setLinuxUserEnvVar 在 Linux 中设置用户级环境变量.
func (em *EnvManager) setLinuxUserEnvVar(envKey, value string) error {
	shellFiles := []string{".bashrc", ".profile"} // bash (最常见), 通用 profile
	return setUnixUserEnvVar(envKey, value, shellFiles)
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

	fmt.Printf("✓ 环境变量 %s 已添加到 shell 配置文件\n", envKey)
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

		// 检查是否是注释行且下一行是要清理的环境变量
		if (trimmed == "# Codex Mirror Switch - API Key" || trimmed == "# Codex Mirror Switch - API Key.") &&
			i+1 < len(lines) {
			nextLine := lines[i+1]
			nextTrimmed := strings.TrimSpace(nextLine)

			// 如果下一行是要清理的环境变量，跳过注释行和环境变量行
			if shouldCleanupEnvVar(nextTrimmed) {
				i += 2 // 跳过注释行和环境变量行
				continue
			}
		}

		// 检查当前行是否是要清理的环境变量
		if shouldCleanupEnvVar(trimmed) {
			i += 1 // 跳过环境变量行
			continue
		}

		// 保留当前行
		cleanedLines = append(cleanedLines, line)
		i += 1
	}

	return cleanedLines
}

// shouldCleanupEnvVar 判断是否应该清理该环境变量行.
func shouldCleanupEnvVar(line string) bool {
	// 跳过旧的 CODEX_*_API_KEY 环境变量
	if strings.HasPrefix(line, "export CODEX_") && strings.HasSuffix(line, "_API_KEY=") {
		return true
	}

	// 跳过 OPENAI_API_KEY 环境变量（避免冲突）
	if strings.HasPrefix(line, "export OPENAI_API_KEY=") {
		return true
	}

	return false
}
