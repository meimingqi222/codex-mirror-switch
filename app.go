package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"codex-mirror/internal"
)

// App GUI 应用结构体，封装 internal 包调用
type App struct {
	mirrorManager *internal.MirrorManager
	configPath    string
}

// MirrorDTO 镜像源数据传输对象
type MirrorDTO struct {
	Name         string            `json:"name"`
	BaseURL      string            `json:"base_url"`
	APIKey       string            `json:"api_key"`       // 前端显示掩码
	HasAPIKey    bool              `json:"has_api_key"`   // 是否有 API Key
	EnvKey       string            `json:"env_key"`       // 环境变量key
	ToolType     string            `json:"tool_type"`     // "codex" 或 "claude"
	ModelName    string            `json:"model_name"`
	ExtraEnv     map[string]string `json:"extra_env"`
	IsCurrent    bool              `json:"is_current"`     // 是否当前激活
	CreatedAt    string            `json:"created_at"`
	LastModified string            `json:"last_modified"`
}

// StatusDTO 状态数据传输对象
type StatusDTO struct {
	CurrentCodex  string        `json:"current_codex"`
	CurrentClaude string        `json:"current_claude"`
	CodexStatus   ConfigStatus  `json:"codex_status"`
	ClaudeStatus  ConfigStatus  `json:"claude_status"`
	VSCodeStatus  ConfigStatus  `json:"vscode_status"`
	ConfigPath    string        `json:"config_path"`
}

// ConfigStatus 配置状态
type ConfigStatus struct {
	Exists bool   `json:"exists"`
	Path   string `json:"path"`
	Error  string `json:"error,omitempty"`
}

// NewApp 创建 App 实例
func NewApp() (*App, error) {
	mm, err := internal.NewMirrorManager()
	if err != nil {
		return nil, fmt.Errorf("创建镜像源管理器失败: %w", err)
	}
	return &App{
		mirrorManager: mm,
		configPath:    mm.GetConfigPath(),
	}, nil
}

// ListMirrors 获取所有镜像源
func (a *App) ListMirrors() []MirrorDTO {
	mirrors := a.mirrorManager.ListActiveMirrors()
	config := a.mirrorManager.GetConfig()

	result := make([]MirrorDTO, 0, len(mirrors))
	for _, m := range mirrors {
		result = append(result, a.toMirrorDTO(m, config))
	}
	return result
}

// GetMirror 获取指定镜像源
func (a *App) GetMirror(name string) (MirrorDTO, error) {
	mirror, err := a.mirrorManager.GetMirrorByName(name)
	if err != nil {
		return MirrorDTO{}, err
	}
	return a.toMirrorDTO(*mirror, a.mirrorManager.GetConfig()), nil
}

// AddMirror 添加镜像源
func (a *App) AddMirror(mirror MirrorDTO) error {
	toolType := internal.ToolType(mirror.ToolType)
	if toolType == "" {
		toolType = internal.ToolTypeCodex // 默认为 codex 类型
	}

	return a.mirrorManager.AddMirrorWithExtra(
		mirror.Name,
		mirror.BaseURL,
		mirror.APIKey,
		toolType,
		mirror.ModelName,
		mirror.ExtraEnv,
	)
}

// UpdateMirror 更新镜像源
func (a *App) UpdateMirror(mirror MirrorDTO) error {
	// 先检查镜像源是否存在
	_, err := a.mirrorManager.GetMirrorByName(mirror.Name)
	if err != nil {
		return err
	}

	// 更新镜像源
	err = a.mirrorManager.UpdateMirrorFull(
		mirror.Name,
		mirror.BaseURL,
		mirror.APIKey,
		mirror.ModelName,
		mirror.ToolType,
	)
	if err != nil {
		return err
	}

	// 如果有额外环境变量，需要特殊处理
	if len(mirror.ExtraEnv) > 0 {
		// 获取原始镜像源配置
		mirrorConfig, err := a.mirrorManager.GetMirrorByName(mirror.Name)
		if err != nil {
			return err
		}
		mirrorConfig.ExtraEnv = mirror.ExtraEnv
		return a.mirrorManager.SaveConfig()
	}

	return nil
}

// RemoveMirror 删除镜像源
func (a *App) RemoveMirror(name string) error {
	return a.mirrorManager.RemoveMirror(name)
}

// SwitchMirror 切换镜像源
func (a *App) SwitchMirror(name string) error {
	// 获取镜像源配置
	mirror, err := a.mirrorManager.GetMirrorByName(name)
	if err != nil {
		return err
	}

	// 切换镜像源
	if err := a.mirrorManager.SwitchMirror(name); err != nil {
		return err
	}

	// 应用配置到 Codex
	if mirror.ToolType == internal.ToolTypeCodex {
		if err := a.applyCodexConfig(name); err != nil {
			return fmt.Errorf("应用 Codex 配置失败: %w", err)
		}
	}

	// 应用配置到 Claude
	if mirror.ToolType == internal.ToolTypeClaude {
		if err := a.applyClaudeConfig(name); err != nil {
			return fmt.Errorf("应用 Claude 配置失败: %w", err)
		}
	}

	return nil
}

// GetCurrentStatus 获取当前状态
func (a *App) GetCurrentStatus() StatusDTO {
	config := a.mirrorManager.GetConfig()

	status := StatusDTO{
		CurrentCodex:  config.CurrentCodex,
		CurrentClaude: config.CurrentClaude,
		ConfigPath:    a.configPath,
	}

	// 检查 Codex 配置状态
	if codexPath, err := internal.GetCodexConfigPath(); err == nil {
		status.CodexStatus = ConfigStatus{
			Path: codexPath,
		}
		if _, err := internal.GetPathConfig(); err == nil {
			status.CodexStatus.Exists = true
		}
	}

	// 检查 Claude 配置状态
	status.ClaudeStatus = ConfigStatus{
		Exists: true, // Claude 使用环境变量，总是返回 true
		Path:   "环境变量",
	}

	// 检查 VS Code 配置状态
	if vscodePath, err := internal.GetVSCodeSettingsPath(); err == nil {
		status.VSCodeStatus = ConfigStatus{
			Path: vscodePath,
		}
	}

	return status
}

// ValidateURL 验证 URL 格式
func (a *App) ValidateURL(url string) error {
	return internal.ValidateBaseURL(url)
}

// toMirrorDTO 将 MirrorConfig 转换为 MirrorDTO
func (a *App) toMirrorDTO(m internal.MirrorConfig, config *internal.SystemConfig) MirrorDTO {
	dto := MirrorDTO{
		Name:         m.Name,
		BaseURL:      m.BaseURL,
		HasAPIKey:    m.APIKey != "",
		EnvKey:       m.EnvKey,
		ToolType:     string(m.ToolType),
		ModelName:    m.ModelName,
		ExtraEnv:     m.ExtraEnv,
		CreatedAt:    m.CreatedAt.Format("2006-01-02 15:04:05"),
		LastModified: m.LastModified.Format("2006-01-02 15:04:05"),
	}

	// API Key 掩码处理
	if m.APIKey != "" {
		dto.APIKey = a.maskAPIKey(m.APIKey)
	}

	// 判断是否为当前激活的镜像源
	if m.ToolType == internal.ToolTypeCodex && m.Name == config.CurrentCodex {
		dto.IsCurrent = true
	} else if m.ToolType == internal.ToolTypeClaude && m.Name == config.CurrentClaude {
		dto.IsCurrent = true
	}

	return dto
}

// maskAPIKey 掩码 API Key
func (a *App) maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}
	// 显示前4个字符和后4个字符，中间用*代替
	return apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
}

// applyCodexConfig 应用配置到 Codex
func (a *App) applyCodexConfig(mirrorName string) error {
	mirror, err := a.mirrorManager.GetMirrorByName(mirrorName)
	if err != nil {
		return err
	}

	// 使用 CodexConfigManager 应用配置
	ccm, err := internal.NewCodexConfigManager()
	if err != nil {
		return err
	}

	return ccm.ApplyMirror(mirror)
}

// applyClaudeConfig 应用配置到 Claude
func (a *App) applyClaudeConfig(mirrorName string) error {
	mirror, err := a.mirrorManager.GetMirrorByName(mirrorName)
	if err != nil {
		return err
	}

	// 使用 ClaudeConfigManager 应用配置
	ccm, err := internal.NewClaudeConfigManager()
	if err != nil {
		return err
	}

	return ccm.ApplyMirror(mirror)
}

// GetConfigPath 获取配置文件路径
func (a *App) GetConfigPath() string {
	return a.configPath
}

// ExportConfig 导出配置（用于备份）
func (a *App) ExportConfig() (string, error) {
	config := a.mirrorManager.GetConfig()
	// 这里可以返回 JSON 格式的配置
	return fmt.Sprintf("%+v", config), nil
}

// Startup 应用启动时的回调
func (a *App) Startup(ctx context.Context) {
	// 初始化操作
}

// DomReady DOM 加载完成时的回调
func (a *App) DomReady(ctx context.Context) {
	// DOM 已准备好
}

// BeforeClose 应用关闭前的回调
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	return false
}

// Shutdown 应用关闭时的回调
func (a *App) Shutdown(ctx context.Context) {
	// 清理操作
}

// ============ 云同步相关方法 ============

// SyncStatusDTO 同步状态数据传输对象
type SyncStatusDTO struct {
	Enabled      bool   `json:"enabled"`
	Provider     string `json:"provider"`
	Endpoint     string `json:"endpoint"`
	DeviceID     string `json:"device_id"`
	GistID       string `json:"gist_id"`
	AutoSync     bool   `json:"auto_sync"`
	SyncInterval int    `json:"sync_interval"`
	LastSync     string `json:"last_sync"`
	Message      string `json:"message"`
}

// SyncInitRequest 初始化同步请求
type SyncInitRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
	GistID   string `json:"gist_id,omitempty"`
}

// SyncUpdateRequest 更新同步设置请求
type SyncUpdateRequest struct {
	NewPassword string `json:"new_password,omitempty"`
	NewGistID   string `json:"new_gist_id,omitempty"`
}

// SyncInitResult 初始化同步结果
type SyncInitResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// GetSyncStatus 获取同步状态
func (a *App) GetSyncStatus() SyncStatusDTO {
	config := a.mirrorManager.GetConfig()

	if config.Sync == nil {
		return SyncStatusDTO{
			Enabled: false,
			Message: "未配置云同步",
		}
	}

	result := SyncStatusDTO{
		Enabled:      config.Sync.Enabled,
		Provider:     config.Sync.Provider,
		Endpoint:     config.Sync.Endpoint,
		DeviceID:     config.Sync.DeviceID,
		GistID:       config.Sync.GistID,
		AutoSync:     config.Sync.AutoSync,
		SyncInterval: config.Sync.SyncInterval,
	}

	if config.Sync.LastSync.IsZero() {
		result.Message = "尚未进行过同步"
	} else {
		result.LastSync = config.Sync.LastSync.Format("2006-01-02 15:04:05")
		result.Message = "上次同步: " + formatDuration(time.Since(config.Sync.LastSync))
	}

	return result
}

// InitSync 初始化云同步
func (a *App) InitSync(req SyncInitRequest) SyncInitResult {
	// 验证参数
	if req.Token == "" {
		return SyncInitResult{
			Success: false,
			Message: "GitHub Token 不能为空",
		}
	}

	if req.Password == "" {
		return SyncInitResult{
			Success: false,
			Message: "加密密码不能为空",
		}
	}

	if len(req.Password) < 8 {
		return SyncInitResult{
			Success: false,
			Message: "加密密码长度至少8位",
		}
	}

	syncManager := internal.NewSyncManager(a.mirrorManager)

	if err := syncManager.InitSyncWithPasswordAndGist("gist", "https://api.github.com", req.Token, req.Password, req.GistID); err != nil {
		return SyncInitResult{
			Success: false,
			Message: "初始化失败: " + err.Error(),
		}
	}

	return SyncInitResult{
		Success: true,
		Message: "云同步初始化成功",
	}
}

// SyncPush 推送配置到云端
func (a *App) SyncPush() (string, error) {
	syncManager := internal.NewSyncManager(a.mirrorManager)

	if err := syncManager.Push(); err != nil {
		return "", err
	}

	return "配置已推送到云端", nil
}

// SyncPull 从云端拉取配置
func (a *App) SyncPull() (string, error) {
	syncManager := internal.NewSyncManager(a.mirrorManager)

	if err := syncManager.Pull(); err != nil {
		return "", err
	}

	return "配置已从云端拉取", nil
}

// DisableSync 禁用云同步
func (a *App) DisableSync() error {
	config := a.mirrorManager.GetConfig()
	if config.Sync == nil {
		return fmt.Errorf("云同步未配置")
	}

	config.Sync.Enabled = false
	return a.mirrorManager.SaveConfig()
}

// EnableSync 启用云同步
func (a *App) EnableSync() error {
	config := a.mirrorManager.GetConfig()
	if config.Sync == nil {
		return fmt.Errorf("云同步未初始化")
	}

	config.Sync.Enabled = true
	return a.mirrorManager.SaveConfig()
}

// UpdateSyncSettings 更新同步设置（密码或 Gist ID）
func (a *App) UpdateSyncSettings(req SyncUpdateRequest) SyncInitResult {
	config := a.mirrorManager.GetConfig()
	if config.Sync == nil {
		return SyncInitResult{
			Success: false,
			Message: "云同步未初始化",
		}
	}

	// 更新密码
	if req.NewPassword != "" {
		if len(req.NewPassword) < 8 {
			return SyncInitResult{
				Success: false,
				Message: "密码长度至少8位",
			}
		}
		config.Sync.EncryptionPwd = req.NewPassword
	}

	// 更新 Gist ID
	if req.NewGistID != "" {
		config.Sync.GistID = req.NewGistID
	}

	// 保存配置
	if err := a.mirrorManager.SaveConfig(); err != nil {
		return SyncInitResult{
			Success: false,
			Message: "保存配置失败: " + err.Error(),
		}
	}

	return SyncInitResult{
		Success: true,
		Message: "同步设置已更新",
	}
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%.0f秒前", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%.0f分钟前", d.Minutes())
	case d < 24*time.Hour:
		return fmt.Sprintf("%.1f小时前", d.Hours())
	default:
		return fmt.Sprintf("%.1f天前", d.Hours()/24)
	}
}
