package cmd

import (
	"fmt"
	"strings"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

var (
	// 命令行标志.
	codexOnly  bool
	vscodeOnly bool
	noBackup   bool
	shellFmt   string
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
  codex-mirror switch mycodex --no-backup

即时刷新当前终端环境变量：
  eval "$(codex-mirror switch myclaude --shell bash)"
  # zsh 同上；fish: codex-mirror switch myclaude --shell fish | source
  # PowerShell: codex-mirror switch myclaude --shell powershell | iex
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mirrorName := args[0]

		// 创建镜像源管理器
		mm, err := internal.NewMirrorManager()
		if err != nil {
			return fmt.Errorf("错误: %w", err)
		}

		// 先检查镜像源是否存在，避免对不存在的镜像进行修复
		_, err = mm.GetMirrorByName(mirrorName)
		if err != nil {
			return fmt.Errorf("获取镜像源配置失败: %w", err)
		}

		// 修复mirrors.toml中的env_key格式
		if err := mm.FixEnvKeyFormat(); err != nil {
			return fmt.Errorf("修复env_key格式失败: %w", err)
		}

		// 重新获取目标镜像源配置（修复后可能已更新）
		mirror, err := mm.GetMirrorByName(mirrorName)
		if err != nil {
			return fmt.Errorf("获取镜像源配置失败: %w", err)
		}

		// 如果是shell输出模式，只收集环境变量并输出shell导出语句
		if shellFmt != "" {
			envToEmit := map[string]string{}

			switch mirror.ToolType {
			case internal.ToolTypeClaude:
				envToEmit[internal.AnthropicBaseURLEnv] = mirror.BaseURL
				envToEmit[internal.AnthropicAuthTokenEnv] = mirror.APIKey
				if strings.TrimSpace(mirror.ModelName) != "" {
					envToEmit[internal.AnthropicModelEnv] = mirror.ModelName
				} else {
					// 如果目标镜像没有模型名称，明确清除 ANTHROPIC_MODEL
					envToEmit[internal.AnthropicModelEnv] = ""
				}
			case internal.ToolTypeCodex:
				// Codex 使用镜像EnvKey来读取API KEY
				envKey := mirror.EnvKey
				if strings.TrimSpace(envKey) == "" {
					envKey = internal.CodexSwitchAPIKeyEnv
				}
				envToEmit[envKey] = mirror.APIKey
			default:
				return fmt.Errorf("错误: 不支持的配置类型 '%s'", mirror.ToolType)
			}

			// 输出shell导出语句并退出
			emitShellExports(envToEmit, shellFmt)
			return nil
		}

		// 非shell模式：正常执行配置应用和状态切换
		fmt.Printf("正在切换到镜像源 '%s' (%s)...\n", mirrorName, mirror.ToolType)

		// 根据工具类型应用配置
		switch mirror.ToolType {
		case internal.ToolTypeClaude:
			if err := applyClaudeConfig(mirror); err != nil {
				return fmt.Errorf("应用Claude配置失败: %w", err)
			}
		case internal.ToolTypeCodex:
			if err := applyCodexConfig(mirror); err != nil {
				return fmt.Errorf("应用Codex配置失败: %w", err)
			}
		default:
			return fmt.Errorf("错误: 不支持的配置类型 '%s'", mirror.ToolType)
		}

		// 切换镜像源状态
		if err := mm.SwitchMirror(mirrorName); err != nil {
			return fmt.Errorf("切换镜像源状态失败: %w", err)
		}

		fmt.Printf("\n成功切换到镜像源 '%s'\n", mirrorName)
		fmt.Printf("  类型: %s\n", mirror.ToolType)
		fmt.Printf("  URL: %s\n", mirror.BaseURL)
		if mirror.APIKey != "" {
			fmt.Printf("  API密钥: %s\n", maskAPIKey(mirror.APIKey))
		}
		return nil
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
	fmt.Println("[OK] Claude Code环境变量已设置")
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
	fmt.Println("[OK] Codex CLI配置已更新")

	// 更新VS Code配置
	if !vscodeOnly {
		if err := updateVSCodeConfig(mirror); err != nil {
			return err
		}
		fmt.Println("[OK] VS Code配置已更新")
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
	switchCmd.Flags().StringVar(&shellFmt, "shell", "", "输出适配当前shell的导出语句(bash|zsh|fish|powershell|cmd)")
}

// emitShellExports 将环境变量以指定shell格式输出到stdout。
// 注意：仅输出导出语句；所有人类可读日志应走stderr。
func emitShellExports(vars map[string]string, shell string) {
	if len(vars) == 0 {
		return
	}

	exportFunc := getShellExportFunc(strings.ToLower(shell))
	for k, v := range vars {
		exportFunc(k, v)
	}
}

func getShellExportFunc(shell string) func(key, value string) {
	switch shell {
	case internal.BashShell, internal.ZshShell, "sh":
		return func(k, v string) {
			if v == "" {
				fmt.Printf("unset %s\n", k)
			} else {
				fmt.Printf("export %s='%s'\n", k, shSingleQuote(v))
			}
		}
	case internal.FishShell:
		return func(k, v string) {
			if v == "" {
				fmt.Printf("set -e %s\n", k)
			} else {
				fmt.Printf("set -gx %s %s\n", k, fishEscape(v))
			}
		}
	case internal.PowerShellShell, internal.PwshShell:
		return func(k, v string) {
			if v == "" {
				fmt.Printf("Remove-Item Env:%s -ErrorAction SilentlyContinue\n", k)
			} else {
				fmt.Printf("$Env:%s = \"%s\"\n", k, psDoubleQuote(v))
			}
		}
	case "cmd", "bat":
		return func(k, v string) {
			if v == "" {
				fmt.Printf("set %s=\n", k)
			} else {
				fmt.Printf("set %s=%s\n", k, v)
			}
		}
	default:
		return func(k, v string) {
			if v == "" {
				fmt.Printf("unset %s\n", k)
			} else {
				fmt.Printf("export %s='%s'\n", k, shSingleQuote(v))
			}
		}
	}
}

// shSingleQuote 对单引号进行安全转义：' -> '\".
func shSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

// fishEscape 对fish传参做最小必要转义（包裹为单个arg）.
func fishEscape(s string) string {
	// 如果包含空格或特殊字符，包裹引号，并转义内部引号
	if strings.ContainsAny(s, " \t\n\"'\\$`){}[]()<>|&;*") {
		return "'" + strings.ReplaceAll(s, "'", "\\'") + "'"
	}
	return s
}

// psDoubleQuote 对 PowerShell 双引号进行转义.
func psDoubleQuote(s string) string {
	// 在 PowerShell 中，双引号可通过 `\"` 或 ``"`` 方式转义；使用反引号更直观
	s = strings.ReplaceAll(s, "`", "``")
	s = strings.ReplaceAll(s, "\"", "`\"")
	return s
}
