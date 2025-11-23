package cmd

import (
	"fmt"
	"os"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

// removeCmd 代表remove命令.
var removeCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "删除指定的镜像源",
	Long: `删除指定的镜像源配置。

注意：
- 不能删除官方镜像源
- 如果删除的是当前使用的镜像源，会自动切换到官方镜像源

参数：
  name  要删除的镜像源名称

示例：
  codex-mirror remove myapi
  codex-mirror remove local`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mirrorName := args[0]

		// 创建镜像源管理器
		mm, err := internal.NewMirrorManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
			return
		}

		// 检查镜像源是否存在（只检查未删除的）
		mirrors := mm.ListActiveMirrors()
		found := false
		for _, mirror := range mirrors {
			if mirror.Name == mirrorName {
				found = true
				break
			}
		}

		if !found {
			fmt.Printf("镜像源 '%s' 不存在\n", mirrorName)
			return
		}

		// 检查是否为当前使用的镜像源
		currentMirror, err := mm.GetCurrentMirror()
		if err != nil {
			fmt.Fprintf(os.Stderr, "获取当前镜像源失败: %v\n", err)
			os.Exit(1)
			return
		}

		isCurrentMirror := currentMirror.Name == mirrorName

		// 删除镜像源
		if err := mm.RemoveMirror(mirrorName); err != nil {
			fmt.Fprintf(os.Stderr, "删除镜像源失败: %v\n", err)
			os.Exit(1)
			return
		}

		fmt.Printf("成功删除镜像源 '%s'\n", mirrorName)

		// 如果删除的是当前镜像源，提示用户已切换到官方镜像源
		if isCurrentMirror {
			fmt.Println("由于删除的是当前使用的镜像源，已自动切换到官方镜像源")
			fmt.Println("请运行 'codex-mirror switch official' 来更新配置文件")
		}
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
