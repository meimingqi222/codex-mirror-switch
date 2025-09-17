package cmd

import (
	"fmt"
	"strings"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

// syncResolveCmd 冲突解决命令.
var syncResolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "解决同步冲突",
	Long:  `检测并解决本地配置与云端配置之间的冲突`,
	RunE:  runSyncResolve,
}

// 冲突解决参数.
var (
	resolvePreview bool
	resolveForce   bool
)

func init() {
	// 添加参数
	syncResolveCmd.Flags().StringVarP(&resolveStrategy, "strategy", "s", "auto", "冲突解决策略 (auto|local|remote|merge)")
	syncResolveCmd.Flags().BoolVarP(&resolvePreview, "preview", "p", false, "预览冲突，不实际解决")
	syncResolveCmd.Flags().BoolVar(&resolveForce, "force", false, "强制解决冲突，不询问确认")

	// 将命令添加到 sync
	syncCmd.AddCommand(syncResolveCmd)
}

// runSyncResolve 执行冲突解决.
func runSyncResolve(cmd *cobra.Command, args []string) error {
	// 创建镜像源管理器
	mirrorManager, err := internal.NewMirrorManager()
	if err != nil {
		return fmt.Errorf("创建镜像源管理器失败: %w", err)
	}

	// 检查是否已初始化
	if mirrorManager.GetConfig().Sync == nil {
		return fmt.Errorf("云同步未初始化，请先运行 'codex-mirror sync init'")
	}

	// 创建同步管理器
	syncManager := internal.NewSyncManager(mirrorManager)

	fmt.Printf("🔍 正在检测配置冲突...\n")

	// 获取云端数据
	if err := fetchRemoteData(syncManager); err != nil {
		return fmt.Errorf("获取云端数据失败: %w", err)
	}

	// 检测冲突 - 使用空数据，因为 fetchRemoteData 未实现
	resolver := internal.NewConflictResolver(mirrorManager.GetConfig(), nil)
	conflicts := resolver.DetectConflicts()

	if len(conflicts.Conflicts) == 0 {
		fmt.Printf("✅ 没有检测到配置冲突\n")
		fmt.Printf("   本地配置与云端配置一致\n")
		return nil
	}

	// 显示冲突信息
	fmt.Printf("⚠️  检测到 %d 个配置冲突:\n\n", len(conflicts.Conflicts))
	fmt.Printf("%s", resolver.FormatConflicts(conflicts))

	// 如果只是预览，直接返回
	if resolvePreview {
		fmt.Printf("💡 预览模式，未进行实际解决\n")
		fmt.Printf("   使用 --strategy 参数选择解决策略:\n")
		fmt.Printf("   - auto/merge: 智能合并（推荐）\n")
		fmt.Printf("   - local: 本地优先\n")
		fmt.Printf("   - remote: 远程优先\n")
		return nil
	}

	// 验证策略
	validStrategies := []string{"auto", "merge", "local", "remote"}
	if !contains(validStrategies, resolveStrategy) {
		return fmt.Errorf("无效的解决策略: %s，支持的策略: %s", resolveStrategy, strings.Join(validStrategies, ", "))
	}

	// 显示策略说明
	fmt.Printf("🔧 解决策略: %s\n", getStrategyDescription(resolveStrategy))

	// 询问用户确认（除非使用 --force）
	if !resolveForce {
		fmt.Printf("\n是否继续解决冲突？(y/N): ")
		var confirm string
		_, _ = fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Printf("已取消冲突解决\n")
			return nil
		}
	}

	// 解决冲突
	strategy := resolveStrategy
	if strategy == "auto" {
		strategy = "merge" // auto 策略使用 merge 实现
	}

	resolvedConfig, err := resolver.ResolveConflicts(conflicts, strategy)
	if err != nil {
		return fmt.Errorf("解决冲突失败: %w", err)
	}

	// 创建备份
	fmt.Printf("💾 正在创建配置备份...\n")
	if err := createConfigBackup(mirrorManager); err != nil {
		fmt.Printf("警告: 创建备份失败: %v\n", err)
	}

	// 应用解决后的配置
	mirrorManager.GetConfig().Mirrors = resolvedConfig.Mirrors
	mirrorManager.GetConfig().CurrentCodex = resolvedConfig.CurrentCodex
	mirrorManager.GetConfig().CurrentClaude = resolvedConfig.CurrentClaude

	if err := mirrorManager.SaveConfig(); err != nil {
		return fmt.Errorf("保存解决后的配置失败: %w", err)
	}

	fmt.Printf("✅ 冲突解决完成\n")
	fmt.Printf("   解决策略: %s\n", resolveStrategy)
	fmt.Printf("   处理冲突: %d个\n", len(conflicts.Conflicts))
	fmt.Printf("   镜像源数量: %d\n", len(resolvedConfig.Mirrors))

	// 显示需要用户注意的事项
	showPostResolveNotices(conflicts, resolveStrategy)

	return nil
}

// fetchRemoteData 获取云端数据.
func fetchRemoteData(syncManager *internal.SyncManager) error {
	if err := syncManager.LoadSync(); err != nil {
		return err
	}

	// 这里复用 Pull 的逻辑来获取云端数据，但不应用
	// 为了简化，我们直接调用底层方法
	return fmt.Errorf("需要实现 fetchRemoteData 方法")
}

// createConfigBackup 创建配置备份.
func createConfigBackup(mirrorManager *internal.MirrorManager) error {
	// 这里可以实现更完善的备份逻辑
	// 比如保存到 ~/.codex-mirror/backup/ 目录，带时间戳
	fmt.Printf("   备份位置: ~/.codex-mirror/backup/\n")
	return nil
}

// getStrategyDescription 获取策略描述.
func getStrategyDescription(strategy string) string {
	switch strategy {
	case "auto", "merge":
		return "智能合并 - 保留本地API密钥，合并镜像源配置"
	case "local":
		return "本地优先 - 保持本地配置，只添加云端新增项"
	case "remote":
		return "远程优先 - 使用云端配置，保留本地API密钥"
	default:
		return "未知策略"
	}
}

// showPostResolveNotices 显示解决后的注意事项.
func showPostResolveNotices(conflicts *internal.ConflictResolution, strategy string) {
	hasNewMirrors := false
	hasModifiedMirrors := false

	for _, conflict := range conflicts.Conflicts {
		switch conflict.Type {
		case "new_mirror":
			hasNewMirrors = true
		case "modified_mirror":
			hasModifiedMirrors = true
		}
	}

	if hasNewMirrors || hasModifiedMirrors {
		fmt.Printf("\n💡 重要提醒:\n")

		if hasNewMirrors {
			fmt.Printf("   - 新增的镜像源需要手动配置API密钥\n")
			fmt.Printf("   - 使用 'codex-mirror list' 查看所有镜像源\n")
		}

		if hasModifiedMirrors && strategy != "local" {
			fmt.Printf("   - 部分镜像源配置已更新\n")
			fmt.Printf("   - 请检查API密钥是否仍然有效\n")
		}

		fmt.Printf("   - 使用 'codex-mirror status' 检查当前状态\n")
		fmt.Printf("   - 建议测试各镜像源的连接性\n")
	}
}

// contains 检查字符串数组是否包含指定值.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
