package cmd

import (
	"codex-mirror/internal/tui"

	"github.com/spf13/cobra"
)

// tuiCmd 代表 tui 命令.
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "启动交互式 TUI 界面",
	Long:  `启动用于管理镜像源的交互式文本用户界面 (TUI)。`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = tui.Start()
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
