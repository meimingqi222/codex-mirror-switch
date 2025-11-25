package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// 版本信息变量，通过 ldflags 在构建时注入.
// 构建命令示例:
// go build -ldflags "-X codex-mirror/cmd.Version=1.2.0 -X codex-mirror/cmd.GitCommit=abc123 -X codex-mirror/cmd.BuildTime=2024-01-01T00:00:00Z" .
var (
	// Version 版本号.
	Version = "dev"
	// GitCommit Git 提交 hash.
	GitCommit = "unknown"
	// BuildTime 构建时间.
	BuildTime = "unknown"
)

// versionCmd 代表 version 命令.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long:  `显示 codex-mirror 的版本、构建时间和 Git 提交信息。`,
	Run: func(cmd *cobra.Command, args []string) {
		short, _ := cmd.Flags().GetBool("short")
		if short {
			fmt.Println(Version)
			return
		}

		fmt.Printf("codex-mirror %s\n", Version)
		fmt.Printf("  Git Commit: %s\n", GitCommit)
		fmt.Printf("  Build Time: %s\n", BuildTime)
		fmt.Printf("  Go Version: %s\n", runtime.Version())
		fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	versionCmd.Flags().BoolP("short", "s", false, "只显示版本号")
	rootCmd.AddCommand(versionCmd)
}
