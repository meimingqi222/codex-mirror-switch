package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

var (
	initShell string
)

// initCmd installs a small shell wrapper so that `codex-mirror switch` instantly updates
// the current shell session by evaluating `--shell` output automatically.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "安装 shell 集成，实现 switch 后即时生效",
	Long: `在当前用户的 shell 配置文件中安装包装函数：

安装后，直接运行 "codex-mirror switch <name>" 将自动：
1) 执行常规切换（写入配置/持久化），并将日志输出到 stderr；
2) 评估 "--shell" 输出，让当前会话立即生效。

支持 bash/zsh/fish/PowerShell，可通过 --shell 显式指定。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		platform := internal.GetCurrentPlatform()
		shell := strings.ToLower(strings.TrimSpace(initShell))
		if shell == "" {
			shell = detectShell(platform)
		}

		switch shell {
		case internal.BashShell, internal.ZshShell, "sh":
			return setupPOSIX(shell)
		case internal.FishShell:
			return setupFish()
		case internal.PowerShellShell, internal.PwshShell:
			return setupPowerShell()
		case internal.CmdShell, internal.BatShell:
			return fmt.Errorf("不支持 cmd 集成，请使用 PowerShell")
		default:
			// 尝试按 POSIX 处理
			return setupPOSIX(internal.BashShell)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&initShell, "shell", "", fmt.Sprintf("指定 shell (%s|%s|%s|%s)", internal.BashShell, internal.ZshShell, internal.FishShell, internal.PowerShellShell))
}

func detectShell(platform internal.Platform) string {
	// If SHELL is set (including on Windows under MSYS/Git Bash), prefer it.
	if sh := os.Getenv("SHELL"); sh != "" {
		base := strings.ToLower(filepath.Base(sh))
		switch base {
		case internal.BashShell, internal.ZshShell, internal.FishShell, "sh":
			return base
		}
	}

	if platform == internal.PlatformWindows {
		// PowerShell heuristic: PSModulePath exists in PowerShell sessions
		if os.Getenv("PSModulePath") != "" {
			return internal.PowerShellShell
		}
		// CMD heuristic
		if strings.Contains(strings.ToLower(os.Getenv("ComSpec")), "cmd.exe") {
			return internal.CmdShell
		}
		// Fallback to PowerShell
		return internal.PowerShellShell
	}

	// POSIX fallback
	return internal.BashShell
}

func setupPOSIX(shell string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	// Pick best rc file:
	// - zsh: ~/.zshrc
	// - bash: prefer an existing one among ~/.bashrc, ~/.bash_profile; if none, choose
	//         ~/.bash_profile on macOS or ~/.bashrc on Linux.
	var rc string
	switch shell {
	case internal.ZshShell:
		rc = filepath.Join(home, ".zshrc")
	case internal.BashShell, "sh":
		// Check existing
		cand := []string{filepath.Join(home, ".bashrc"), filepath.Join(home, ".bash_profile")}
		for _, c := range cand {
			if _, err := os.Stat(c); err == nil {
				rc = c
				break
			}
		}
		if rc == "" {
			// Decide based on platform
			if internal.GetCurrentPlatform() == internal.PlatformMac {
				rc = filepath.Join(home, ".bash_profile")
			} else {
				rc = filepath.Join(home, ".bashrc")
			}
		}
	default:
		rc = filepath.Join(home, ".bashrc")
	}

	if err := ensureFile(rc); err != nil {
		return fmt.Errorf("创建配置文件失败: %w", err)
	}

	// 获取当前可执行文件的绝对路径
	execPath, err := getCurrentExecutablePath()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	block := posixWrapperBlock(execPath)
	if err := upsertBlock(rc, block); err != nil {
		return err
	}

	fmt.Printf("已在 %s 安装 codex-mirror shell 集成。请执行: source %s 或重开终端。\n", rc, rc)
	return nil
}

func setupFish() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".config", "fish", "functions")
	if err := internal.EnsureDir(dir); err != nil {
		return err
	}
	path := filepath.Join(dir, "codex-mirror.fish")

	// 获取当前可执行文件的绝对路径
	execPath, err := getCurrentExecutablePath()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	content := fishWrapperFunction(execPath)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("已在 %s 安装 codex-mirror.fish。fish 会自动加载该函数。\n", path)
	return nil
}

func setupPowerShell() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	// Candidate locations (PowerShell 7+ and Windows PowerShell 5.1)
	var candidates []string
	candidates = append(candidates,
		filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
	)
	// OneDrive redirected Documents
	if od := os.Getenv("OneDrive"); od != "" {
		candidates = append(candidates,
			filepath.Join(od, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
			filepath.Join(od, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		)
	}

	// Pick the first whose parent directory exists; otherwise use the first and create dirs
	var path string
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Dir(c)); err == nil {
			path = c
			break
		}
	}
	if path == "" {
		path = candidates[0]
	}
	if err := ensureFile(path); err != nil {
		return err
	}

	// 获取当前可执行文件的绝对路径
	execPath, err := getCurrentExecutablePath()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	block := powershellWrapperBlock(execPath)
	if err := upsertBlock(path, block); err != nil {
		return err
	}
	fmt.Printf("已在 %s 安装 codex-mirror PowerShell 集成。请重启 PowerShell 或执行 . '%s'\n", path, path)
	return nil
}

func ensureFile(path string) error {
	if err := internal.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.WriteFile(path, []byte("\n"), 0o644)
	}
	return nil
}

// getCurrentExecutablePath 获取当前正在运行的可执行文件的绝对路径.
func getCurrentExecutablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	// 解析符号链接，获取真实路径
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return execPath, nil // 如果无法解析符号链接，返回原路径
	}
	return realPath, nil
}

const startMarker = "# >>> codex-mirror init >>>"
const endMarker = "# <<< codex-mirror init <<<"

func upsertBlock(path, block string) error {
	data, _ := os.ReadFile(path)
	content := string(data)
	start := strings.Index(content, startMarker)
	end := strings.Index(content, endMarker)

	if start >= 0 && end > start {
		// 替换现有块
		newContent := content[:start] + block + content[end+len(endMarker):]
		return os.WriteFile(path, []byte(newContent), 0o644)
	}
	// 追加
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	newContent := content + "\n" + block + "\n"
	return os.WriteFile(path, []byte(newContent), 0o644)
}

func posixWrapperBlock(execPath string) string {
	// Use the absolute path to the executable to avoid PATH dependency
	// First run persists changes, then eval for instant effect.
	body := fmt.Sprintf(`# >>> codex-mirror init >>>
codex-mirror() {
  local _cm="%s"
  if [ "$#" -gt 0 ] && [ "$1" = "switch" ]; then
    shift
    # Run the actual command with absolute path
    "$_cm" switch "$@" 1>&2
    code=$?
    if [ "$code" -eq 0 ]; then
      # remove possible CR from Windows exe when running under WSL
      eval "$($_cm switch $@ --shell bash 2>/dev/null | tr -d '\r')"
    fi
    return $code
  else
    "$_cm" "$@"
  fi
}
# <<< codex-mirror init <<<`, execPath)
	return body
}

func fishWrapperFunction(execPath string) string {
	return fmt.Sprintf(`function codex-mirror
  set -l _cm "%s"
  if test (count $argv) -gt 0; and test $argv[1] = switch
    set -l rest $argv[2..-1]
    # Use absolute path to executable
    $_cm switch $rest 1>&2
    set -l code $status
    if test $code -eq 0
      set -l exports ($_cm switch $rest --shell fish 2>/dev/null | string replace -a '\r' '')
      if test (count $exports) -gt 0
        eval (string join \n $exports)
      end
    end
    return $code
  else
    $_cm $argv
  end
end
`, execPath)
}

func powershellWrapperBlock(execPath string) string {
	// Use the absolute path to the executable to avoid PATH dependency
	return fmt.Sprintf(`# >>> codex-mirror init >>>
function codex-mirror {
  param([Parameter(ValueFromRemainingArguments=$true)][string[]]$Args)
  $cmd = "%s"
  if ($Args.Length -gt 0 -and $Args[0] -eq 'switch') {
    # 1) Run original command as-typed for persistence/logs
    & $cmd @Args | Out-Host
    # 2) Run again adding --shell powershell to update current session
    $switchArgs = $Args[1..($Args.Length-1)]  # Get arguments after 'switch'
    $exports = (& $cmd 'switch' @switchArgs '--shell' 'powershell' 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -eq 0 -and -not [string]::IsNullOrWhiteSpace($exports)) { Invoke-Expression $exports }
  } else {
    & $cmd @Args
  }
}
# <<< codex-mirror init <<<`, execPath)
}
