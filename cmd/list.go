package cmd

import (
	"fmt"
	"strings"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

// listCmd 代表list命令.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有镜像源",
	Long: `列出所有已配置的镜像源，并显示当前激活的配置。

示例：
  codex-mirror list`,
	Run: func(cmd *cobra.Command, args []string) {
		// 创建镜像源管理器
		mm, err := internal.NewMirrorManager()
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			return
		}

		// 获取所有镜像源
		mirrors := mm.ListMirrors()
		if len(mirrors) == 0 {
			fmt.Println("没有配置任何镜像源")
			return
		}

		// 获取当前激活的配置
		currentCodex, _ := mm.GetCurrentCodexMirror()
		currentClaude, _ := mm.GetCurrentClaudeMirror()

		fmt.Println("可用的镜像源:")
		fmt.Println(strings.Repeat("-", 70))
		fmt.Printf("%-20s %-10s %-40s %s\n", "名称", "类型", "URL", "状态")
		fmt.Println(strings.Repeat("-", 70))

		for _, mirror := range mirrors {
			// 确定状态
			status := ""
			if mirror.ToolType == internal.ToolTypeCodex && currentCodex != nil && mirror.Name == currentCodex.Name {
				status = "*"
			} else if mirror.ToolType == internal.ToolTypeClaude && currentClaude != nil && mirror.Name == currentClaude.Name {
				status = "*"
			}

			// 截断过长的URL
			url := mirror.BaseURL
			if len(url) > 38 {
				url = url[:35] + "..."
			}

			fmt.Printf("%-20s %-10s %-40s %s\n",
				mirror.Name,
				mirror.ToolType,
				url,
				status)
		}

		fmt.Println(strings.Repeat("-", 70))

		// 显示当前激活的配置
		fmt.Println("\n当前激活的配置:")
		if currentCodex != nil {
			fmt.Printf("  Codex:  %s (%s)\n", currentCodex.Name, currentCodex.BaseURL)
		} else {
			fmt.Printf("  Codex:  未设置\n")
		}

		if currentClaude != nil {
			fmt.Printf("  Claude: %s (%s)\n", currentClaude.Name, currentClaude.BaseURL)
		} else {
			fmt.Printf("  Claude: 未设置\n")
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
