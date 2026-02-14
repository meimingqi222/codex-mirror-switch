package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
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
	useEnvVar  bool // 使用环境变量方式设置 Claude 配置（默认使用配置文件）
	dryRun     bool // 预览模式，不实际修改配置
)

// switchCmd 代表switch命令.
var switchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "切换到指定的镜像源",
	Long: `切换到指定的镜像源，并根据配置类型自动处理。

Claude 配置：
  - 默认：修改 ~/.claude/settings.json 配置文件中的 env 字段
  - --env：使用系统环境变量方式 (ANTHROPIC_BASE_URL, ANTHROPIC_AUTH_TOKEN)
Codex 配置：修改配置文件并设置环境变量

参数：
  name  要切换到的镜像源名称（省略时进入交互式选择）

示例：
  codex-mirror switch myclaude              # 使用配置文件方式
  codex-mirror switch myclaude --env        # 使用环境变量方式
  codex-mirror switch mycodex
  codex-mirror switch mycodex --no-backup
  codex-mirror switch mycodex --dry-run     # 预览切换效果，不实际修改

即时刷新当前终端环境变量：
  eval "$(codex-mirror switch myclaude --shell bash)"
  # zsh 同上；fish: codex-mirror switch myclaude --shell fish | source
  # PowerShell: codex-mirror switch myclaude --shell powershell | iex
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var mirrorName string

		// 如果没有提供参数，进入交互式选择
		if len(args) == 0 {
			if dryRun {
				return fmt.Errorf("--dry-run 需要指定镜像源名称")
			}
			selected, err := interactiveSelectMirror()
			if err != nil {
				return fmt.Errorf("交互式选择失败: %w", err)
			}
			mirrorName = selected
		} else {
			mirrorName = args[0]
		}

		// 创建镜像源管理器
		mm, err := internal.NewMirrorManager()
		if err != nil {
			return fmt.Errorf("错误: %w", err)
		}

		// 先检查镜像源是否存在
		mirror, err := mm.GetMirrorByName(mirrorName)
		if err != nil {
			return fmt.Errorf("获取镜像源配置失败: %w", err)
		}

		// 预览模式
		if dryRun {
			return showDryRunPreview(mm, mirror)
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
			// 获取当前 Claude 镜像的 ExtraEnv 用于清理
			var oldExtraEnv map[string]string
			if currentClaude := mm.GetConfig().CurrentClaude; currentClaude != "" {
				if oldMirror, err := mm.GetMirrorByName(currentClaude); err == nil {
					oldExtraEnv = oldMirror.ExtraEnv
				}
			}
			if err := applyClaudeConfig(mirror, oldExtraEnv); err != nil {
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

// applyClaudeConfig 应用Claude配置（默认使用配置文件，--env 时使用环境变量）.
func applyClaudeConfig(mirror *internal.MirrorConfig, oldExtraEnv map[string]string) error {
	if useEnvVar {
		// 使用环境变量方式
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

	// 默认：使用配置文件方式
	ccm, err := internal.NewClaudeConfigManager()
	if err != nil {
		return err
	}

	// 备份现有配置
	if !noBackup {
		if err := ccm.BackupSettings(); err != nil {
			fmt.Printf("警告: 备份Claude配置失败: %v\n", err)
		}
	}

	// 应用新配置（同时清理旧镜像的额外环境变量）
	if err := ccm.ApplyMirrorWithCleanup(mirror, oldExtraEnv); err != nil {
		return err
	}

	fmt.Println("[OK] Claude Code配置文件已更新")
	fmt.Printf("  配置文件: %s\n", ccm.GetSettingsPath())
	if mirror.ModelName != "" {
		fmt.Printf("  模型: %s\n", mirror.ModelName)
	}
	return nil
}

// applyCodexConfig 应用Codex配置（修改配置文件并设置环境变量）.
func applyCodexConfig(mirror *internal.MirrorConfig) error {
	// 检查标志互斥
	if codexOnly && vscodeOnly {
		return fmt.Errorf("--codex-only 和 --vscode-only 不能同时使用")
	}

	// 并行更新Codex CLI和VS Code配置
	pt := internal.NewParallelTask()

	if !vscodeOnly {
		pt.Add(func() error {
			err := updateCodexConfig(mirror)
			if err == nil {
				fmt.Println("[OK] Codex CLI配置已更新")
			}
			return err
		})
	}

	if !codexOnly {
		pt.Add(func() error {
			err := updateVSCodeConfig(mirror)
			if err == nil {
				fmt.Println("[OK] VS Code配置已更新")
			}
			return err
		})
	}

	// 等待所有任务完成
	errs := pt.Wait()

	// 收集错误信息
	var allErrs []error
	for _, err := range errs {
		if err != nil {
			allErrs = append(allErrs, err)
		}
	}

	return internal.CombinedError(allErrs)
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

// showDryRunPreview 预览切换效果（不实际修改配置）.
func showDryRunPreview(_ *internal.MirrorManager, mirror *internal.MirrorConfig) error {
	fmt.Printf("[DRY-RUN] 预览切换到 '%s' (%s)\n\n", mirror.Name, mirror.ToolType)

	switch mirror.ToolType {
	case internal.ToolTypeClaude:
		fmt.Println("将修改 Claude Code 配置文件:")
		ccm, _ := internal.NewClaudeConfigManager()
		fmt.Printf("  配置文件: %s\n", ccm.GetSettingsPath())
		fmt.Printf("  %s = %s\n", internal.AnthropicBaseURLEnv, mirror.BaseURL)
		if mirror.APIKey != "" {
			fmt.Printf("  %s = %s\n", internal.AnthropicAuthTokenEnv, internal.MaskAPIKey(mirror.APIKey))
		}
		if mirror.ModelName != "" {
			fmt.Printf("  %s = %s\n", internal.AnthropicModelEnv, mirror.ModelName)
		}

	case internal.ToolTypeCodex:
		fmt.Println("将修改以下配置:")

		if !vscodeOnly {
			fmt.Println("  Codex CLI:")
			ccm, _ := internal.NewCodexConfigManager()
			fmt.Printf("    配置文件: %s\n", ccm.GetConfigPath())
			fmt.Printf("    %s = %s\n", mirror.EnvKey, internal.MaskAPIKey(mirror.APIKey))
		}

		if !codexOnly {
			fmt.Println("  VS Code:")
			vcm, _ := internal.NewVSCodeConfigManager()
			fmt.Printf("    配置文件: %s\n", vcm.GetSettingsPath())
			fmt.Printf("    chatgpt.apiBase = %s\n", mirror.BaseURL)
		}
	}

	fmt.Println("\n将更新的系统配置:")
	fmt.Printf("  当前镜像源 -> %s\n", mirror.Name)

	return nil
}

// interactiveSelectMirror 交互式选择镜像源.
func interactiveSelectMirror() (string, error) {
	mm, err := internal.NewMirrorManager()
	if err != nil {
		return "", err
	}

	mirrors := mm.ListActiveMirrors()
	if len(mirrors) == 0 {
		return "", fmt.Errorf("没有可用的镜像源，请先使用 'codex-mirror add' 添加")
	}

	fmt.Println("可用镜像源:")
	fmt.Println("==================================================")

	// 按工具类型分组显示
	codexMirrors := make([]internal.MirrorConfig, 0)
	claudeMirrors := make([]internal.MirrorConfig, 0)

	for i := range mirrors {
		switch mirrors[i].ToolType {
		case internal.ToolTypeCodex:
			codexMirrors = append(codexMirrors, mirrors[i])
		case internal.ToolTypeClaude:
			claudeMirrors = append(claudeMirrors, mirrors[i])
		}
	}

	// 显示 Codex 镜像源
	if len(codexMirrors) > 0 {
		fmt.Println("【Codex 类型】")
		for i := range codexMirrors {
			m := codexMirrors[i]
			indicator := "  "
			if mm.GetConfig().CurrentCodex == m.Name {
				indicator = "* "
			}
			keyDisplay := internal.MaskAPIKey(m.APIKey)
			if keyDisplay == "" {
				keyDisplay = "(无API密钥)"
			}
			fmt.Printf("  %d. %s%s  %s  %s  (%s)\n", i+1, indicator, m.Name, m.BaseURL, keyDisplay, m.ToolType)
		}
		fmt.Println()
	}

	// 显示 Claude 镜像源
	if len(claudeMirrors) > 0 {
		fmt.Println("【Claude 类型】")
		for i := range claudeMirrors {
			m := claudeMirrors[i]
			indicator := "  "
			if mm.GetConfig().CurrentClaude == m.Name {
				indicator = "* "
			}
			keyDisplay := internal.MaskAPIKey(m.APIKey)
			if keyDisplay == "" {
				keyDisplay = "(无API密钥)"
			}
			fmt.Printf("  %d. %s%s  %s  %s  (%s)\n", i+len(codexMirrors)+1, indicator, m.Name, m.BaseURL, keyDisplay, m.ToolType)
		}
	}

	fmt.Println("==================================================")
	fmt.Println("输入编号选择镜像源 (直接回车取消): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("用户取消操作")
	}

	// 解析输入
	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(mirrors) {
		return "", fmt.Errorf("无效的选择，请输入 1-%d 之间的数字", len(mirrors))
	}

	return mirrors[idx-1].Name, nil
}

func init() {
	rootCmd.AddCommand(switchCmd)

	// 添加命令行标志.
	switchCmd.Flags().BoolVar(&codexOnly, "codex-only", false, "只更新Codex CLI配置")
	switchCmd.Flags().BoolVar(&vscodeOnly, "vscode-only", false, "只更新VS Code配置")
	switchCmd.Flags().BoolVar(&noBackup, "no-backup", false, "不备份现有配置")
	switchCmd.Flags().StringVar(&shellFmt, "shell", "", "输出适配当前shell的导出语句(bash|zsh|fish|powershell|cmd)")
	switchCmd.Flags().BoolVar(&useEnvVar, "env", false, "Claude类型使用系统环境变量方式（默认使用配置文件）")
	switchCmd.Flags().BoolVar(&dryRun, "dry-run", false, "预览切换效果，不实际修改配置")
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
