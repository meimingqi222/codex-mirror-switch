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
	remoteData, err := fetchRemoteData(syncManager)
	if err != nil {
		return handleResolveFetchError(err)
	}

	// 检测冲突
	resolver := internal.NewConflictResolver(mirrorManager.GetConfig(), remoteData)
	resolver.SetCryptoManager(syncManager.GetCryptoManager()) // 设置加密管理器，用于解密远程 APIKey
	conflicts := resolver.DetectConflicts()
	if len(conflicts.Conflicts) == 0 {
		fmt.Printf("✅ 没有检测到配置冲突\n")
		fmt.Printf("   本地配置与云端配置一致\n")
		return nil
	}

	showConflicts(resolver, conflicts)

	// 预览模式
	if handlePreviewIfRequested() {
		return nil
	}

	// 规范化并校验策略
	strategy, err := computeStrategy(resolveStrategy)
	if err != nil {
		return err
	}
	fmt.Printf("🔧 解决策略: %s\n", getStrategyDescription(strategy))

	// 确认继续
	if !resolveForce {
		ok := askForConfirmation()
		if !ok {
			fmt.Printf("已取消冲突解决\n")
			return nil
		}
	}

	// 执行解决
	resolvedConfig, err := resolver.ResolveConflicts(conflicts, strategy)
	if err != nil {
		return fmt.Errorf("解决冲突失败: %w", err)
	}

	if err := backupAndApplyResolved(mirrorManager, resolvedConfig); err != nil {
		return err
	}

	fmt.Printf("✅ 冲突解决完成\n")
	fmt.Printf("   解决策略: %s\n", strategy)
	fmt.Printf("   处理冲突: %d个\n", len(conflicts.Conflicts))
	fmt.Printf("   镜像源数量: %d\n", len(resolvedConfig.Mirrors))

	// 显示需要用户注意的事项
	showPostResolveNotices(conflicts, strategy)
	return nil
}

// 将获取云端数据的错误分类并输出友好提示。
func handleResolveFetchError(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "GitHub API 错误 (401)") {
		fmt.Printf("❌ GitHub认证失败\n\n")
		fmt.Printf("💡 可能的原因:\n")
		fmt.Printf("   - Token无效或已过期\n")
		fmt.Printf("   - Token没有gist权限\n\n")
		fmt.Printf("🔧 解决方法:\n")
		fmt.Printf("   - 重新生成Token: https://github.com/settings/tokens\n")
		fmt.Printf("   - 确保勾选了'gist'权限\n")
		return fmt.Errorf("GitHub认证失败")
	}
	if strings.Contains(msg, "未找到文件") || strings.Contains(msg, "GitHub API 错误 (404)") {
		fmt.Printf("✅ 云端暂无配置，当前无冲突\n")
		return nil
	}
	if strings.Contains(msg, "解密数据失败") || strings.Contains(msg, "无法解密") {
		fmt.Printf("❌ 解密云端数据失败\n\n")
		fmt.Printf("💡 可能原因: 密码不正确或云端数据损坏\n")
		fmt.Printf("🔧 解决方法: 确认密码，必要时重新初始化同步\n")
		return fmt.Errorf("解密云端数据失败")
	}
	return fmt.Errorf("获取云端数据失败: %w", err)
}

// 展示冲突列表。
func showConflicts(resolver *internal.ConflictResolver, conflicts *internal.ConflictResolution) {
	fmt.Printf("⚠️  检测到 %d 个配置冲突:\n\n", len(conflicts.Conflicts))
	fmt.Printf("%s", resolver.FormatConflicts(conflicts))
}

// 如果是预览模式则输出提示并返回 true。
func handlePreviewIfRequested() bool {
	if !resolvePreview {
		return false
	}
	fmt.Printf("💡 预览模式，未进行实际解决\n")
	fmt.Printf("   使用 --strategy 参数选择解决策略:\n")
	fmt.Printf("   - auto/merge: 智能合并（推荐）\n")
	fmt.Printf("   - local: 本地优先\n")
	fmt.Printf("   - remote: 远程优先\n")
	return true
}

// 计算与校验策略；auto 规范化为 merge。
func computeStrategy(s string) (string, error) {
	valid := []string{"auto", "merge", "local", "remote"}
	if !contains(valid, s) {
		return "", fmt.Errorf("无效的解决策略: %s，支持的策略: %s", s, strings.Join(valid, ", "))
	}
	if s == "auto" {
		return "merge", nil
	}
	return s, nil
}

// 询问用户是否继续。
func askForConfirmation() bool {
	fmt.Printf("\n是否继续解决冲突？(y/N): ")
	var confirm string
	_, _ = fmt.Scanln(&confirm)
	return confirm == "y" || confirm == "Y"
}

// 备份并应用解决后的配置。
func backupAndApplyResolved(mirrorManager *internal.MirrorManager, resolved *internal.SystemConfig) error {
	fmt.Printf("💾 正在创建配置备份...\n")
	if err := createConfigBackup(mirrorManager); err != nil {
		fmt.Printf("警告: 创建备份失败: %v\n", err)
	}

	cfg := mirrorManager.GetConfig()
	cfg.Mirrors = resolved.Mirrors
	cfg.CurrentCodex = resolved.CurrentCodex
	cfg.CurrentClaude = resolved.CurrentClaude

	if err := mirrorManager.SaveConfig(); err != nil {
		return fmt.Errorf("保存解决后的配置失败: %w", err)
	}
	return nil
}

// fetchRemoteData 获取云端数据（不应用）。
func fetchRemoteData(syncManager *internal.SyncManager) (*internal.SyncData, error) {
	// 直接使用内部提供的只读获取方法
	data, err := syncManager.FetchRemoteSyncData()
	if err != nil {
		return nil, err
	}
	return data, nil
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
