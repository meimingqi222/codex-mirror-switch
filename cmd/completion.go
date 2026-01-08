package cmd

import (
	"bytes"
	"codex-mirror/internal"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// completionCmd 代表 completion 命令.
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for your shell.

To load completions:

Bash:
  $ source <(codex-mirror completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ codex-mirror completion bash > /etc/bash_completion.d/codex-mirror
  # macOS:
  $ codex-mirror completion bash > /usr/local/etc/bash_completion.d/codex-mirror

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ codex-mirror completion zsh > "${fpath[1]}/_codex-mirror"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ codex-mirror completion fish | source

  # To load completions for each session, execute once:
  $ codex-mirror completion fish > ~/.config/fish/completions/codex-mirror.fish

PowerShell:
  # To load completions for each session, execute once:
  $ codex-mirror completion powershell | Invoke-Expression

  # To install permanently, add to your profile:
  $ codex-mirror completion powershell >> $PROFILE
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletionV2(cmd.OutOrStdout(), false)
		case "zsh":
			return rootCmd.GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return generatePowerShellCompletion(cmd.OutOrStdout())
		default:
			return nil
		}
	},
}

// generatePowerShellCompletion 生成 PowerShell 补全脚本（带安装说明）.
func generatePowerShellCompletion(w io.Writer) error {
	// 先生成补全脚本
	var buf bytes.Buffer
	if err := rootCmd.GenPowerShellCompletion(&buf); err != nil {
		return err
	}
	completionScript := buf.String()

	// 包装成安全的可追加脚本
	script := fmt.Sprintf(`# codex-mirror shell completion
if (Get-Command codex-mirror -ErrorAction SilentlyContinue) {
%s
}
`, completionScript)

	_, err := w.Write([]byte(script))
	return err
}

func init() {
	rootCmd.AddCommand(completionCmd)

	// 为镜像源名称添加补全函数
	_ = rootCmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"codex", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})

	// 为 switch 命令的镜像源参数添加补全
	switchCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getMirrorNamesForCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

// getMirrorNamesForCompletion 获取可补全的镜像源名称列表.
func getMirrorNamesForCompletion(toComplete string) []string {
	mm, err := internal.NewMirrorManager()
	if err != nil {
		return nil
	}

	mirrors := mm.ListActiveMirrors()
	var names []string
	for _, m := range mirrors {
		if hasPrefix(m.Name, toComplete) {
			names = append(names, m.Name)
		}
	}
	return names
}

// hasPrefix 检查字符串是否以指定前缀开头（不区分大小写）.
func hasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return len(prefix) == 0 || equalFold(s[:len(prefix)], prefix)
}

// equalFold 比较字符串（不区分大小写）.
func equalFold(s1, s2 string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if lower(s1[i]) != lower(s2[i]) {
			return false
		}
	}
	return true
}

// lower 将字符转换为小写.
func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
