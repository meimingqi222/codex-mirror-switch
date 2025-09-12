package cmd

import (
	"fmt"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

var (
	// 命令行标志.
	codexOnly  bool
	vscodeOnly bool
	noBackup   bool
)

// switchCmd 代表switch命令.
var switchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "切换到指定的镜像源",
	Long: `切换到指定的镜像源，并根据配置类型自动处理。

Claude 配置：只设置环境变量 (ANTHROPIC_BASE_URL, ANTHROPIC_AUTH_TOKEN)
Codex 配置：修改配置文件并设置环境变量

参数：
  name  要切换到的镜像源名称

示例：
  codex-mirror switch myclaude
  codex-mirror switch mycodex
  codex-mirror switch mycodex --no-backup`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mirrorName := args[0]

		// 创建镜像源管理器
		mm, err := internal.NewMirrorManager()
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			return
		}

		// 修复mirrors.toml中的env_key格式
		if err := mm.FixEnvKeyFormat(); err != nil {
			fmt.Printf("修复env_key格式失败: %v\n", err)
			return
		}

		// 获取目标镜像源配置
		mirror, err := mm.GetMirrorByName(mirrorName)
		if err != nil {
			fmt.Printf("获取镜像源配置失败: %v\n", err)
			return
		}

		fmt.Printf("正在切换到镜像源 '%s' (%s)...\n", mirrorName, mirror.ToolType)

		// 根据工具类型应用配置
		switch mirror.ToolType {
		case internal.ToolTypeClaude:
			if err := applyClaudeConfig(mirror); err != nil {
				fmt.Printf("应用Claude配置失败: %v\n", err)
				return
			}
		case internal.ToolTypeCodex:
			if err := applyCodexConfig(mirror); err != nil {
				fmt.Printf("应用Codex配置失败: %v\n", err)
				return
			}
		default:
			fmt.Printf("错误: 不支持的配置类型 '%s'\n", mirror.ToolType)
			return
		}

		// 切换镜像源状态
		if err := mm.SwitchMirror(mirrorName); err != nil {
			fmt.Printf("切换镜像源状态失败: %v\n", err)
			return
		}

		fmt.Printf("\n成功切换到镜像源 '%s'\n", mirrorName)
		fmt.Printf("  类型: %s\n", mirror.ToolType)
		fmt.Printf("  URL: %s\n", mirror.BaseURL)
		if mirror.APIKey != "" {
			fmt.Printf("  API密钥: %s\n", maskAPIKey(mirror.APIKey))
		}
	},
}

// applyClaudeConfig 应用Claude配置（只设置环境变量）.
func applyClaudeConfig(mirror *internal.MirrorConfig) error {
	envManager := internal.NewEnvManager()

	// 设置 Claude 环境变量（包括可选的模型名称）
	if err := envManager.SetClaudeEnvVarsWithModel(mirror.BaseURL, mirror.APIKey, mirror.ModelName); err != nil {
		return err
	}

	// 显示设置的环境变量
	fmt.Println("✓ Claude Code环境变量已设置")
	if mirror.ModelName != "" {
		fmt.Printf("  模型: %s\n", mirror.ModelName)
	}
	return nil
}

// applyCodexConfig 应用Codex配置（修改配置文件并设置环境变量）.
func applyCodexConfig(mirror *internal.MirrorConfig) error {
	// 更新Codex CLI配置
	if err := updateCodexConfig(mirror); err != nil {
		return err
	}
	fmt.Println("✓ Codex CLI配置已更新")

	// 更新VS Code配置
	if !vscodeOnly {
		if err := updateVSCodeConfig(mirror); err != nil {
			return err
		}
		fmt.Println("✓ VS Code配置已更新")
	}

	return nil
}

// updateCodexConfig 更新Codex配置.
func updateCodexConfig(mirror *internal.MirrorConfig) error {
	ccm, err := internal.NewCodexConfigManager()
	if err != nil {
		return err
	}

	// 备份现有配置
	if !noBackup {
		if err := ccm.BackupConfig(); err != nil {
			fmt.Printf("警告: 备份Codex配置失败: %v\n", err)
		}
	}

	// 应用新配置
	return ccm.ApplyMirror(mirror)
}

// updateVSCodeConfig 更新VS Code配置.
func updateVSCodeConfig(mirror *internal.MirrorConfig) error {
	vcm, err := internal.NewVSCodeConfigManager()
	if err != nil {
		return err
	}

	// 备份现有配置
	if !noBackup {
		if err := vcm.BackupSettings(); err != nil {
			fmt.Printf("警告: 备份VS Code配置失败: %v\n", err)
		}
	}

	// 应用新配置
	return vcm.ApplyMirror(mirror)
}

func init() {
	rootCmd.AddCommand(switchCmd)

	// 添加命令行标志.
	switchCmd.Flags().BoolVar(&codexOnly, "codex-only", false, "只更新Codex CLI配置")
	switchCmd.Flags().BoolVar(&vscodeOnly, "vscode-only", false, "只更新VS Code配置")
	switchCmd.Flags().BoolVar(&noBackup, "no-backup", false, "不备份现有配置")
}
