package cmd

import (
	"fmt"
	"os"

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
  --model  模型名称 (可选，主Claude使用，如 claude-3-5-sonnet-20241022)

示例：
  codex-mirror add myapi https://api.example.com sk-1234567890
  codex-mirror add myclaude https://api.anthropic.com sk-ant-123 --type claude
  codex-mirror add custom https://api.custom.com sk-key --type claude --model claude-3-5-sonnet-20241022
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

		// 获取模型名称
		modelName, _ := cmd.Flags().GetString("model")

		// 验证工具类型
		var internalToolType internal.ToolType
		switch toolType {
		case "codex":
			internalToolType = internal.ToolTypeCodex
		case "claude":
			internalToolType = internal.ToolTypeClaude
		default:
			fmt.Fprintf(os.Stderr, "错误: 无效的工具类型 '%s'，支持: codex, claude\n", toolType)
			os.Exit(1)
			return
		}

		// 创建镜像源管理器
		mm, err := internal.NewMirrorManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
			return
		}

		// 添加镜像源
		if err := mm.AddMirrorWithModel(name, baseURL, apiKey, internalToolType, modelName); err != nil {
			fmt.Fprintf(os.Stderr, "添加镜像源失败: %v\n", err)
			os.Exit(1)
			return
		}

		fmt.Printf("成功添加镜像源 '%s'\n", name)
		fmt.Printf("  名称: %s\n", name)
		fmt.Printf("  类型: %s\n", toolType)
		fmt.Printf("  URL: %s\n", baseURL)
		if apiKey != "" {
			fmt.Printf("  API密钥: %s\n", maskAPIKey(apiKey))
		}
		if modelName != "" {
			fmt.Printf("  模型: %s\n", modelName)
		}
	},
}

func init() {
	addCmd.Flags().StringP("type", "t", "codex", "工具类型 (codex|claude)")
	addCmd.Flags().StringP("model", "m", "", "模型名称 (可选，主Claude使用)")
	rootCmd.AddCommand(addCmd)
}
