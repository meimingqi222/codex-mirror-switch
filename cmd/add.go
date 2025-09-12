package cmd

import (
	"fmt"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

// addCmd 代表add命令.
var addCmd = &cobra.Command{
	Use:   "add [name] [base-url] [api-key]",
	Short: "添加新的镜像源",
	Long: `添加一个新的镜像源配置。

参数：
  name     镜像源名称（必需）
  base-url API基础URL（必需）
  api-key  API密钥（可选）

标志：
  --type   工具类型 (codex|claude, 默认: codex)

示例：
  codex-mirror add myapi https://api.example.com sk-1234567890
  codex-mirror add myclaude https://api.anthropic.com sk-ant-123 --type claude
  codex-mirror add local http://localhost:8080`,
	Args: cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		baseURL := args[1]
		apiKey := ""
		if len(args) > 2 {
			apiKey = args[2]
		}

		// 获取工具类型
		toolType, _ := cmd.Flags().GetString("type")
		if toolType == "" {
			toolType = "codex" // 默认为 codex
		}

		// 验证工具类型
		var internalToolType internal.ToolType
		switch toolType {
		case "codex":
			internalToolType = internal.ToolTypeCodex
		case "claude":
			internalToolType = internal.ToolTypeClaude
		default:
			fmt.Printf("错误: 无效的工具类型 '%s'，支持: codex, claude\n", toolType)
			return
		}

		// 创建镜像源管理器
		mm, err := internal.NewMirrorManager()
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			return
		}

		// 添加镜像源
		if err := mm.AddMirrorWithType(name, baseURL, apiKey, internalToolType); err != nil {
			fmt.Printf("添加镜像源失败: %v\n", err)
			return
		}

		fmt.Printf("成功添加镜像源 '%s'\n", name)
		fmt.Printf("  名称: %s\n", name)
		fmt.Printf("  类型: %s\n", toolType)
		fmt.Printf("  URL: %s\n", baseURL)
		if apiKey != "" {
			fmt.Printf("  API密钥: %s\n", maskAPIKey(apiKey))
		}
	},
}

func init() {
	addCmd.Flags().StringP("type", "t", "codex", "工具类型 (codex|claude)")
	rootCmd.AddCommand(addCmd)
}
