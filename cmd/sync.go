package cmd

import (
	"fmt"
	"strings"

	"codex-mirror/internal"
	"github.com/spf13/cobra"
)

// syncCmd 云同步根命令.
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "云同步管理",
	Long:  `管理配置的云同步功能，支持多设备间的配置同步`,
}

// syncInitCmd 初始化云同步命令.
var syncInitCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化云同步",
	Long:  `初始化云同步功能，配置GitHub Token和加密密码。使用 'codex-mirror sync help' 查看详细帮助。`,
	RunE:  runSyncInit,
}

// syncPushCmd 推送配置命令.
var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "推送配置到云端",
	Long:  `将当前配置推送到云端存储`,
	RunE:  runSyncPush,
}

// syncPullCmd 拉取配置命令.
var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "从云端拉取配置",
	Long:  `从云端存储拉取配置并应用到本地`,
	RunE:  runSyncPull,
}

// syncStatusCmd 查看同步状态命令.
var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看同步状态",
	Long:  `查看当前云同步的配置和状态信息`,
	RunE:  runSyncStatus,
}

// syncConfigCmd 配置同步设置命令.
var syncConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "配置同步设置",
	Long:  `配置云同步的各项设置，如自动同步、同步间隔等`,
	RunE:  runSyncConfig,
}

// 命令行参数
var (
	syncToken        string
	syncAutoSync     bool
	syncInterval     int
	syncDisable      bool
	syncEncryptPwd   string
	resolveStrategy  string
	pushStrategy     string
	syncGistID       string
)

func init() {
	// 添加子命令
	syncCmd.AddCommand(syncInitCmd)
	syncCmd.AddCommand(syncPushCmd)
	syncCmd.AddCommand(syncPullCmd)
	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncConfigCmd)

	// syncInitCmd 参数
	syncInitCmd.Flags().StringVarP(&syncToken, "token", "t", "", "GitHub访问令牌 (必需)")
	syncInitCmd.Flags().StringVarP(&syncEncryptPwd, "password", "p", "", "加密密码 (必需)")
	syncInitCmd.Flags().StringVar(&syncGistID, "gist-id", "", "现有的Gist ID (可选，用于连接到现有配置)")
	syncInitCmd.MarkFlagRequired("token")
	syncInitCmd.MarkFlagRequired("password")

	// syncConfigCmd 参数
	syncConfigCmd.Flags().BoolVar(&syncAutoSync, "auto-sync", false, "启用自动同步")
	syncConfigCmd.Flags().IntVar(&syncInterval, "interval", 30, "同步间隔(分钟)")
	syncConfigCmd.Flags().BoolVar(&syncDisable, "disable", false, "禁用云同步")
	syncConfigCmd.Flags().StringVar(&syncEncryptPwd, "password", "", "更改加密密码")

	// syncPushCmd 参数
	syncPushCmd.Flags().StringVar(&pushStrategy, "strategy", "auto", "推送策略 (auto|merge|force|manual)")

	// syncPullCmd 参数
	syncPullCmd.Flags().StringVar(&resolveStrategy, "strategy", "auto", "冲突解决策略 (auto|local|remote|merge)")

	// 将 sync 命令添加到根命令
	rootCmd.AddCommand(syncCmd)
}

// runSyncInit 执行同步初始化.
func runSyncInit(cmd *cobra.Command, args []string) error {
	// 验证参数
	if syncToken == "" {
		fmt.Printf("❌ GitHub访问令牌不能为空\n\n")
		fmt.Printf("💡 如何获取GitHub Token:\n")
		fmt.Printf("   1. 访问: https://github.com/settings/tokens\n")
		fmt.Printf("   2. 点击 'Generate new token (classic)'\n")
		fmt.Printf("   3. 勾选 'gist' 权限\n")
		fmt.Printf("   4. 复制生成的Token\n\n")
		fmt.Printf("📖 详细帮助: codex-mirror sync help\n")
		return fmt.Errorf("GitHub访问令牌不能为空")
	}
	
	if syncEncryptPwd == "" {
		fmt.Printf("❌ 加密密码不能为空\n\n")
		fmt.Printf("💡 密码要求:\n")
		fmt.Printf("   - 长度至少8位\n")
		fmt.Printf("   - 建议包含字母和数字\n")
		fmt.Printf("   - 请妥善保管，忘记密码将无法解密云端数据\n")
		return fmt.Errorf("加密密码不能为空")
	}
	
	if len(syncEncryptPwd) < 8 {
		return fmt.Errorf("加密密码长度至少8位，当前长度: %d", len(syncEncryptPwd))
	}

	// 创建镜像源管理器
	mirrorManager, err := internal.NewMirrorManager()
	if err != nil {
		return fmt.Errorf("创建镜像源管理器失败: %w", err)
	}

	// 创建同步管理器
	syncManager := internal.NewSyncManager(mirrorManager)

	fmt.Printf("🔧 正在初始化云同步...\n")
	fmt.Printf("   提供商: GitHub Gist\n")
	fmt.Printf("   端点: https://api.github.com\n")
	fmt.Printf("   🔐 全量同步: 启用（包含加密的API密钥）\n")
	
	fmt.Printf("\n🛡️  安全说明:\n")
	fmt.Printf("   - 所有数据使用AES-256加密\n")
	fmt.Printf("   - 使用你提供的密码进行加密\n")
	fmt.Printf("   - 存储在私有GitHub Gist中\n")
	fmt.Printf("   - 请妥善保管你的密码和GitHub Token\n")

	// 初始化同步
	if err := syncManager.InitSyncWithPasswordAndGist("gist", "https://api.github.com", syncToken, syncEncryptPwd, syncGistID); err != nil {
		return fmt.Errorf("初始化云同步失败: %w", err)
	}

	fmt.Printf("\n💡 使用提示:\n")
	fmt.Printf("   - 使用 'codex-mirror sync push' 推送配置到云端\n")
	fmt.Printf("   - 使用 'codex-mirror sync pull' 从云端拉取配置\n")
	fmt.Printf("   - 使用 'codex-mirror sync status' 查看同步状态\n")
	fmt.Printf("   - 在其他设备上使用相同的密码初始化同步\n")
	fmt.Printf("   - 查看详细帮助: 'codex-mirror sync help'\n")

	return nil
}

// runSyncPush 执行推送配置.
func runSyncPush(cmd *cobra.Command, args []string) error {
	// 创建镜像源管理器
	mirrorManager, err := internal.NewMirrorManager()
	if err != nil {
		return fmt.Errorf("创建镜像源管理器失败: %w", err)
	}

	// 检查是否已初始化
	if mirrorManager.GetConfig().Sync == nil {
		fmt.Printf("❌ 云同步未初始化\n\n")
		fmt.Printf("💡 请先初始化云同步:\n")
		fmt.Printf("   codex-mirror sync init --token <GitHub-Token> --password <加密密码>\n\n")
		fmt.Printf("📖 详细帮助: codex-mirror sync help\n")
		return fmt.Errorf("云同步未初始化，请先运行 'codex-mirror sync init'")
	}

	// 创建同步管理器
	syncManager := internal.NewSyncManager(mirrorManager)

	// 推送配置（使用策略参数）
	if err := syncManager.PushWithStrategy(pushStrategy); err != nil {
		if strings.Contains(err.Error(), "GitHub API 错误 (401)") {
			fmt.Printf("❌ GitHub认证失败\n\n")
			fmt.Printf("💡 可能的原因:\n")
			fmt.Printf("   - Token无效或已过期\n")
			fmt.Printf("   - Token没有gist权限\n\n")
			fmt.Printf("🔧 解决方法:\n")
			fmt.Printf("   - 重新生成Token: https://github.com/settings/tokens\n")
			fmt.Printf("   - 确保勾选了'gist'权限\n")
			fmt.Printf("   - 使用新Token重新初始化同步\n")
			return fmt.Errorf("GitHub认证失败")
		}
		if strings.Contains(err.Error(), "加密失败") {
			fmt.Printf("❌ 数据加密失败\n\n")
			fmt.Printf("💡 可能的原因:\n")
			fmt.Printf("   - 密码配置异常\n")
			fmt.Printf("   - 系统加密组件故障\n\n")
			fmt.Printf("🔧 解决方法:\n")
			fmt.Printf("   - 重新初始化同步: codex-mirror sync init\n")
			return fmt.Errorf("数据加密失败")
		}
		return fmt.Errorf("推送配置失败: %w", err)
	}

	return nil
}

// runSyncPull 执行拉取配置.
func runSyncPull(cmd *cobra.Command, args []string) error {
	// 创建镜像源管理器
	mirrorManager, err := internal.NewMirrorManager()
	if err != nil {
		return fmt.Errorf("创建镜像源管理器失败: %w", err)
	}

	// 检查是否已初始化
	if mirrorManager.GetConfig().Sync == nil {
		fmt.Printf("❌ 云同步未初始化\n\n")
		fmt.Printf("💡 请先初始化云同步:\n")
		fmt.Printf("   codex-mirror sync init --token <GitHub-Token> --password <加密密码>\n\n")
		fmt.Printf("🔑 如何获取GitHub Token:\n")
		fmt.Printf("   1. 访问: https://github.com/settings/tokens\n")
		fmt.Printf("   2. 点击 'Generate new token (classic)'\n")
		fmt.Printf("   3. 勾选 'gist' 权限\n")
		fmt.Printf("   4. 复制生成的Token\n\n")
		fmt.Printf("📖 详细帮助: codex-mirror sync help\n")
		return fmt.Errorf("云同步未初始化，请先运行 'codex-mirror sync init'")
	}

	// 创建同步管理器
	syncManager := internal.NewSyncManager(mirrorManager)

	// 拉取配置
	if err := syncManager.PullWithStrategy(resolveStrategy); err != nil {
		if strings.Contains(err.Error(), "解密失败") {
			fmt.Printf("❌ 解密失败\n\n")
			fmt.Printf("💡 可能的原因:\n")
			fmt.Printf("   - 密码不正确\n")
			fmt.Printf("   - 云端数据损坏\n")
			fmt.Printf("   - 使用了不同的密码\n\n")
			fmt.Printf("🔧 解决方法:\n")
			fmt.Printf("   - 检查密码是否正确\n")
			fmt.Printf("   - 如果忘记密码，请重新初始化: codex-mirror sync init\n")
			return fmt.Errorf("解密失败，请检查密码是否正确")
		}
		if strings.Contains(err.Error(), "GitHub API 错误 (401)") {
			fmt.Printf("❌ GitHub认证失败\n\n")
			fmt.Printf("💡 可能的原因:\n")
			fmt.Printf("   - Token无效或已过期\n")
			fmt.Printf("   - Token没有gist权限\n\n")
			fmt.Printf("🔧 解决方法:\n")
			fmt.Printf("   - 检查Token是否正确\n")
			fmt.Printf("   - 重新生成Token: https://github.com/settings/tokens\n")
			fmt.Printf("   - 确保勾选了'gist'权限\n")
			return fmt.Errorf("GitHub认证失败")
		}
		if strings.Contains(err.Error(), "未找到文件") {
			fmt.Printf("❌ 云端没有找到配置文件\n\n")
			fmt.Printf("💡 可能的原因:\n")
			fmt.Printf("   - 这是第一次使用云同步\n")
			fmt.Printf("   - 还没有从其他设备推送过配置\n\n")
			fmt.Printf("🔧 解决方法:\n")
			fmt.Printf("   - 先在一台设备上配置镜像源\n")
			fmt.Printf("   - 使用 'codex-mirror sync push' 推送配置\n")
			return fmt.Errorf("云端没有找到配置文件")
		}
		return fmt.Errorf("拉取配置失败: %w", err)
	}

	return nil
}

// runSyncStatus 执行查看同步状态.
func runSyncStatus(cmd *cobra.Command, args []string) error {
	// 创建镜像源管理器
	mirrorManager, err := internal.NewMirrorManager()
	if err != nil {
		return fmt.Errorf("创建镜像源管理器失败: %w", err)
	}

	// 创建同步管理器
	syncManager := internal.NewSyncManager(mirrorManager)

	// 获取同步状态
	status, err := syncManager.GetStatus()
	if err != nil {
		return fmt.Errorf("获取同步状态失败: %w", err)
	}

	// 显示状态信息
	fmt.Printf("云同步状态:\n")
	fmt.Printf("==================================================\n")

	if !status.Enabled {
		fmt.Printf("❌ 云同步未启用\n")
		fmt.Printf("   %s\n", status.Message)
		fmt.Printf("\n💡 使用 'codex-mirror sync init' 初始化云同步\n")
		return nil
	}

	fmt.Printf("✅ 云同步已启用\n")
	fmt.Printf("   提供商: %s\n", status.Provider)
	fmt.Printf("   端点: %s\n", status.Endpoint)
	fmt.Printf("   设备ID: %s\n", status.DeviceID)
	fmt.Printf("   自动同步: %s\n", formatBool(status.AutoSync))
	
	if status.AutoSync {
		fmt.Printf("   同步间隔: %d分钟\n", status.SyncInterval)
	}
	
	fmt.Printf("   %s\n", status.Message)
	
	// 显示加密状态
	if mirrorManager.GetConfig().Sync != nil {
		fmt.Printf("   全量同步: 是（包含加密的API密钥）\n")
	}

	return nil
}

// runSyncConfig 执行配置同步设置.
func runSyncConfig(cmd *cobra.Command, args []string) error {
	// 创建镜像源管理器
	mirrorManager, err := internal.NewMirrorManager()
	if err != nil {
		return fmt.Errorf("创建镜像源管理器失败: %w", err)
	}

	// 检查是否已初始化同步
	if mirrorManager.GetConfig().Sync == nil {
		return fmt.Errorf("云同步未初始化，请先运行 'codex-mirror sync init'")
	}

	config := mirrorManager.GetConfig().Sync

	// 处理禁用同步
	if syncDisable {
		config.Enabled = false
		fmt.Printf("❌ 云同步已禁用\n")
	} else {
		config.Enabled = true
		fmt.Printf("✅ 云同步已启用\n")
	}

	// 更新自动同步设置
	if cmd.Flags().Changed("auto-sync") {
		config.AutoSync = syncAutoSync
		fmt.Printf("   自动同步: %s\n", formatBool(syncAutoSync))
	}

	// 更新同步间隔
	if cmd.Flags().Changed("interval") {
		if syncInterval < 1 {
			return fmt.Errorf("同步间隔必须大于0分钟")
		}
		config.SyncInterval = syncInterval
		fmt.Printf("   同步间隔: %d分钟\n", syncInterval)
	}

	// 更新加密密码
	if cmd.Flags().Changed("password") {
		if syncEncryptPwd == "" {
			return fmt.Errorf("加密密码不能为空")
		}
		if len(syncEncryptPwd) < 8 {
			return fmt.Errorf("加密密码长度至少8位")
		}
		
		fmt.Printf("\n⚠️  更改加密密码:\n")
		fmt.Printf("   - 更改密码后，之前的云端数据将无法解密\n")
		fmt.Printf("   - 建议先备份当前配置\n")
		fmt.Printf("是否继续？(y/N): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Printf("已取消更改\n")
			return nil
		}
		
		config.EncryptionPwd = syncEncryptPwd
		fmt.Printf("   ✅ 加密密码已更新\n")
		fmt.Printf("   💡 请使用 'codex-mirror sync push' 重新上传配置\n")
	}

	// 保存配置
	if err := mirrorManager.SaveConfig(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	fmt.Printf("\n✅ 同步设置已更新\n")

	return nil
}

// formatBool 格式化布尔值显示.
func formatBool(b bool) string {
	if b {
		return "是"
	}
	return "否"
}