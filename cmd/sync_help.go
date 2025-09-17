package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// syncHelpCmd 同步帮助命令.
var syncHelpCmd = &cobra.Command{
	Use:   "help",
	Short: "云同步帮助指南",
	Long:  `显示云同步功能的详细使用指南，包括GitHub Token获取方法`,
	RunE:  runSyncHelp,
}

func init() {
	// 将 help 命令添加到 sync 命令
	syncCmd.AddCommand(syncHelpCmd)
}

// runSyncHelp 显示同步帮助信息.
func runSyncHelp(cmd *cobra.Command, args []string) error {
	fmt.Printf(`
🔧 Codex Mirror 云同步帮助指南
==================================================

📋 功能概述:
   云同步功能可以将你的镜像源配置（包括API密钥）安全地同步到多台设备。
   所有数据使用AES-256加密，存储在私有GitHub Gist中。

🔑 GitHub Token 获取步骤:
   1. 访问 GitHub 设置页面:
      https://github.com/settings/tokens

   2. 点击 "Generate new token" → "Generate new token (classic)"

   3. 设置Token信息:
      - Note: 填写 "Codex Mirror Sync" 或其他描述
      - Expiration: 建议选择 "90 days" 或 "1 year"
      - Scopes: 只需勾选 "gist" 权限

   4. 点击 "Generate token"

   5. 复制生成的Token (格式: ghp_xxxxxxxxxxxx)
      ⚠️  Token只显示一次，请立即保存！

🚀 快速开始:

   第一台设备:
   ┌─────────────────────────────────────────────────────────────┐
   │ # 1. 添加镜像源                                              │
   │ codex-mirror add openai https://api.openai.com sk-proj-xxx │
   │ codex-mirror add claude https://api.anthropic.com sk-ant-xxx --type claude │
   │                                                             │
   │ # 2. 初始化云同步                                            │
   │ codex-mirror sync init --token ghp_xxx --password MySecret123 │
   │                                                             │
   │ # 3. 推送到云端                                              │
   │ codex-mirror sync push                                      │
   └─────────────────────────────────────────────────────────────┘

   其他设备 (自动发现):
   ┌─────────────────────────────────────────────────────────────┐
   │ # 1. 初始化同步（使用相同密码，自动发现现有配置）              │
   │ codex-mirror sync init --token ghp_xxx --password MySecret123 │
   │    💡 系统会自动搜索并连接到最新的配置 Gist                  │
   │                                                             │
   │ # 2. 拉取配置                                                │
   │ codex-mirror sync pull                                      │
   │                                                             │
   │ # 完成！所有配置已恢复                                        │
   └─────────────────────────────────────────────────────────────┘

   其他设备 (手动指定):
   ┌─────────────────────────────────────────────────────────────┐
   │ # 1. 使用已知的Gist ID初始化                                 │
   │ codex-mirror sync init --token ghp_xxx --password MySecret123 --gist-id abc123 │
   │                                                             │
   │ # 2. 拉取配置                                                │
   │ codex-mirror sync pull                                      │
   └─────────────────────────────────────────────────────────────┘

📖 常用命令:

   初始化同步:
   codex-mirror sync init --token <GitHub-Token> --password <加密密码>
   
   连接现有配置:
   codex-mirror sync init --token <GitHub-Token> --password <加密密码> --gist-id <Gist-ID>

   推送配置:
   codex-mirror sync push

   拉取配置:
   codex-mirror sync pull

   查看状态:
   codex-mirror sync status

   启用自动同步:
   codex-mirror sync config --auto-sync --interval 30

   更改密码:
   codex-mirror sync config --password <新密码>

🛡️  安全说明:

   ✅ 数据加密: 使用AES-256加密算法
   ✅ 私有存储: 存储在私有GitHub Gist中
   ✅ 密码保护: 只有你的密码能解密数据
   ✅ 传输安全: 全程HTTPS加密传输

   ⚠️  重要提醒:
   - 请妥善保管GitHub Token和加密密码
   - 建议启用GitHub两步验证(2FA)
   - 如果忘记密码，云端数据将无法解密
   - 定期更换GitHub Token以提高安全性

🔧 故障排除:

   问题: "GitHub API 错误 (401)"
   解决: 检查Token是否正确，是否有gist权限

   问题: "解密失败"
   解决: 检查密码是否正确，或重新初始化同步

   问题: "未找到配置文件"
   解决: 先在一台设备上push配置到云端

   问题: Token过期
   解决: 重新生成Token，使用 sync config 更新

💡 最佳实践:

   1. 使用强密码（至少8位，包含字母数字）
   2. 定期备份重要的API密钥
   3. 在多台设备间保持同步
   4. 定期检查同步状态
   5. 及时更新过期的Token
   6. 新设备初始化时会自动发现现有配置
   7. 如果存在多个配置，自动选择最新更新的
   8. 如果自动发现失败，可手动指定Gist ID

📞 获取帮助:

   查看命令帮助: codex-mirror sync <command> --help
   查看项目文档: 访问项目GitHub页面
   报告问题: 在GitHub Issues中提交

==================================================
`)

	return nil
}
