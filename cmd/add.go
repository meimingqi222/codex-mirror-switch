package cmd

import (
	"fmt"
	"os"
	"strings"

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
  --extra-env  额外环境变量 (可选，格式: KEY=VALUE，可多次使用)

示例：
  codex-mirror add myapi https://api.example.com sk-1234567890
  codex-mirror add myclaude https://api.anthropic.com sk-ant-123 --type claude
  codex-mirror add custom https://api.custom.com sk-key --type claude --model claude-3-5-sonnet-20241022
  codex-mirror add proxy https://proxy.example.com sk-key --type claude \
    --extra-env ANTHROPIC_DEFAULT_HAIKU_MODEL=gemini-2.5-flash-lite \
    --extra-env ANTHROPIC_DEFAULT_SONNET_MODEL=gemini-claude-sonnet-4-5-thinking \
    --extra-env ANTHROPIC_DEFAULT_OPUS_MODEL=gemini-claude-opus-4-5-thinking
  codex-mirror add local http://localhost:8080`,
	Args: cobra.RangeArgs(2, 3),
	RunE: runAddCommand,
}

// runAddCommand 执行add命令的实际逻辑.
func runAddCommand(cmd *cobra.Command, args []string) error {
	name := args[0]
	baseURL := args[1]
	apiKey := ""
	if len(args) > 2 {
		apiKey = args[2]
	}

	// 验证 URL 格式
	if err := internal.ValidateBaseURL(baseURL); err != nil {
		return fmt.Errorf("无效的 API 地址: %v", err)
	}

	// 获取工具类型
	toolType, _ := cmd.Flags().GetString("type")
	if toolType == "" {
		toolType = string(internal.ToolTypeCodex) // 默认为 codex
	}

	// 获取模型名称
	modelName, _ := cmd.Flags().GetString("model")

	// 获取额外环境变量
	extraEnvSlice, _ := cmd.Flags().GetStringArray("extra-env")
	extraEnv := parseExtraEnv(extraEnvSlice)

	// 验证工具类型
	var internalToolType internal.ToolType
	switch toolType {
	case string(internal.ToolTypeCodex):
		internalToolType = internal.ToolTypeCodex
	case string(internal.ToolTypeClaude):
		internalToolType = internal.ToolTypeClaude
	default:
		return fmt.Errorf("无效的工具类型 '%s'，支持: %s, %s", toolType, internal.ToolTypeCodex, internal.ToolTypeClaude)
	}

	// 创建镜像源管理器
	mm, err := internal.NewMirrorManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		return fmt.Errorf("%v", err)
	}

	// 添加镜像源
	if err := mm.AddMirrorWithExtra(name, baseURL, apiKey, internalToolType, modelName, extraEnv); err != nil {
		fmt.Fprintf(os.Stderr, "添加镜像源失败: %v\n", err)
		return fmt.Errorf("添加镜像源失败: %v", err)
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
	if len(extraEnv) > 0 {
		fmt.Println("  额外环境变量:")
		for key, value := range extraEnv {
			fmt.Printf("    %s=%s\n", key, value)
		}
	}

	return nil
}

// parseExtraEnv 解析额外环境变量参数.
func parseExtraEnv(envSlice []string) map[string]string {
	result := make(map[string]string)
	for _, env := range envSlice {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func init() {
	addCmd.Flags().StringP("type", "t", "codex", "工具类型 (codex|claude)")
	addCmd.Flags().StringP("model", "m", "", "模型名称 (可选，主Claude使用)")
	addCmd.Flags().StringArrayP("extra-env", "e", []string{}, "额外环境变量 (格式: KEY=VALUE，可多次使用)")
	rootCmd.AddCommand(addCmd)
}
