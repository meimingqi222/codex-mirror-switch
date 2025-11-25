package cmd

import (
	"fmt"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

// update 命令的标志.
var (
	updateURL   string
	updateKey   string
	updateModel string
	updateType  string
)

// updateCmd 代表 update 命令.
var updateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "更新镜像源配置",
	Long: `更新指定镜像源的配置信息。

可更新的字段：
  --url    API 基础 URL
  --key    API 密钥
  --model  模型名称
  --type   工具类型 (codex|claude)

注意：
- 至少需要指定一个要更新的字段
- 不能更新官方镜像源

示例：
  codex-mirror update myapi --url https://new-api.example.com
  codex-mirror update myapi --key sk-new-key
  codex-mirror update myapi --url https://api.example.com --key sk-key
  codex-mirror update myclaude --model claude-3-5-sonnet-20241022`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdateCommand,
}

// runUpdateCommand 执行 update 命令的实际逻辑.
func runUpdateCommand(cmd *cobra.Command, args []string) error {
	name := args[0]

	// 检查是否有任何更新
	if updateURL == "" && updateKey == "" && updateModel == "" && updateType == "" {
		return fmt.Errorf("请至少指定一个要更新的字段 (--url, --key, --model, --type)")
	}

	// 验证 URL 格式
	if updateURL != "" {
		if err := internal.ValidateBaseURL(updateURL); err != nil {
			return fmt.Errorf("无效的 API 地址: %v", err)
		}
	}

	// 验证工具类型
	if updateType != "" && updateType != "codex" && updateType != "claude" {
		return fmt.Errorf("无效的工具类型 '%s'，支持: codex, claude", updateType)
	}

	// 创建镜像源管理器
	mm, err := internal.NewMirrorManager()
	if err != nil {
		return fmt.Errorf("错误: %w", err)
	}

	// 检查镜像源是否存在
	mirror, err := mm.GetMirrorByName(name)
	if err != nil {
		return fmt.Errorf("镜像源 '%s' 不存在", name)
	}

	// 不能更新官方镜像源
	if name == internal.DefaultMirrorName {
		return fmt.Errorf("不能更新官方镜像源")
	}

	// 更新镜像源
	if err := mm.UpdateMirrorFull(name, updateURL, updateKey, updateModel, updateType); err != nil {
		return fmt.Errorf("更新镜像源失败: %w", err)
	}

	fmt.Printf("成功更新镜像源 '%s'\n", name)

	// 显示更新后的信息
	updatedMirror, _ := mm.GetMirrorByName(name)
	if updatedMirror != nil {
		fmt.Printf("  类型: %s\n", updatedMirror.ToolType)
		fmt.Printf("  URL: %s\n", updatedMirror.BaseURL)
		if updatedMirror.APIKey != "" {
			fmt.Printf("  API密钥: %s\n", maskAPIKey(updatedMirror.APIKey))
		}
		if updatedMirror.ModelName != "" {
			fmt.Printf("  模型: %s\n", updatedMirror.ModelName)
		}
	}

	// 提示是否需要重新应用
	config := mm.GetConfig()
	if config.CurrentCodex == name || config.CurrentClaude == name {
		fmt.Printf("\n💡 提示: '%s' 是当前激活的配置，运行以下命令应用更改:\n", name)
		fmt.Printf("   codex-mirror switch %s\n", name)
	}

	_ = mirror // 避免未使用警告
	return nil
}

func init() {
	updateCmd.Flags().StringVar(&updateURL, "url", "", "API 基础 URL")
	updateCmd.Flags().StringVar(&updateKey, "key", "", "API 密钥")
	updateCmd.Flags().StringVar(&updateModel, "model", "", "模型名称")
	updateCmd.Flags().StringVar(&updateType, "type", "", "工具类型 (codex|claude)")
	rootCmd.AddCommand(updateCmd)
}
