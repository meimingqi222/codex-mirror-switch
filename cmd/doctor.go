package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "诊断并修复配置问题",
	Long: `运行健康检查，诊断 codex-mirror 配置问题并提供修复建议。

检查项目：
- 配置文件完整性
- 环境变量一致性
- 镜像源有效性
- VS Code / Codex 配置状态

示例：
  codex-mirror doctor           # 运行所有检查
  codex-mirror doctor --verbose # 详细输出`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		skipTest, _ := cmd.Flags().GetBool("skip-test")

		return runDoctor(verbose, skipTest)
	},
}

// CheckResult 健康检查结果
type CheckResult struct {
	Name        string
	Description string
	Status      string // "ok", "warning", "error", "skipped"
	Message     string
	Fix         string
}

// HealthCheckFunc 健康检查函数类型
type HealthCheckFunc func(verbose bool) CheckResult

func init() {
	doctorCmd.Flags().Bool("verbose", false, "显示详细输出")
	doctorCmd.Flags().Bool("skip-test", false, "跳过镜像源连通性测试")
	rootCmd.AddCommand(doctorCmd)
}

// runDoctor 运行健康检查
func runDoctor(verbose, skipTest bool) error {
	fmt.Println("🔍 正在运行健康检查...")
	fmt.Println()

	checks := []HealthCheckFunc{
		checkConfigFile,
		checkEnvironmentVariables,
		checkVSCodeConfig,
		checkCodexConfig,
	}

	if !skipTest {
		checks = append(checks, checkMirrorConnectivity)
	}

	var results []CheckResult
	hasError := false
	hasWarning := false

	for i, check := range checks {
		result := check(verbose)
		results = append(results, result)

		fmt.Printf("[%d/%d] %s\n", i+1, len(checks), result.Name)

		switch result.Status {
		case "ok":
			fmt.Printf("    ✅ %s\n", result.Message)
		case "warning":
			fmt.Printf("    ⚠️  %s\n", result.Message)
			if result.Fix != "" {
				fmt.Printf("    💡 建议: %s\n", result.Fix)
			}
			hasWarning = true
		case "error":
			fmt.Printf("    ❌ %s\n", result.Message)
			if result.Fix != "" {
				fmt.Printf("    🔧 修复: %s\n", result.Fix)
			}
			hasError = true
		case "skipped":
			fmt.Printf("    ⏭️  %s\n", result.Message)
		}
		fmt.Println()
	}

	// 汇总
	fmt.Println("📊 检查结果汇总:")
	errorCount := 0
	warningCount := 0
	okCount := 0
	for _, r := range results {
		switch r.Status {
		case "ok":
			okCount++
		case "warning":
			warningCount++
		case "error":
			errorCount++
		}
	}

	fmt.Printf("    ✅ 正常: %d\n", okCount)
	fmt.Printf("    ⚠️  警告: %d\n", warningCount)
	fmt.Printf("    ❌ 错误: %d\n", errorCount)
	fmt.Println()

	if hasError {
		fmt.Println("❌ 发现错误，请根据上述建议修复")
	} else if hasWarning {
		fmt.Println("⚠️  发现警告，建议进行优化")
	} else {
		fmt.Println("✅ 所有检查通过！")
	}

	if hasError {
		return fmt.Errorf("健康检查未通过")
	}
	return nil
}

// checkConfigFile 检查配置文件完整性
func checkConfigFile(verbose bool) CheckResult {
	mm, err := internal.NewMirrorManager()
	if err != nil {
		return CheckResult{
			Name:        "配置文件检查",
			Description: "检查 mirrors.toml 是否存在",
			Status:      "error",
			Message:     fmt.Sprintf("无法加载配置: %v", err),
			Fix:         "运行 'codex-mirror add <name> <url> <api-key>' 添加镜像源",
		}
	}

	config := mm.GetConfig()
	activeMirrors := mm.ListActiveMirrors()

	if len(activeMirrors) == 0 {
		return CheckResult{
			Name:        "配置文件检查",
			Description: "检查镜像源数量",
			Status:      "warning",
			Message:     "未配置任何镜像源",
			Fix:         "运行 'codex-mirror add <name> <url> <api-key>' 添加镜像源",
		}
	}

	// 检查当前激活的镜像是否存在
	currentClaude := config.CurrentClaude
	currentCodex := config.CurrentCodex

	if currentClaude != "" {
		_, err := mm.GetCurrentClaudeMirror()
		if err != nil {
			return CheckResult{
				Name:        "配置文件检查",
				Description: "检查当前 Claude 镜像",
				Status:      "warning",
				Message:     fmt.Sprintf("当前 Claude 镜像 '%s' 不存在", currentClaude),
				Fix:         fmt.Sprintf("运行 'codex-mirror switch <name>' 切换到其他镜像"),
			}
		}
	}

	if currentCodex != "" {
		_, err := mm.GetCurrentCodexMirror()
		if err != nil {
			return CheckResult{
				Name:        "配置文件检查",
				Description: "检查当前 Codex 镜像",
				Status:      "warning",
				Message:     fmt.Sprintf("当前 Codex 镜像 '%s' 不存在", currentCodex),
				Fix:         fmt.Sprintf("运行 'codex-mirror switch <name>' 切换到其他镜像"),
			}
		}
	}

	return CheckResult{
		Name:        "配置文件检查",
		Description: "检查配置文件完整性",
		Status:      "ok",
		Message:     fmt.Sprintf("配置文件正常 (共 %d 个镜像源)", len(activeMirrors)),
	}
}

// checkEnvironmentVariables 检查环境变量一致性
func checkEnvironmentVariables(verbose bool) CheckResult {
	mm, err := internal.NewMirrorManager()
	if err != nil {
		return CheckResult{
			Name:        "环境变量检查",
			Description: "加载配置失败",
			Status:      "error",
			Message:     fmt.Sprintf("无法加载配置: %v", err),
		}
	}

	config := mm.GetConfig()
	var warnings []string

	// 检查 Claude 环境变量
	if config.CurrentClaude != "" {
		mirror, err := mm.GetCurrentClaudeMirror()
		if err == nil {
			envBaseURL := os.Getenv("ANTHROPIC_BASE_URL")
			envToken := os.Getenv("ANTHROPIC_AUTH_TOKEN")

			if envBaseURL != "" && envBaseURL != mirror.BaseURL {
				warnings = append(warnings, fmt.Sprintf("ANTHROPIC_BASE_URL 与配置不一致 (环境: %s, 配置: %s)", envBaseURL, mirror.BaseURL))
			}
			if envToken != "" && envToken != mirror.APIKey {
				warnings = append(warnings, "ANTHROPIC_AUTH_TOKEN 与配置不一致")
			}
		}
	}

	// 检查 Codex 环境变量
	if config.CurrentCodex != "" {
		mirror, err := mm.GetCurrentCodexMirror()
		if err == nil {
			envKey := os.Getenv(internal.CodexSwitchAPIKeyEnv)
			if envKey != "" && envKey != mirror.APIKey {
				warnings = append(warnings, fmt.Sprintf("%s 与配置不一致", internal.CodexSwitchAPIKeyEnv))
			}
		}
	}

	if len(warnings) > 0 {
		msg := "环境变量与配置文件不一致"
		if verbose {
			msg += ":\n"
			for _, w := range warnings {
				msg += fmt.Sprintf("    - %s\n", w)
			}
		}
		return CheckResult{
			Name:        "环境变量检查",
			Description: "检查环境变量与配置一致性",
			Status:      "warning",
			Message:     msg,
			Fix:         "运行 'codex-mirror switch <name>' 重新应用配置",
		}
	}

	return CheckResult{
		Name:        "环境变量检查",
		Description: "检查环境变量状态",
		Status:      "ok",
		Message:     "环境变量与配置一致",
	}
}

// checkVSCodeConfig 检查 VS Code 配置
func checkVSCodeConfig(verbose bool) CheckResult {
	platform := internal.GetCurrentPlatform()

	var settingsPath string
	switch platform {
	case internal.PlatformWindows:
		settingsPath = filepath.Join(os.Getenv("APPDATA"), "Code", "User", "settings.json")
	case internal.PlatformMac:
		settingsPath = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Code", "User", "settings.json")
	default:
		settingsPath = filepath.Join(os.Getenv("HOME"), ".config", "Code", "User", "settings.json")
	}

	// 检查文件是否存在
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return CheckResult{
			Name:        "VS Code 配置检查",
			Description: "检查 settings.json 是否存在",
			Status:      "skipped",
			Message:     "VS Code 配置文件不存在 (可能未安装 VS Code)",
		}
	}

	// 尝试加载配置
	vscodeMgr, err := internal.NewVSCodeConfigManager()
	if err != nil {
		return CheckResult{
			Name:        "VS Code 配置检查",
			Description: "加载 VS Code 配置",
			Status:      "warning",
			Message:     fmt.Sprintf("无法加载 VS Code 配置: %v", err),
		}
	}

	settings, err := vscodeMgr.LoadSettings()
	if err != nil {
		return CheckResult{
			Name:        "VS Code 配置检查",
			Description: "解析 VS Code 配置",
			Status:      "warning",
			Message:     fmt.Sprintf("无法解析 VS Code 配置: %v", err),
		}
	}

	// 检查 chatgpt.apiBase 类型
	apiBase, exists := settings["chatgpt.apiBase"]
	s, ok := apiBase.(string)
	if !exists || !ok || strings.TrimSpace(s) == "" {
		return CheckResult{
			Name:        "VS Code 配置检查",
			Description: "检查 chatgpt.apiBase 设置",
			Status:      "warning",
			Message:     "未配置 chatgpt.apiBase 或类型错误",
			Fix:         "运行 'codex-mirror switch <codex-mirror>' 应用 VS Code 配置",
		}
	}

	return CheckResult{
		Name:        "VS Code 配置检查",
		Description: "检查 VS Code 配置状态",
		Status:      "ok",
		Message:     fmt.Sprintf("chatgpt.apiBase: %v", apiBase),
	}
}

// checkCodexConfig 检查 Codex CLI 配置
func checkCodexConfig(verbose bool) CheckResult {
	mm, err := internal.NewMirrorManager()
	if err != nil {
		return CheckResult{
			Name:        "Codex CLI 配置检查",
			Description: "加载配置失败",
			Status:      "error",
			Message:     fmt.Sprintf("无法加载配置: %v", err),
		}
	}

	config := mm.GetConfig()
	if config.CurrentCodex == "" {
		return CheckResult{
			Name:        "Codex CLI 配置检查",
			Description: "检查当前 Codex 镜像",
			Status:      "warning",
			Message:     "未设置当前 Codex 镜像源",
			Fix:         "运行 'codex-mirror switch <codex-mirror>' 设置",
		}
	}

	mirror, err := mm.GetCurrentCodexMirror()
	if err != nil {
		return CheckResult{
			Name:        "Codex CLI 配置检查",
			Description: "检查当前 Codex 镜像是否存在",
			Status:      "error",
			Message:     fmt.Sprintf("当前 Codex 镜像 '%s' 不存在", config.CurrentCodex),
		}
	}

	// 尝试加载 Codex 配置
	codexMgr, err := internal.NewCodexConfigManager()
	if err != nil {
		return CheckResult{
			Name:        "Codex CLI 配置检查",
			Description: "加载 Codex 配置",
			Status:      "warning",
			Message:     fmt.Sprintf("无法加载 Codex 配置: %v", err),
		}
	}

	codexConfig, err := codexMgr.GetCurrentConfig()
	if err != nil {
		return CheckResult{
			Name:        "Codex CLI 配置检查",
			Description: "解析 Codex 配置",
			Status:      "warning",
			Message:     fmt.Sprintf("无法解析 Codex 配置: %v", err),
		}
	}

	// 检查配置是否匹配
	if codexConfig.ModelProvider == "" && len(codexConfig.ModelProviders) == 0 {
		return CheckResult{
			Name:        "Codex CLI 配置检查",
			Description: "检查 Codex 模型配置",
			Status:      "warning",
			Message:     "Codex 配置中未找到模型提供商",
			Fix:         "运行 'codex-mirror switch <codex-mirror>' 重新应用配置",
		}
	}

	return CheckResult{
		Name:        "Codex CLI 配置检查",
		Description: "检查 Codex CLI 配置状态",
		Status:      "ok",
		Message:     fmt.Sprintf("当前镜像: %s (%s)", config.CurrentCodex, mirror.BaseURL),
	}
}

// checkMirrorConnectivity 检查镜像源连通性
func checkMirrorConnectivity(verbose bool) CheckResult {
	mm, err := internal.NewMirrorManager()
	if err != nil {
		return CheckResult{
			Name:        "镜像源连通性检查",
			Description: "加载配置失败",
			Status:      "error",
			Message:     fmt.Sprintf("无法加载配置: %v", err),
		}
	}

	mirrors := mm.ListActiveMirrors()
	if len(mirrors) == 0 {
		return CheckResult{
			Name:        "镜像源连通性检查",
			Description: "无镜像源可测试",
			Status:      "skipped",
			Message:     "未配置任何镜像源",
		}
	}

	fmt.Println("    测试镜像源连通性...")

	// 使用 test 命令的测试函数
	results := GetTestResultsFromAll(mm, 10)

	var okMirrors []string
	var errorMirrors []string
	var skippedMirrors int

	for _, r := range results {
		if r.Success {
			okMirrors = append(okMirrors, r.Name)
		} else if r.Error == "需要 API Key (401)" {
			skippedMirrors++
		} else {
			errorMirrors = append(errorMirrors, r.Name)
		}
	}

	if verbose {
		fmt.Println("    正常:")
		for _, m := range okMirrors {
			fmt.Printf("      ✅ %s\n", m)
		}
		if len(errorMirrors) > 0 {
			fmt.Println("    异常:")
			for _, m := range errorMirrors {
				fmt.Printf("      ❌ %s\n", m)
			}
		}
	}

	if len(errorMirrors) > 0 {
		return CheckResult{
			Name:        "镜像源连通性检查",
			Description: "检查所有镜像源状态",
			Status:      "warning",
			Message:     fmt.Sprintf("正常: %d, 异常: %d, 跳过: %d", len(okMirrors), len(errorMirrors), skippedMirrors),
			Fix:         "运行 'codex-mirror test --remove-invalid' 清理无效镜像源",
		}
	}

	if len(okMirrors) == 0 && skippedMirrors > 0 {
		return CheckResult{
			Name:        "镜像源连通性检查",
			Description: "所有镜像源都缺少 API Key",
			Status:      "warning",
			Message:     "所有镜像源都缺少 API Key，请配置有效的 Key",
		}
	}

	return CheckResult{
		Name:        "镜像源连通性检查",
		Description: "检查镜像源连通性",
		Status:      "ok",
		Message:     fmt.Sprintf("正常: %d, 异常: %d", len(okMirrors), len(errorMirrors)),
	}
}
