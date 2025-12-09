package cmd

import (
	"fmt"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

var (
	cleanAll    bool // 清理所有 ANTHROPIC_ 开头的环境变量
	cleanConfig bool // 清理配置文件中的环境变量
)

// cleanCmd 代表clean命令.
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "清理系统环境变量或配置文件中的 ANTHROPIC 相关配置",
	Long: `清理系统环境变量或 ~/.claude/settings.json 中的 ANTHROPIC 相关配置。

默认清理系统环境变量（通过 setx/shell profile 设置的）。

标志：
  --all     清理所有 ANTHROPIC_ 开头的环境变量（默认只清理基础配置）
  --config  清理配置文件中的环境变量（而非系统环境变量）

示例：
  codex-mirror clean              # 清理系统中的基础 ANTHROPIC 环境变量
  codex-mirror clean --all        # 清理系统中所有 ANTHROPIC_ 环境变量
  codex-mirror clean --config     # 清理配置文件中的基础 ANTHROPIC 环境变量
  codex-mirror clean --config --all  # 清理配置文件中所有 ANTHROPIC_ 环境变量
`,
	RunE: runCleanCommand,
}

func runCleanCommand(cmd *cobra.Command, args []string) error {
	if cleanConfig {
		return cleanConfigFile()
	}
	return cleanSystemEnv()
}

// cleanSystemEnv 清理系统环境变量.
func cleanSystemEnv() error {
	envManager := internal.NewEnvManager()

	// 基础环境变量列表
	baseEnvVars := []string{
		internal.AnthropicBaseURLEnv,
		internal.AnthropicAuthTokenEnv,
		internal.AnthropicModelEnv,
	}

	// 额外的常见环境变量
	extraEnvVars := []string{
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"ANTHROPIC_API_KEY",
		"ANTHROPIC_SMALL_FAST_MODEL",
	}

	var toClean []string
	if cleanAll {
		toClean = make([]string, 0, len(baseEnvVars)+len(extraEnvVars))
		toClean = append(toClean, baseEnvVars...)
		toClean = append(toClean, extraEnvVars...)
	} else {
		toClean = baseEnvVars
	}

	fmt.Println("正在清理系统环境变量...")
	cleaned := 0
	for _, envVar := range toClean {
		if err := envManager.UnsetEnvVar(envVar); err != nil {
			fmt.Printf("  警告: 清理 %s 失败: %v\n", envVar, err)
		} else {
			cleaned++
		}
	}

	if cleaned > 0 {
		fmt.Printf("\n已清理 %d 个环境变量\n", cleaned)
		fmt.Println("\n注意: 请重启终端或重新登录以使更改生效")
	} else {
		fmt.Println("没有需要清理的环境变量")
	}

	return nil
}

// cleanConfigFile 清理配置文件中的环境变量.
func cleanConfigFile() error {
	ccm, err := internal.NewClaudeConfigManager()
	if err != nil {
		return fmt.Errorf("创建配置管理器失败: %w", err)
	}

	settings, err := ccm.LoadSettings()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	if len(settings.Env) == 0 {
		fmt.Println("配置文件中没有环境变量")
		return nil
	}

	// 基础环境变量列表
	baseEnvVars := map[string]bool{
		internal.AnthropicBaseURLEnv:   true,
		internal.AnthropicAuthTokenEnv: true,
		internal.AnthropicModelEnv:     true,
	}

	var cleaned []string
	for key := range settings.Env {
		shouldClean := false

		if cleanAll {
			// 清理所有 ANTHROPIC_ 开头的变量
			if len(key) > 10 && key[:10] == "ANTHROPIC_" {
				shouldClean = true
			}
		} else {
			// 默认：只清理基础变量
			if baseEnvVars[key] {
				shouldClean = true
			}
		}

		if shouldClean {
			cleaned = append(cleaned, key)
			delete(settings.Env, key)
		}
	}

	if len(cleaned) == 0 {
		fmt.Println("没有需要清理的环境变量")
		return nil
	}

	if err := ccm.SaveSettings(settings); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	fmt.Printf("已清理配置文件中 %d 个环境变量:\n", len(cleaned))
	for _, key := range cleaned {
		fmt.Printf("  - %s\n", key)
	}
	fmt.Printf("\n配置文件: %s\n", ccm.GetSettingsPath())

	return nil
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "清理所有 ANTHROPIC_ 开头的环境变量")
	cleanCmd.Flags().BoolVar(&cleanConfig, "config", false, "清理配置文件中的环境变量（而非系统环境变量）")
	rootCmd.AddCommand(cleanCmd)
}
