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
	Long: `列出所有已配置的镜像源，并显示当前使用的镜像源。

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

		// 获取当前镜像源
		currentMirror, err := mm.GetCurrentMirror()
		if err != nil {
			fmt.Printf("获取当前镜像源失败: %v\n", err)
			return
		}

		fmt.Println("可用的镜像源:")
		fmt.Println(strings.Repeat("-", 60))

		for _, mirror := range mirrors {
			prefix := "  "
			if mirror.Name == currentMirror.Name {
				prefix = "* " // 标记当前使用的镜像源.
			}

			fmt.Printf("%s%s\n", prefix, mirror.Name)
			fmt.Printf("    URL: %s\n", mirror.BaseURL)
			if mirror.APIKey != "" {
				fmt.Printf("    API密钥: %s\n", maskAPIKey(mirror.APIKey))
			} else {
				fmt.Printf("    API密钥: 未设置\n")
			}
			fmt.Println()
		}

		fmt.Printf("当前使用: %s\n", currentMirror.Name)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
