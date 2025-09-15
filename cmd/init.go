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
        case "bash", "zsh", "sh":
            return setupPOSIX(shell)
        case "fish":
            return setupFish()
        case "powershell", "pwsh":
            return setupPowerShell()
        case "cmd", "bat":
            return fmt.Errorf("不支持 cmd 集成，请使用 PowerShell")
        default:
            // 尝试按 POSIX 处理
            return setupPOSIX("bash")
        }
    },
}

func init() {
    rootCmd.AddCommand(initCmd)
    initCmd.Flags().StringVar(&initShell, "shell", "", "指定 shell (bash|zsh|fish|powershell)")
}

func detectShell(platform internal.Platform) string {
    // If SHELL is set (including on Windows under MSYS/Git Bash), prefer it.
    if sh := os.Getenv("SHELL"); sh != "" {
        base := strings.ToLower(filepath.Base(sh))
        switch base {
        case "bash", "zsh", "fish", "sh":
            return base
        }
    }

    if platform == internal.PlatformWindows {
        // PowerShell heuristic: PSModulePath exists in PowerShell sessions
        if os.Getenv("PSModulePath") != "" {
            return "powershell"
        }
        // CMD heuristic
        if strings.Contains(strings.ToLower(os.Getenv("ComSpec")), "cmd.exe") {
            return "cmd"
        }
        // Fallback to PowerShell
        return "powershell"
    }

    // POSIX fallback
    return "bash"
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
    case "zsh":
        rc = filepath.Join(home, ".zshrc")
    case "bash", "sh":
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

    block := posixWrapperBlock()
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
    content := fishWrapperFunction()
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
    block := powershellWrapperBlock()
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

func posixWrapperBlock() string {
    // Use an alternate name to call the actual binary to avoid recursion: `command codex-mirror`
    // First run persists changes, then eval for instant effect.
    body := `# >>> codex-mirror init >>>
codex-mirror() {
  if [ "$#" -gt 0 ] && [ "$1" = "switch" ]; then
    shift
    # resolve binary name: prefer codex-mirror, fallback codex-mirror.exe (WSL)
    local _cm="codex-mirror"
    if ! command -v "$_cm" >/dev/null 2>&1; then
      if command -v codex-mirror.exe >/dev/null 2>&1; then
        _cm="codex-mirror.exe"
      fi
    fi
    command "$_cm" switch "$@" 1>&2
    code=$?
    if [ "$code" -eq 0 ]; then
      # remove possible CR from Windows exe when running under WSL
      eval "$(command \"$_cm\" switch \"$@\" --shell bash 2>/dev/null | tr -d '\r')"
    fi
    return $code
  else
    command codex-mirror "$@"
  fi
}
# <<< codex-mirror init <<<`
    return body
}

func fishWrapperFunction() string {
    return `function codex-mirror
  if test (count $argv) -gt 0; and test $argv[1] = switch
    set -l rest $argv[2..-1]
    # resolve binary name: prefer codex-mirror, fallback codex-mirror.exe (WSL)
    set -l cm codex-mirror
    if not type -q $cm
      if type -q codex-mirror.exe
        set cm codex-mirror.exe
      end
    end
    command $cm switch $rest 1>&2
    set -l code $status
    if test $code -eq 0
      set -l exports (command $cm switch $rest --shell fish 2>/dev/null | string replace -a '\r' '')
      if test (count $exports) -gt 0
        eval (string join \n $exports)
      end
    end
    return $code
  else
    command codex-mirror $argv
  end
end
`
}

func powershellWrapperBlock() string {
    // Use external application path to avoid recursive function invocation
    return `# >>> codex-mirror init >>>
# 优先将无扩展名/扩展名命令名指向函数（不影响显式路径 .\codex-mirror.exe）
Set-Alias codex-mirror.exe codex-mirror -Scope Global -Force

function codex-mirror {
  param([Parameter(ValueFromRemainingArguments=$true)][string[]]$Args)
  $exeCmd = (Get-Command codex-mirror -CommandType Application -ErrorAction SilentlyContinue)
  $cmd = if ($null -ne $exeCmd) { $exeCmd.Path } else { 'codex-mirror' }
  if ($Args.Length -gt 0 -and $Args[0] -eq 'switch') {
    # 1) Run original command as-typed for persistence/logs
    & $cmd @Args | Out-Host
    # 2) Run again adding --shell powershell to update current session
    $cmdArgs = @()
    $cmdArgs += $Args
    $cmdArgs += '--shell'
    $cmdArgs += 'powershell'
    $exports = (& $cmd @cmdArgs 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -eq 0 -and -not [string]::IsNullOrWhiteSpace($exports)) { Invoke-Expression $exports }
  } else {
    & $cmd @Args
  }
}
# <<< codex-mirror init <<<`
}
