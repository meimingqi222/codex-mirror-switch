package cmd

import (
    "fmt"
    "os"
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
    Run: func(cmd *cobra.Command, args []string) {
        mirrorName := args[0]

        // 创建镜像源管理器
        mm, err := internal.NewMirrorManager()
        if err != nil {
            fmt.Fprintf(os.Stderr, "错误: %v\n", err)
            return
        }

		// 修复mirrors.toml中的env_key格式
        if err := mm.FixEnvKeyFormat(); err != nil {
            fmt.Fprintf(os.Stderr, "修复env_key格式失败: %v\n", err)
            return
        }

		// 获取目标镜像源配置
        mirror, err := mm.GetMirrorByName(mirrorName)
        if err != nil {
            fmt.Fprintf(os.Stderr, "获取镜像源配置失败: %v\n", err)
            return
        }

        // 在shell输出模式下，把提示打印到stderr，避免污染stdout中的可执行导出语句
        logf := func(format string, a ...any) {
            if shellFmt != "" {
                fmt.Fprintf(os.Stderr, format, a...)
            } else {
                fmt.Printf(format, a...)
            }
        }

        logf("正在切换到镜像源 '%s' (%s)...\n", mirrorName, mirror.ToolType)

        // 若需要在stdout中输出导出语句，收集需要设置的环境变量
        envToEmit := map[string]string{}

        // 根据工具类型应用配置
        switch mirror.ToolType {
        case internal.ToolTypeClaude:
            if shellFmt != "" {
                // 仅收集需要导出的变量，不修改配置文件、不提示刷新
                envToEmit[internal.AnthropicBaseURLEnv] = mirror.BaseURL
                envToEmit[internal.AnthropicAuthTokenEnv] = mirror.APIKey
                if strings.TrimSpace(mirror.ModelName) != "" {
                    envToEmit[internal.AnthropicModelEnv] = mirror.ModelName
                }
            } else {
                if err := applyClaudeConfig(mirror); err != nil {
                    fmt.Fprintf(os.Stderr, "应用Claude配置失败: %v\n", err)
                    return
                }
            }
        case internal.ToolTypeCodex:
            if err := applyCodexConfig(mirror); err != nil {
                fmt.Fprintf(os.Stderr, "应用Codex配置失败: %v\n", err)
                return
            }
            if shellFmt != "" {
                // Codex 使用镜像EnvKey来读取API KEY
                envKey := mirror.EnvKey
                if strings.TrimSpace(envKey) == "" {
                    envKey = internal.CodexSwitchAPIKeyEnv
                }
                envToEmit[envKey] = mirror.APIKey
            }
        default:
            fmt.Fprintf(os.Stderr, "错误: 不支持的配置类型 '%s'\n", mirror.ToolType)
            return
        }

		// 切换镜像源状态
        if err := mm.SwitchMirror(mirrorName); err != nil {
            fmt.Fprintf(os.Stderr, "切换镜像源状态失败: %v\n", err)
            return
        }

        logf("\n成功切换到镜像源 '%s'\n", mirrorName)
        logf("  类型: %s\n", mirror.ToolType)
        logf("  URL: %s\n", mirror.BaseURL)
        if mirror.APIKey != "" {
            logf("  API密钥: %s\n", maskAPIKey(mirror.APIKey))
        }

        // 如果需要，输出当前shell可直接eval/source的导出语句到stdout
        if shellFmt != "" {
            emitShellExports(envToEmit, shellFmt)
            if mirror.ToolType == internal.ToolTypeClaude {
                logf("\n提示: 使用 --shell 仅让当前会话即时生效，未持久化写入配置文件。若需持久化，请不带 --shell 重新执行。\n")
            }
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
    switchCmd.Flags().StringVar(&shellFmt, "shell", "", "输出适配当前shell的导出语句(bash|zsh|fish|powershell|cmd)")
}

// emitShellExports 将环境变量以指定shell格式输出到stdout。
// 注意：仅输出导出语句；所有人类可读日志应走stderr。
func emitShellExports(vars map[string]string, shell string) {
    if len(vars) == 0 {
        return
    }

    switch strings.ToLower(shell) {
    case "bash", "zsh", "sh":
        for k, v := range vars {
            fmt.Printf("export %s='%s'\n", k, shSingleQuote(v))
        }
    case "fish":
        for k, v := range vars {
            // fish 推荐使用全局导出变量
            fmt.Printf("set -gx %s %s\n", k, fishEscape(v))
        }
    case "powershell", "pwsh":
        for k, v := range vars {
            fmt.Printf("$Env:%s = \"%s\"\n", k, psDoubleQuote(v))
        }
    case "cmd", "bat":
        for k, v := range vars {
            // 注意：cmd.exe 的 set 只影响当前会话
            fmt.Printf("set %s=%s\n", k, v)
        }
    default:
        // 默认按 POSIX shell 处理
        for k, v := range vars {
            fmt.Printf("export %s='%s'\n", k, shSingleQuote(v))
        }
    }
}

// shSingleQuote 对单引号进行安全转义：' -> '\''
func shSingleQuote(s string) string {
    return strings.ReplaceAll(s, "'", "'\\''")
}

// fishEscape 对fish传参做最小必要转义（包裹为单个arg）
func fishEscape(s string) string {
    // 如果包含空格或特殊字符，包裹引号，并转义内部引号
    if strings.ContainsAny(s, " \t\n\"'\\$`){}[]()<>|&;*") {
        return "'" + strings.ReplaceAll(s, "'", "\\'") + "'"
    }
    return s
}

// psDoubleQuote 对 PowerShell 双引号进行转义
func psDoubleQuote(s string) string {
    // 在 PowerShell 中，双引号可通过 `\"` 或 ``"`` 方式转义；使用反引号更直观
    s = strings.ReplaceAll(s, "`", "``")
    s = strings.ReplaceAll(s, "\"", "`\"")
    return s
}
