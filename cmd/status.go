package cmd

import (
	"fmt"
	"os"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

// statusCmd 代表status命令.
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "显示当前配置状态",
	Long: `显示当前镜像源配置状态，包括：
- Claude Code配置状态
- Codex CLI配置状态
- VS Code配置状态

示例：
  codex-mirror status`,
	Run: func(cmd *cobra.Command, args []string) {
		// 创建镜像源管理器
		mm, err := internal.NewMirrorManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
			return
		}

		fmt.Println("当前配置状态:")
		fmt.Println("==================================================")

		// 检查Claude Code配置状态
		fmt.Println("Claude Code配置:")
		checkClaudeStatus(mm)
		fmt.Println()

		// 检查Codex CLI配置状态
		fmt.Println("Codex CLI配置:")
		checkCodexStatus(mm)
		fmt.Println()

		// 检查VS Code配置状态
		fmt.Println("VS Code配置:")
		checkVSCodeStatus(mm)
	},
}

// checkClaudeStatus 检查Claude配置状态.
func checkClaudeStatus(mm *internal.MirrorManager) {
	// 获取当前激活的Claude配置
	currentClaude, err := mm.GetCurrentClaudeMirror()
	if err != nil {
		fmt.Printf("  ❌ 未设置Claude配置: %v\n", err)
		return
	}

	fmt.Printf("  当前配置: %s\n", currentClaude.Name)
	fmt.Printf("  API端点: %s\n", currentClaude.BaseURL)

	// 检查环境变量
	baseURL := ""
	authToken := ""

	if envURL := os.Getenv("ANTHROPIC_BASE_URL"); envURL != "" {
		baseURL = envURL
	}
	if envToken := os.Getenv("ANTHROPIC_AUTH_TOKEN"); envToken != "" {
		authToken = envToken
	}

	// 比较环境变量
	urlMatch := baseURL == currentClaude.BaseURL
	tokenMatch := authToken == currentClaude.APIKey

	fmt.Printf("  环境变量 ANTHROPIC_BASE_URL: ")
	switch {
	case baseURL == "":
		fmt.Printf("❌ 未设置\n")
	case urlMatch:
		fmt.Printf("[OK] 正确\n")
	default:
		fmt.Printf("⚠️  不匹配 (当前: %s, 期望: %s)\n", baseURL, currentClaude.BaseURL)
	}

	fmt.Printf("  环境变量 ANTHROPIC_AUTH_TOKEN: ")
	switch {
	case authToken == "":
		fmt.Printf("❌ 未设置\n")
	case tokenMatch:
		fmt.Printf("[OK] 正确\n")
	default:
		fmt.Printf("⚠️  不匹配\n")
	}
}

// checkCodexStatus 检查Codex配置状态.
func checkCodexStatus(mm *internal.MirrorManager) {
	// 获取当前激活的Codex配置
	currentCodex, err := mm.GetCurrentCodexMirror()
	if err != nil {
		fmt.Printf("  ❌ 未设置Codex配置: %v\n", err)
		return
	}

	fmt.Printf("  当前配置: %s\n", currentCodex.Name)
	fmt.Printf("  API端点: %s\n", currentCodex.BaseURL)

	ccm, err := internal.NewCodexConfigManager()
	if err != nil {
		fmt.Printf("  ❌ 无法访问Codex配置: %v\n", err)
		return
	}

	// 检查配置文件
	config, err := ccm.GetCurrentConfig()
	if err != nil {
		fmt.Printf("  ❌ 配置文件不存在或无法读取: %v\n", err)
		return
	}

	// 检查认证文件
	auth, err := ccm.GetCurrentAuth()
	if err != nil {
		fmt.Printf("  ❌ 认证文件不存在或无法读取: %v\n", err)
		return
	}

	// 获取当前镜像源的base_url
	currentBaseURL := ""
	if config.ModelProviders != nil {
		if provider, exists := config.ModelProviders[currentCodex.Name]; exists {
			currentBaseURL = provider.BaseURL
		}
	}

	// 比较配置
	configMatch := currentBaseURL == currentCodex.BaseURL
	authMatch := auth.APIKey == currentCodex.APIKey

	// 检查环境变量
	envVarName := internal.CodexSwitchAPIKeyEnv // Codex 固定使用专用的环境变量名
	envKey := os.Getenv(envVarName)
	envMatch := envKey == currentCodex.APIKey

	fmt.Printf("  配置文件 (~/.codex/config.toml): ")
	if configMatch {
		fmt.Printf("[OK] 正确\n")
	} else {
		fmt.Printf("⚠️  不匹配 (当前: %s)\n", currentBaseURL)
	}

	fmt.Printf("  认证文件 (~/.codex/auth.json): ")
	if authMatch {
		fmt.Printf("[OK] 正确\n")
	} else {
		fmt.Printf("⚠️  不匹配\n")
	}

	fmt.Printf("  环境变量 %s: ", envVarName)
	switch {
	case envKey == "":
		fmt.Printf("❌ 未设置\n")
	case envMatch:
		fmt.Printf("[OK] 正确\n")
	default:
		fmt.Printf("⚠️  不匹配\n")
	}
}

// checkVSCodeStatus 检查VS Code配置状态.
func checkVSCodeStatus(mm *internal.MirrorManager) {
	// 获取当前激活的Codex配置（VS Code通常与Codex配置相同）
	currentCodex, err := mm.GetCurrentCodexMirror()
	if err != nil {
		fmt.Printf("  ❌ 未设置Codex配置，无法检查VS Code配置: %v\n", err)
		return
	}

	vcm, err := internal.NewVSCodeConfigManager()
	if err != nil {
		fmt.Printf("  ❌ 无法访问VS Code配置: %v\n", err)
		return
	}

	// 获取当前配置
	config, err := vcm.GetCurrentConfig()
	if err != nil {
		fmt.Printf("  ❌ 配置文件不存在或无法读取: %v\n", err)
		return
	}

	// 检查配置
	apiBaseMatch := false

	if apiBase, exists := config["apiBase"]; exists {
		if apiBaseStr, ok := apiBase.(string); ok {
			apiBaseMatch = apiBaseStr == currentCodex.BaseURL
		}
	}

	switch {
	case apiBaseMatch:
		fmt.Printf("  [OK] 配置正确 (chatgpt.apiBase: %s)\n", currentCodex.BaseURL)
	case len(config) == 0:
		fmt.Println("  ⚠️  未配置ChatGPT插件")
	default:
		fmt.Println("  ⚠️  配置不匹配")
		if !apiBaseMatch {
			fmt.Printf("    chatgpt.apiBase不匹配\n")
		}
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
