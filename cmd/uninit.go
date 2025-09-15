package cmd

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "codex-mirror/internal"

    "github.com/spf13/cobra"
)

var uninitShell string

// uninitCmd removes previously installed shell integration blocks/files.
var uninitCmd = &cobra.Command{
    Use:   "uninit",
    Short: "卸载 shell 集成，恢复默认行为",
    Long:  "移除在 init 时写入的 shell 包装函数或集成片段。",
    RunE: func(cmd *cobra.Command, args []string) error {
        platform := internal.GetCurrentPlatform()
        shell := strings.ToLower(strings.TrimSpace(uninitShell))

        // 在 POSIX 平台上，若未显式指定 --shell，则同时尝试 .bashrc 与 .zshrc
        if shell == "" && (platform == internal.PlatformLinux || platform == internal.PlatformMac) {
            return removePOSIXBoth()
        }

        if shell == "" {
            shell = detectShell(platform)
        }

        switch shell {
        case "bash", "zsh", "sh":
            return removePOSIX(shell)
        case "fish":
            return removeFish()
        case "powershell", "pwsh":
            return removePowerShell()
        case "cmd", "bat":
            fmt.Println("未安装 cmd 集成，无需卸载（推荐使用 PowerShell）")
            return nil
        default:
            // 尝试按 POSIX 处理
            return removePOSIX("bash")
        }
    },
}

func init() {
    rootCmd.AddCommand(uninitCmd)
    uninitCmd.Flags().StringVar(&uninitShell, "shell", "", "指定 shell (bash|zsh|fish|powershell)")
}

const rmStartMarker = "# >>> codex-mirror init >>>"
const rmEndMarker = "# <<< codex-mirror init <<<"

func removePOSIX(shell string) error {
    home, err := os.UserHomeDir()
    if err != nil {
        return err
    }
    var rc string
    switch shell {
    case "zsh":
        rc = filepath.Join(home, ".zshrc")
    case "bash", "sh":
        rc = filepath.Join(home, ".bashrc")
    default:
        rc = filepath.Join(home, ".bashrc")
    }

    removed, err := removeBlock(rc)
    if err != nil {
        return err
    }
    if removed {
        fmt.Printf("已从 %s 移除 codex-mirror 集成片段。请执行: source %s 或重开终端。\n", rc, rc)
    } else {
        fmt.Printf("在 %s 未找到 codex-mirror 集成片段。\n", rc)
    }
    return nil
}

// removePOSIXBoth 尝试同时从 ~/.bashrc 与 ~/.zshrc 中移除集成片段。
func removePOSIXBoth() error {
    home, err := os.UserHomeDir()
    if err != nil {
        return err
    }
    rcs := []string{
        filepath.Join(home, ".bashrc"),
        filepath.Join(home, ".zshrc"),
    }
    any := false
    for _, rc := range rcs {
        if _, err := os.Stat(rc); err != nil {
            continue
        }
        removed, err := removeBlock(rc)
        if err != nil {
            return err
        }
        if removed {
            fmt.Printf("已从 %s 移除 codex-mirror 集成片段。\n", rc)
            any = true
        }
    }
    if !any {
        fmt.Println("未在 ~/.bashrc 或 ~/.zshrc 中找到 codex-mirror 集成片段。")
    } else {
        fmt.Println("如需立即生效，请执行: source ~/.bashrc 或 source ~/.zshrc，或重开终端。")
    }
    return nil
}

func removeFish() error {
    home, err := os.UserHomeDir()
    if err != nil {
        return err
    }
    path := filepath.Join(home, ".config", "fish", "functions", "codex-mirror.fish")
    if _, err := os.Stat(path); os.IsNotExist(err) {
        fmt.Printf("未找到 %s，无需卸载。\n", path)
        return nil
    }
    if err := os.Remove(path); err != nil {
        return err
    }
    fmt.Printf("已删除 %s。\n", path)
    return nil
}

func removePowerShell() error {
    home, err := os.UserHomeDir()
    if err != nil {
        return err
    }
    var candidates []string
    candidates = append(candidates,
        filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
        filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
    )
    if od := os.Getenv("OneDrive"); od != "" {
        candidates = append(candidates,
            filepath.Join(od, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
            filepath.Join(od, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
        )
    }

    anyRemoved := false
    for _, p := range candidates {
        if _, err := os.Stat(p); err != nil {
            continue
        }
        removed, err := removeBlock(p)
        if err != nil {
            return err
        }
        if removed {
            fmt.Printf("已从 %s 移除 codex-mirror 集成片段。\n", p)
            anyRemoved = true
        }
    }
    if !anyRemoved {
        fmt.Println("未在任何 PowerShell Profile 中找到 codex-mirror 集成片段。")
    } else {
        fmt.Println("请重启 PowerShell 或执行 . \"$PROFILE\" 使更改生效。")
    }
    return nil
}

func removeBlock(path string) (bool, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return false, err
    }
    content := string(data)
    start := strings.Index(content, rmStartMarker)
    end := strings.Index(content, rmEndMarker)
    if start >= 0 && end > start {
        newContent := content[:start] + content[end+len(rmEndMarker):]
        if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
            return false, err
        }
        return true, nil
    }
    return false, nil
}
