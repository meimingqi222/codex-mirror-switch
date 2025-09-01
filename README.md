# Codex Mirror Switch

一个用于管理和切换 Codex CLI 和 VS Code 插件镜像源的命令行工具。

## 功能特性

- 🔄 **镜像源管理**: 添加、删除、列出镜像源配置
- 🚀 **一键切换**: 快速切换不同的 API 镜像源
- 🔧 **自动配置**: 自动更新 Codex CLI 和 VS Code 配置文件
- 🌍 **跨平台支持**: 支持 Windows、macOS、Linux
- 🔐 **环境变量管理**: 自动设置对应的 API 密钥环境变量
- 💾 **配置备份**: 切换前自动备份原有配置
- 📊 **状态查看**: 查看当前使用的镜像源状态

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

### 使用示例

#### 1. 添加镜像源

```bash
# 添加官方 OpenAI API
codex-mirror add official https://api.openai.com sk-your-api-key

# 添加本地代理
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
* official    https://api.openai.com           sk-12****7890
  local       http://localhost:8080            (无API密钥)
  mirror      https://api.example.com          sk-mi****key

当前使用: official
```

#### 3. 切换镜像源

```bash
# 切换到指定镜像源
codex-mirror switch mirror

# 只更新 Codex CLI 配置
codex-mirror switch mirror --codex-only

# 只更新 VS Code 配置
codex-mirror switch mirror --vscode-only

# 切换时不备份原配置
codex-mirror switch mirror --no-backup
```

#### 4. 查看当前状态

```bash
codex-mirror status
```

#### 5. 删除镜像源

```bash
codex-mirror remove mirror
```

## 配置文件

### 镜像源配置

配置文件位置：`~/.codex-mirror/mirrors.toml`

```toml
current_mirror = "official"

[[mirrors]]
name = "official"
base_url = "https://api.openai.com"
api_key = "sk-your-api-key"

[[mirrors]]
name = "local"
base_url = "http://localhost:8080"
api_key = ""
```

### Codex CLI 配置

- 配置文件：`~/.codex/config.toml`
- 认证文件：`~/.codex/auth.json`

### VS Code 配置

- Windows: `%APPDATA%\Code\User\settings.json`
- macOS: `~/Library/Application Support/Code/User/settings.json`
- Linux: `~/.config/Code/User/settings.json`

## 环境变量

切换镜像源时，工具会自动设置对应的环境变量：

- 格式：`CODEX_<镜像源名称>_API_KEY`
- 示例：
  - `CODEX_OFFICIAL_API_KEY`
  - `CODEX_LOCAL_API_KEY`
  - `CODEX_MIRROR_API_KEY`

## 命令行选项

### 全局选项

- `--help, -h`: 显示帮助信息

### switch 命令选项

- `--codex-only`: 只更新 Codex CLI 配置
- `--vscode-only`: 只更新 VS Code 配置
- `--no-backup`: 切换时不备份原配置

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

- Go 1.24.4 或更高版本

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

---

如有问题或建议，请提交 [Issue](https://github.com/your-username/codex-mirror-switch/issues)。