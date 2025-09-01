package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd 代表基础命令，当不带任何子命令调用时执行.
var rootCmd = &cobra.Command{
	Use:   "codex-mirror",
	Short: "Codex镜像切换工具",
	Long: `Codex镜像切换工具是一个用于管理和切换Codex CLI和VS Code插件镜像源的命令行工具。

支持功能：
- 添加、删除、列出镜像源
- 切换镜像源
- 自动更新Codex CLI和VS Code配置
- 跨平台支持（Windows、macOS、Linux）

使用示例：
  codex-mirror add myapi https://api.example.com sk-1234567890
  codex-mirror list
  codex-mirror switch myapi
  codex-mirror status`,
}

// Execute 添加所有子命令到根命令并设置标志.
// 这由main.main()调用。只需要对rootCmd执行一次.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// maskAPIKey 遮蔽API密钥，只显示前4位和后4位.
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

func init() {
	// 在这里可以定义标志和配置设置.
	// Cobra支持持久标志，如果在这里定义，将对所有子命令全局可用.
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.codex-mirror.yaml)")

	// Cobra也支持本地标志，只对特定命令运行.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
