package internal

import "time"

// ToolType 工具类型.
type ToolType string

const (
	ToolTypeCodex  ToolType = "codex"
	ToolTypeClaude ToolType = "claude"
)

// MirrorConfig 镜像源配置结构.
type MirrorConfig struct {
	Name         string    `json:"name" toml:"name"`                                       // 镜像源名称
	BaseURL      string    `json:"base_url" toml:"base_url"`                               // API基础URL.
	APIKey       string    `json:"api_key" toml:"api_key"`                                 // API密钥.
	EnvKey       string    `json:"env_key" toml:"env_key"`                                 // 环境变量key
	ToolType     ToolType  `json:"tool_type" toml:"tool_type"`                             // 工具类型
	ModelName    string    `json:"model_name,omitempty" toml:"model_name,omitempty"`       // 模型名称 (可选，主要用于Claude)
	CreatedAt    time.Time `json:"created_at,omitempty" toml:"created_at,omitempty"`       // 创建时间
	LastModified time.Time `json:"last_modified,omitempty" toml:"last_modified,omitempty"` // 最后修改时间
	Deleted      bool      `json:"deleted,omitempty" toml:"deleted,omitempty"`             // 删除标记
	DeletedAt    time.Time `json:"deleted_at,omitempty" toml:"deleted_at,omitempty"`       // 删除时间
	// Claude Code 额外环境变量配置
	ExtraEnv map[string]string `json:"extra_env,omitempty" toml:"extra_env,omitempty"` // 额外环境变量 (如 ANTHROPIC_DEFAULT_HAIKU_MODEL 等)
}

// SystemConfig 系统配置结构.
type SystemConfig struct {
	CurrentMirror string         `json:"current_mirror" toml:"current_mirror"` // 当前使用的镜像源（兼容旧版本）
	CurrentCodex  string         `json:"current_codex" toml:"current_codex"`   // 当前使用的 Codex 镜像源
	CurrentClaude string         `json:"current_claude" toml:"current_claude"` // 当前使用的 Claude 镜像源
	Mirrors       []MirrorConfig `json:"mirrors" toml:"mirrors"`               // 可用镜像源列表
	Sync          *SyncConfig    `json:"sync,omitempty" toml:"sync,omitempty"` // 云同步配置
}

// CodexConfig Codex CLI配置文件结构.
type CodexConfig struct {
	ModelProvider          string                         `toml:"model_provider,omitempty"`
	Model                  string                         `toml:"model,omitempty"`
	ModelReasoningEffort   string                         `toml:"model_reasoning_effort,omitempty"`
	DisableResponseStorage bool                           `toml:"disable_response_storage,omitempty"`
	ModelProviders         map[string]ModelProviderConfig `toml:"model_providers,omitempty"`
	// 保留其他未知字段.
	OtherFields map[string]interface{} `toml:"-"`
}

// ModelProviderConfig 模型提供商配置.
type ModelProviderConfig struct {
	Name               string `toml:"name"`
	BaseURL            string `toml:"base_url"`
	WireAPI            string `toml:"wire_api,omitempty"`
	EnvKey             string `toml:"env_key,omitempty"`
	RequiresOpenAIAuth bool   `toml:"requires_openai_auth,omitempty"`
}

// CodexAuth Codex CLI认证文件结构.
type CodexAuth struct {
	APIKey string `json:"OPENAI_API_KEY"` // API密钥.
}

// VSCodeSettings VS Code设置文件结构.
type VSCodeSettings struct {
	ChatGPTAPIBase string                 `json:"chatgpt.apiBase,omitempty"`
	ChatGPTConfig  map[string]interface{} `json:"chatgpt.config,omitempty"`
	// 保留其他设置.
	OtherSettings map[string]interface{} `json:"-"`
}

// ConfigFile 配置文件操作接口.
type ConfigFile interface {
	Read() error
	Write() error
	GetPath() string
}

// Platform 平台类型.
type Platform string

const (
	PlatformWindows Platform = "windows"
	PlatformMac     Platform = "mac"
	PlatformLinux   Platform = "linux"

	// 平台字符串常量.
	WindowsOS = "windows"
	MacOS     = "darwin"
	LinuxOS   = "linux"

	// 环境变量常量.
	CodexSwitchAPIKeyEnv  = "CODEX_SWITCH_OPENAI_API_KEY"
	AnthropicBaseURLEnv   = "ANTHROPIC_BASE_URL"
	AnthropicAuthTokenEnv = "ANTHROPIC_AUTH_TOKEN"
	AnthropicModelEnv     = "ANTHROPIC_MODEL"

	// 默认镜像源名称.
	DefaultMirrorName = "official"

	// Shell 类型常量.
	BashShell       = "bash"
	ZshShell        = "zsh"
	FishShell       = "fish"
	PowerShellShell = "powershell"
	PwshShell       = "pwsh"
	CmdShell        = "cmd"
	BatShell        = "bat"
)

// PathConfig 路径配置结构.
type PathConfig struct {
	CodexConfigDir  string // Codex配置目录.
	VSCodeConfigDir string // VS Code配置目录.
	HomeDir         string // 用户主目录
}

// SyncConfig 云同步配置结构.
type SyncConfig struct {
	Enabled       bool      `json:"enabled" toml:"enabled"`                                   // 是否启用同步
	Provider      string    `json:"provider" toml:"provider"`                                 // 同步提供商 (gist, webdav, custom)
	Endpoint      string    `json:"endpoint" toml:"endpoint"`                                 // API端点
	Token         string    `json:"token" toml:"token"`                                       // 访问令牌
	EncryptKey    string    `json:"encrypt_key" toml:"encrypt_key"`                           // 加密密钥
	AutoSync      bool      `json:"auto_sync" toml:"auto_sync"`                               // 自动同步
	SyncInterval  int       `json:"sync_interval" toml:"sync_interval"`                       // 同步间隔(分钟)
	LastSync      time.Time `json:"last_sync" toml:"last_sync"`                               // 最后同步时间
	DeviceID      string    `json:"device_id" toml:"device_id"`                               // 设备ID
	GistID        string    `json:"gist_id,omitempty" toml:"gist_id,omitempty"`               // GitHub Gist ID
	SyncAPIKeys   bool      `json:"sync_api_keys" toml:"sync_api_keys"`                       // 是否同步API密钥
	EncryptionPwd string    `json:"encryption_pwd,omitempty" toml:"encryption_pwd,omitempty"` // 加密密码（可选，用于额外安全层）
}

// SyncData 同步数据结构.
type SyncData struct {
	Mirrors           []MirrorConfig `json:"mirrors"`                   // 镜像源配置（可能包含加密的API密钥）
	CurrentCodex      string         `json:"current_codex"`             // 当前 Codex 镜像源
	CurrentClaude     string         `json:"current_claude"`            // 当前 Claude 镜像源
	Timestamp         time.Time      `json:"timestamp"`                 // 时间戳
	DeviceID          string         `json:"device_id"`                 // 设备ID
	Version           string         `json:"version"`                   // 配置版本
	Checksum          string         `json:"checksum,omitempty"`        // 数据校验和
	HasAPIKeys        bool           `json:"has_api_keys"`              // 是否包含API密钥
	DeletedMirrors    []MirrorConfig `json:"deleted_mirrors,omitempty"` // 已删除的镜像源（用于追踪删除操作）
	ValidatedChecksum bool           `json:"-"`                         // 本地校验标记（不参与序列化）
}

// SecureMirrorConfig 安全的镜像源配置（不包含API密钥）.
type SecureMirrorConfig struct {
	Name      string   `json:"name"`                 // 镜像源名称
	BaseURL   string   `json:"base_url"`             // API基础URL
	HasAPIKey bool     `json:"has_api_key"`          // 是否有API密钥
	ToolType  ToolType `json:"tool_type"`            // 工具类型
	ModelName string   `json:"model_name,omitempty"` // 模型名称
}

// SyncProvider 同步提供商接口.
type SyncProvider interface {
	// Upload 上传数据到云端
	Upload(data []byte, filename string) error
	// Download 从云端下载数据
	Download(filename string) ([]byte, error)
	// List 列出云端文件
	List() ([]string, error)
	// Delete 删除云端文件
	Delete(filename string) error
	// GetInfo 获取提供商信息
	GetInfo() ProviderInfo
}

// ProviderInfo 提供商信息.
type ProviderInfo struct {
	Name        string `json:"name"`          // 提供商名称
	Type        string `json:"type"`          // 提供商类型
	Endpoint    string `json:"endpoint"`      // API端点
	MaxFileSize int64  `json:"max_file_size"` // 最大文件大小
	Description string `json:"description"`   // 描述
}

// FieldConflict 字段级冲突信息.
type FieldConflict struct {
	FieldName    string    // 冲突的字段名 (BaseURL, ModelName, ToolType, APIKey)
	LocalValue   string    // 本地值
	RemoteValue  string    // 远程值
	LocalTime    time.Time // 本地修改时间
	RemoteTime   time.Time // 远程修改时间
	RemoteDevice string    // 远程修改设备ID
}

// MirrorConflict 镜像源级冲突信息.
type MirrorConflict struct {
	MirrorName     string          // 镜像源名称
	FieldConflicts []FieldConflict // 该镜像源的所有字段冲突
	LocalMirror    *MirrorConfig   // 本地镜像源配置
	RemoteMirror   *MirrorConfig   // 远程镜像源配置
}

// FieldResolution 字段解决结果.
type FieldResolution struct {
	FieldName     string // 字段名
	ResolvedValue string // 解决后的值
	Choice        string // 用户选择: "local", "remote", "manual"
}
