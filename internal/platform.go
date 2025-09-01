package internal

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetCurrentPlatform 获取当前运行平台.
func GetCurrentPlatform() Platform {
	switch runtime.GOOS {
	case "windows":
		return PlatformWindows
	case "darwin":
		return PlatformMac
	case "linux":
		return PlatformLinux
	default:
		return PlatformLinux // 默认使用Linux路径
	}
}

// GetPathConfig 根据平台获取路径配置.
func GetPathConfig() (*PathConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	platform := GetCurrentPlatform()
	config := &PathConfig{
		HomeDir: homeDir,
	}

	switch platform {
	case PlatformWindows:
		// Windows路径配置
		config.CodexConfigDir = filepath.Join(homeDir, ".codex")
		config.VSCodeConfigDir = filepath.Join(homeDir, "AppData", "Roaming", "Code", "User")

	case PlatformMac:
		// Mac路径配置
		config.CodexConfigDir = filepath.Join(homeDir, ".codex")
		config.VSCodeConfigDir = filepath.Join(homeDir, "Library", "Application Support", "Code", "User")

	case PlatformLinux:
		// Linux路径配置
		config.CodexConfigDir = filepath.Join(homeDir, ".codex")
		config.VSCodeConfigDir = filepath.Join(homeDir, ".config", "Code", "User")
	}

	return config, nil
}

// EnsureDir 确保目录存在，如果不存在则创建.
func EnsureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// GetCodexConfigPath 获取Codex配置文件路径.
func GetCodexConfigPath() (string, error) {
	pathConfig, err := GetPathConfig()
	if err != nil {
		return "", err
	}
	return filepath.Join(pathConfig.CodexConfigDir, "config.toml"), nil
}

// GetCodexAuthPath 获取Codex认证文件路径.
func GetCodexAuthPath() (string, error) {
	pathConfig, err := GetPathConfig()
	if err != nil {
		return "", err
	}
	return filepath.Join(pathConfig.CodexConfigDir, "auth.json"), nil
}

// GetVSCodeSettingsPath 获取VS Code设置文件路径.
func GetVSCodeSettingsPath() (string, error) {
	pathConfig, err := GetPathConfig()
	if err != nil {
		return "", err
	}
	return filepath.Join(pathConfig.VSCodeConfigDir, "settings.json"), nil
}
