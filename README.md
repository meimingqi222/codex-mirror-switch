# Codex Mirror Switch

一个用于管理和切换 Claude Code、Codex CLI 和 VS Code 插件镜像源的命令行工具。

## 📚 文档

- [开发指南](docs/development-guide.md) - 项目架构、开发流程和最佳实践
- [贡献指南](#贡献) - 如何为项目做出贡献

## 功能特性

- 🔄 **镜像源管理**: 添加、删除、列出镜像源配置
- 🚀 **一键切换**: 快速切换不同的 API 镜像源
- 🔧 **自动配置**: 自动更新 Claude Code、Codex CLI 和 VS Code 配置
- 🌍 **跨平台支持**: 支持 Windows、macOS、Linux
- 🔐 **环境变量管理**: 自动设置对应的 API 密钥环境变量
- 💾 **配置备份**: 切换前自动备份原有配置
- 📊 **状态查看**: 查看当前使用的镜像源状态
- 🎯 **工具类型支持**: 支持 `claude` 和 `codex` 两种工具类型

## 安装

### 从源码构建

```bash
git clone https://github.com/your-username/codex-mirror-switch.git
cd codex-mirror-switch
go build -o codex-mirror main.go
```

### 直接下载

从 [Releases](https://github.com/your-username/codex-mirror-switch/releases) 页面下载对应平台的可执行文件。

## 使用方法

### 基本命令

```bash
# 查看帮助
codex-mirror --help

# 添加镜像源
codex-mirror add <名称> <API地址> [API密钥]

# 列出所有镜像源
codex-mirror list

# 切换镜像源
codex-mirror switch <名称>

# 查看当前状态
codex-mirror status

# 删除镜像源
codex-mirror remove <名称>
```

### 工具类型支持

工具支持两种镜像源类型：

**Claude Code 类型 (`claude`)**：
- 只设置环境变量：`ANTHROPIC_BASE_URL` 和 `ANTHROPIC_AUTH_TOKEN`
- 不修改配置文件
- 适用于 Claude Code (claude.ai/code)

**Codex CLI 类型 (`codex`)**：
- 修改配置文件：`~/.codex/config.toml` 和 `~/.codex/auth.json`
- 更新 VS Code 配置：`settings.json`
- 设置环境变量：`CODEX_SWITCH_OPENAI_API_KEY`
- 适用于 Codex CLI 和相关 VS Code 插件

### 使用示例

#### 1. 添加镜像源

```bash
# 添加 Claude Code 官方 API
codex-mirror add claude-official https://api.anthropic.com sk-ant-api-key --type claude

# 添加 Codex CLI 官方 API
codex-mirror add codex-official https://api.openai.com sk-openai-key --type codex

# 添加本地代理 (默认为 codex 类型)
codex-mirror add local http://localhost:8080

# 添加第三方镜像
codex-mirror add mirror https://api.example.com sk-mirror-key
```

#### 2. 查看镜像源列表

```bash
codex-mirror list
```

输出示例：
```
可用镜像源：
* claude-official    https://api.anthropic.com      sk-an****key      (claude)
* codex-official     https://api.openai.com         sk-op****key      (codex)
  local              http://localhost:8080          (无API密钥)       (codex)
  mirror             https://api.example.com         sk-mi****key      (codex)

当前使用: claude-official
```

#### 3. 切换镜像源

```bash
# 切换到 Claude Code 配置
codex-mirror switch claude-official

# 切换到 Codex CLI 配置
codex-mirror switch codex-official

# 只更新 Codex CLI 配置 (仅对 codex 类型有效)
codex-mirror switch codex-official --codex-only

# 只更新 VS Code 配置 (仅对 codex 类型有效)
codex-mirror switch codex-official --vscode-only

# 切换时不备份原配置
codex-mirror switch claude-official --no-backup

# 即时刷新当前终端环境变量

# bash/zsh（推荐）：
eval "$(codex-mirror switch claude-official --shell bash)"

# fish：
codex-mirror switch claude-official --shell fish | source

# PowerShell：
codex-mirror switch claude-official --shell powershell | iex
```

#### 6. 安装/使用 shell 集成（推荐）

安装后，`codex-mirror switch <name>` 将自动：
- 正常更新配置/持久化（输出到 stderr），以及
- 自动评估导出语句，让当前会话立即生效。

安装命令：

```bash
# 自动检测当前 shell 并安装包装函数（Windows 上：PowerShell；若在 Git Bash/MSYS 中运行，会自动识别 bash）
codex-mirror init

# 或显式指定要安装的 shell：
codex-mirror init --shell bash
codex-mirror init --shell zsh
codex-mirror init --shell fish
codex-mirror init --shell powershell
```

加载生效：
- bash/zsh：`source ~/.bashrc` 或 `source ~/.zshrc`，或重新打开终端
- fish：自动加载 `~/.config/fish/functions/codex-mirror.fish`
- PowerShell：重启 PowerShell；或执行 `. "$PROFILE"`

卸载：删除配置文件中标记块（或删除 fish 函数文件）：
```
# >>> codex-mirror init >>>
...
# <<< codex-mirror init <<<
```

Windows 说明：
- PowerShell 集成优先写入：
  - `%USERPROFILE%\Documents\PowerShell\Microsoft.PowerShell_profile.ps1`（PowerShell 7+）
  - 若目录不存在，尝试 `%USERPROFILE%\Documents\WindowsPowerShell\Microsoft.PowerShell_profile.ps1`（Windows PowerShell 5.1）
  - 若存在 OneDrive 重定向，会尝试 `%OneDrive%\Documents\...` 路径
- CMD 不支持透明集成，推荐使用 PowerShell。
 - 若在 Windows 的 Git Bash/MSYS 环境运行 `codex-mirror init`，将自动写入 `~/.bashrc`（或现存的 `~/.bash_profile`）。

#### 4. 查看当前状态

```bash
codex-mirror status
```

状态输出示例：
```
当前配置状态:
==================================================
Claude Code配置:
  当前配置: claude-official
  API端点: https://api.anthropic.com
  环境变量 ANTHROPIC_BASE_URL: ✓ 正确
  环境变量 ANTHROPIC_AUTH_TOKEN: ✓ 正确

Codex CLI配置:
  当前配置: codex-official
  API端点: https://api.openai.com
  配置文件 (~/.codex/config.toml): ✓ 正确
  认证文件 (~/.codex/auth.json): ✓ 正确
  环境变量 CODEX_SWITCH_OPENAI_API_KEY: ✓ 正确

VS Code配置:
  ✓ 配置正确 (chatgpt.apiBase: https://api.openai.com)
```

#### 5. 删除镜像源

```bash
codex-mirror remove mirror
```

## 配置文件

### 镜像源配置

配置文件位置：`~/.codex-mirror/mirrors.toml`

```toml
current_codex = "codex-official"
current_claude = "claude-official"

[[mirrors]]
name = "claude-official"
base_url = "https://api.anthropic.com"
api_key = "sk-ant-api-key"
env_key = ""
tool_type = "claude"

[[mirrors]]
name = "codex-official"
base_url = "https://api.openai.com"
api_key = "sk-openai-key"
env_key = "CODEX_SWITCH_OPENAI_API_KEY"
tool_type = "codex"

[[mirrors]]
name = "local"
base_url = "http://localhost:8080"
api_key = ""
env_key = ""
tool_type = "codex"
```

### Codex CLI 配置

- 配置文件：`~/.codex/config.toml`
- 认证文件：`~/.codex/auth.json`

### VS Code 配置

- Windows: `%APPDATA%\Code\User\settings.json`
- macOS: `~/Library/Application Support/Code/User/settings.json`
- Linux: `~/.config/Code/User/settings.json`

## 环境变量

### Claude Code 环境变量

当切换到 `claude` 类型的镜像源时，工具会设置：
- `ANTHROPIC_BASE_URL`: API 基础地址
- `ANTHROPIC_AUTH_TOKEN`: Claude API 认证令牌

### Codex CLI 环境变量

当切换到 `codex` 类型的镜像源时，工具会设置：
- `CODEX_SWITCH_OPENAI_API_KEY`: Codex CLI 专用的 API 密钥环境变量

### 持久化机制

为了确保环境变量在重启后仍然有效，工具在不同平台使用不同的持久化方式：

**Windows:**
- 使用 `setx` 命令设置用户级环境变量
- 环境变量将永久存储在注册表中

**macOS:**
- 自动写入 `~/.zshrc` 和 `~/.bash_profile` 文件
- 支持 zsh（默认）和 bash shell

**Linux:**
- 自动写入 `~/.bashrc` 和 `~/.profile` 文件
- 支持大多数常见的 shell 环境

> **注意:** 在 macOS 和 Linux 上，需要重新启动终端或执行 `source ~/.bashrc`（或对应的配置文件）才能使环境变量生效。

## 命令行选项

### 全局选项

- `--help, -h`: 显示帮助信息

### add 命令选项

- `--type, -t`: 工具类型 (codex|claude, 默认: codex)

### switch 命令选项

- `--codex-only`: 只更新 Codex CLI 配置 (仅对 codex 类型有效)
- `--vscode-only`: 只更新 VS Code 配置 (仅对 codex 类型有效)
- `--no-backup`: 切换时不备份原配置
- `--shell`: 输出适配当前 shell 的导出语句 (bash|zsh|fish|powershell|cmd)，可配合 `eval`/`source`/`iex` 实现当前会话即时生效

## 项目结构

```
codex-mirror-switch/
├── cmd/                    # 命令行命令
│   ├── add.go             # 添加镜像源命令
│   ├── list.go            # 列出镜像源命令
│   ├── remove.go          # 删除镜像源命令
│   ├── root.go            # 根命令
│   ├── status.go          # 状态查看命令
│   └── switch.go          # 切换镜像源命令
├── internal/              # 内部包
│   ├── codex.go          # Codex CLI 配置管理
│   ├── env.go            # 环境变量管理
│   ├── mirror.go         # 镜像源管理
│   ├── platform.go       # 平台相关功能
│   ├── types.go          # 类型定义
│   └── vscode.go         # VS Code 配置管理
├── main.go               # 程序入口
├── go.mod                # Go 模块文件
└── README.md             # 项目说明
```

## 依赖项

- [cobra](https://github.com/spf13/cobra) - 命令行界面框架
- [toml](https://github.com/BurntSushi/toml) - TOML 配置文件解析

## 开发

### 环境要求

- Go 1.23.12 或更高版本（为了与 golangci-lint 兼容）

### 构建

```bash
# 开发构建
go build -o codex-mirror main.go

# 交叉编译
# Windows
GOOS=windows GOARCH=amd64 go build -o codex-mirror.exe main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o codex-mirror-darwin main.go

# Linux
GOOS=linux GOARCH=amd64 go build -o codex-mirror-linux main.go
```

### 测试

```bash
go test ./...
```

## 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 更新日志

### v1.1.0

- ✨ 新增 Claude Code 支持
- 🎯 支持两种工具类型：`claude` 和 `codex`
- 🔧 改进的环境变量管理
- 📊 增强的状态显示功能
- 🛠️ 新增环境变量管理模块

### v1.0.0

- ✨ 初始版本发布
- 🔄 支持镜像源的添加、删除、列出、切换
- 🔧 自动更新 Codex CLI 和 VS Code 配置
- 🔐 自动设置环境变量
- 💾 配置文件备份功能
- 🌍 跨平台支持

## 常见问题

### Q: 如何恢复到默认配置？

A: 可以切换到 `official` 镜像源，或者删除配置目录重新初始化。

### Q: 支持哪些平台？

A: 支持 Windows、macOS 和 Linux。

### Q: 配置文件在哪里？

A: 镜像源配置在 `~/.codex-mirror/mirrors.toml`，备份文件在 `~/.codex-mirror/backup/` 目录。

### Q: 如何添加不需要 API 密钥的镜像源？

A: 使用 `codex-mirror add <名称> <URL>` 命令，不提供第三个参数即可。

### Q: Claude Code 和 Codex CLI 有什么区别？

A: 
- **Claude Code**: 只设置环境变量，不修改配置文件，适用于 claude.ai/code
- **Codex CLI**: 修改配置文件和环境变量，适用于 Codex CLI 和相关 VS Code 插件

### Q: 如何查看当前使用的工具类型？

A: 使用 `codex-mirror list` 查看所有镜像源，输出会显示每个镜像源的工具类型。使用 `codex-mirror status` 查看当前配置状态。

---

如有问题或建议，请提交 [Issue](https://github.com/your-username/codex-mirror-switch/issues)。
# 安装/卸载 shell 集成（让 switch 立即生效）
codex-mirror init                 # 自动检测当前 shell
codex-mirror init --shell zsh     # 手动指定 shell
codex-mirror uninit               # 卸载（自动检测）
codex-mirror uninit --shell zsh   # 指定 shell 卸载
