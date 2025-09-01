package internal

import (
	"fmt"
	"os"
	"path/filepath"

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
		// 如果配置文件不存在，初始化默认配置
		mm.initDefaultConfig()
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
		CurrentMirror: "official",
		Mirrors: []MirrorConfig{
			{
				Name:    "official",
				BaseURL: "https://api.openai.com",
				APIKey:  "",
			},
		},
	}
}

// AddMirror 添加镜像源.
func (mm *MirrorManager) AddMirror(name, baseURL, apiKey string) error {
	// 检查镜像源是否已存在
	for _, mirror := range mm.config.Mirrors {
		if mirror.Name == name {
			return fmt.Errorf("镜像源 '%s' 已存在", name)
		}
	}

	// 添加新镜像源
	newMirror := MirrorConfig{
		Name:    name,
		BaseURL: baseURL,
		APIKey:  apiKey,
	}

	mm.config.Mirrors = append(mm.config.Mirrors, newMirror)
	return mm.saveConfig()
}

// RemoveMirror 删除镜像源.
func (mm *MirrorManager) RemoveMirror(name string) error {
	if name == "official" {
		return fmt.Errorf("不能删除官方镜像源")
	}

	for i, mirror := range mm.config.Mirrors {
		if mirror.Name == name {
			// 如果删除的是当前使用的镜像源，切换到官方镜像源
			if mm.config.CurrentMirror == name {
				mm.config.CurrentMirror = "official"
			}

			// 删除镜像源
			mm.config.Mirrors = append(mm.config.Mirrors[:i], mm.config.Mirrors[i+1:]...)
			return mm.saveConfig()
		}
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

// SwitchMirror 切换镜像源.
func (mm *MirrorManager) SwitchMirror(name string) error {
	// 检查镜像源是否存在
	for _, mirror := range mm.config.Mirrors {
		if mirror.Name == name {
			mm.config.CurrentMirror = name
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
