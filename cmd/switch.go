package cmd

import (
	"fmt"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

var (
	// 命令行标志.
	codexOnly  bool
	vscodeOnly bool
	noBackup   bool
)

// switchCmd 代表switch命令.
var switchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "切换到指定的镜像源",
	Long: `切换到指定的镜像源，并更新相关配置文件。

默认情况下，会同时更新Codex CLI和VS Code的配置。
可以使用标志来只更新特定的配置。

参数：
  name  要切换到的镜像源名称

示例：
  codex-mirror switch myapi
  codex-mirror switch official --codex-only
  codex-mirror switch local --vscode-only
  codex-mirror switch myapi --no-backup`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mirrorName := args[0]

		// 创建镜像源管理器
		mm, err := internal.NewMirrorManager()
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			return
		}

		// 修复mirrors.toml中的env_key格式
		if err := mm.FixEnvKeyFormat(); err != nil {
			fmt.Printf("修复env_key格式失败: %v\n", err)
			return
		}

		// 切换镜像源
		if err := mm.SwitchMirror(mirrorName); err != nil {
			fmt.Printf("切换镜像源失败: %v\n", err)
			return
		}

		// 获取镜像源配置
		mirror, err := mm.GetCurrentMirror()
		if err != nil {
			fmt.Printf("获取镜像源配置失败: %v\n", err)
			return
		}

		fmt.Printf("正在切换到镜像源 '%s'...\n", mirrorName)

		// 更新Codex CLI配置
		if !vscodeOnly {
			if err := updateCodexConfig(mirror); err != nil {
				fmt.Printf("更新Codex配置失败: %v\n", err)
				return
			}
			fmt.Println("✓ Codex CLI配置已更新")
		}

		// 更新VS Code配置
		if !codexOnly {
			if err := updateVSCodeConfig(mirror); err != nil {
				fmt.Printf("更新VS Code配置失败: %v\n", err)
				return
			}
			fmt.Println("✓ VS Code配置已更新")
		}

		fmt.Printf("\n成功切换到镜像源 '%s'\n", mirrorName)
		fmt.Printf("  URL: %s\n", mirror.BaseURL)
		if mirror.APIKey != "" {
			fmt.Printf("  API密钥: %s\n", maskAPIKey(mirror.APIKey))
		}
	},
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

func init() {
	rootCmd.AddCommand(switchCmd)

	// 添加命令行标志.
	switchCmd.Flags().BoolVar(&codexOnly, "codex-only", false, "只更新Codex CLI配置")
	switchCmd.Flags().BoolVar(&vscodeOnly, "vscode-only", false, "只更新VS Code配置")
	switchCmd.Flags().BoolVar(&noBackup, "no-backup", false, "不备份现有配置")
}
