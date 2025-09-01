package cmd

import (
	"fmt"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

// statusCmd 代表status命令.
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "显示当前配置状态",
	Long: `显示当前镜像源配置状态，包括：
- 当前使用的镜像源
- Codex CLI配置状态
- VS Code配置状态

示例：
  codex-mirror status`,
	Run: func(cmd *cobra.Command, args []string) {
		// 创建镜像源管理器
		mm, err := internal.NewMirrorManager()
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			return
		}

		// 获取当前镜像源
		currentMirror, err := mm.GetCurrentMirror()
		if err != nil {
			fmt.Printf("获取当前镜像源失败: %v\n", err)
			return
		}

		fmt.Println("当前配置状态:")
		fmt.Println("==================================================")

		// 显示当前镜像源信息
		fmt.Printf("当前镜像源: %s\n", currentMirror.Name)
		fmt.Printf("  URL: %s\n", currentMirror.BaseURL)
		if currentMirror.APIKey != "" {
			fmt.Printf("  API密钥: %s\n", maskAPIKey(currentMirror.APIKey))
		} else {
			fmt.Printf("  API密钥: 未设置\n")
		}
		fmt.Println()

		// 检查Codex CLI配置状态
		fmt.Println("Codex CLI配置:")
		checkCodexStatus(currentMirror)
		fmt.Println()

		// 检查VS Code配置状态
		fmt.Println("VS Code配置:")
		checkVSCodeStatus(currentMirror)
	},
}

// checkCodexStatus 检查Codex配置状态.
func checkCodexStatus(expectedMirror *internal.MirrorConfig) {
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
		if provider, exists := config.ModelProviders[expectedMirror.Name]; exists {
			currentBaseURL = provider.BaseURL
		}
	}

	// 比较配置
	configMatch := currentBaseURL == expectedMirror.BaseURL
	authMatch := auth.APIKey == expectedMirror.APIKey

	if configMatch && authMatch {
		fmt.Println("  ✓ 配置正确")
	} else {
		fmt.Println("  ⚠️  配置不匹配")
		if !configMatch {
			fmt.Printf("    配置文件URL: %s (期望: %s)\n", currentBaseURL, expectedMirror.BaseURL)
		}
		if !authMatch {
			fmt.Printf("    认证文件API密钥不匹配\n")
		}
	}
}

// checkVSCodeStatus 检查VS Code配置状态.
func checkVSCodeStatus(expectedMirror *internal.MirrorConfig) {
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
	configMatch := false

	if apiBase, exists := config["apiBase"]; exists {
		if apiBaseStr, ok := apiBase.(string); ok {
			apiBaseMatch = apiBaseStr == expectedMirror.BaseURL
		}
	}

	if chatgptConfig, exists := config["config"]; exists {
		if configMap, ok := chatgptConfig.(map[string]interface{}); ok {
			if apiBaseUrl, exists := configMap["apiBaseUrl"]; exists {
				if apiBaseUrlStr, ok := apiBaseUrl.(string); ok {
					configMatch = apiBaseUrlStr == expectedMirror.BaseURL
				}
			}
		}
	}

	switch {
	case apiBaseMatch && configMatch:
		fmt.Println("  ✓ 配置正确")
	case len(config) == 0:
		fmt.Println("  ⚠️  未配置ChatGPT插件")
	default:
		fmt.Println("  ⚠️  配置不匹配")
		if !apiBaseMatch {
			fmt.Printf("    chatgpt.apiBase不匹配\n")
		}
		if !configMatch {
			fmt.Printf("    chatgpt.config.apiBaseUrl不匹配\n")
		}
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
