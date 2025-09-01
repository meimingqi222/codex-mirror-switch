package internal

// MirrorConfig 镜像源配置结构.
type MirrorConfig struct {
	Name    string `json:"name" toml:"name"`         // 镜像源名称
	BaseURL string `json:"base_url" toml:"base_url"` // API基础URL
	APIKey  string `json:"api_key" toml:"api_key"`   // API密钥
}

// SystemConfig 系统配置结构.
type SystemConfig struct {
	CurrentMirror string         `json:"current_mirror" toml:"current_mirror"` // 当前使用的镜像源
	Mirrors       []MirrorConfig `json:"mirrors" toml:"mirrors"`               // 可用镜像源列表
}

// CodexConfig Codex CLI配置文件结构.
type CodexConfig struct {
	ModelProvider        string                         `toml:"model_provider,omitempty"`
	Model                string                         `toml:"model,omitempty"`
	ModelReasoningEffort string                         `toml:"model_reasoning_effort,omitempty"`
	ModelProviders       map[string]ModelProviderConfig `toml:"model_providers,omitempty"`
	// 保留其他未知字段
	OtherFields map[string]interface{} `toml:"-"`
}

// ModelProviderConfig 模型提供商配置.
type ModelProviderConfig struct {
	Name    string `toml:"name"`
	BaseURL string `toml:"base_url"`
	WireAPI string `toml:"wire_api,omitempty"`
	EnvKey  string `toml:"env_key,omitempty"`
}

// CodexAuth Codex CLI认证文件结构.
type CodexAuth struct {
	APIKey string `json:"api_key"` // API密钥
}

// VSCodeSettings VS Code设置文件结构.
type VSCodeSettings struct {
	ChatGPTAPIBase string                 `json:"chatgpt.apiBase,omitempty"`
	ChatGPTConfig  map[string]interface{} `json:"chatgpt.config,omitempty"`
	// 保留其他设置
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
)

// PathConfig 路径配置结构.
type PathConfig struct {
	CodexConfigDir  string // Codex配置目录
	VSCodeConfigDir string // VS Code配置目录
	HomeDir         string // 用户主目录
}
