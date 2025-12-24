package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

// needAPIKey401Msg 需要API Key的错误消息.
const needAPIKey401Msg = "需要 API Key (401)"

// testCmd represents the test command.
var testCmd = &cobra.Command{
	Use:   "test [mirror-name]",
	Short: "测试镜像源连通性和 API Key 有效性",
	Long: `测试镜像源的连通性和 API Key 是否有效。

支持测试类型：
- OpenAI 兼容 API：测试 /v1/models 端点
- Anthropic API：测试 /v1/messages 端点

示例：
  codex-mirror test                    # 测试当前镜像源
  codex-mirror test mymirror           # 测试指定镜像源
  codex-mirror test --all              # 测试所有镜像源
  codex-mirror test --all --parallel   # 并行测试所有镜像源
  codex-mirror test --remove-invalid   # 测试并移除无效的 API Key`,
	Aliases: []string{"check", "verify"},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			mm, err := internal.NewMirrorManager()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			mirrors := mm.ListActiveMirrors()
			names := make([]string, 0, len(mirrors))
			for _, m := range mirrors {
				names = append(names, m.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		allMirrors, _ := cmd.Flags().GetBool("all")
		parallel, _ := cmd.Flags().GetBool("parallel")
		timeout, _ := cmd.Flags().GetInt("timeout")
		removeInvalid, _ := cmd.Flags().GetBool("remove-invalid")
		removeAllInvalid, _ := cmd.Flags().GetBool("remove-all-invalid")

		mm, err := internal.NewMirrorManager()
		if err != nil {
			return fmt.Errorf("无法创建镜像管理器: %v", err)
		}

		// 如果指定了移除无效 key 的选项
		if removeInvalid || removeAllInvalid {
			return testAndRemoveInvalidKeys(mm, allMirrors, removeAllInvalid, timeout)
		}

		// 如果没有指定镜像名且没有 --all，测试当前激活的镜像
		if len(args) == 0 && !allMirrors {
			var currentMirror *internal.MirrorConfig
			switch {
			case mm.GetConfig().CurrentClaude != "":
				currentMirror, _ = mm.GetCurrentClaudeMirror()
			default:
				currentMirror, _ = mm.GetCurrentCodexMirror()
			}
			if currentMirror != nil {
				return testMirror(mm, currentMirror, timeout)
			}
			return fmt.Errorf("未找到当前激活的镜像源，请使用 'codex-mirror switch' 先切换")
		}

		// 测试所有镜像源
		if allMirrors {
			return testAllMirrors(mm, parallel, timeout)
		}

		// 测试指定镜像源
		mirror, err := mm.GetMirrorByName(args[0])
		if err != nil {
			return fmt.Errorf("镜像源 '%s' 不存在", args[0])
		}
		return testMirror(mm, mirror, timeout)
	},
}

// TestResult 测试结果.
type TestResult struct {
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	ToolType     internal.ToolType `json:"tool_type"`
	Success      bool              `json:"success"`
	Latency      int64             `json:"latency_ms"` // 改为 int64 毫秒
	StatusCode   int               `json:"status_code,omitempty"`
	Error        string            `json:"error,omitempty"`
	HasAPIKey    bool              `json:"has_api_key"`
	NetworkError bool              `json:"network_error,omitempty"` // 新增字段区分网络错误
}

// OpenAIModelsResponse OpenAI models API 响应.
type OpenAIModelsResponse struct {
	Data   []interface{} `json:"data"`
	Object string        `json:"object"`
}

// AnthropicMessagesResponse Anthropic messages API 响应 (错误时).
type AnthropicMessagesResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func init() {
	testCmd.Flags().BoolP("all", "a", false, "测试所有镜像源")
	testCmd.Flags().BoolP("parallel", "p", false, "并行测试所有镜像源 (与 --all 配合使用)")
	testCmd.Flags().IntP("timeout", "t", 10, "超时时间（秒）")
	testCmd.Flags().Bool("remove-invalid", false, "测试后移除无效的 API Key (仅移除已失效的)")
	testCmd.Flags().Bool("remove-all-invalid", false, "测试后移除所有无效的 API Key (包括认证失败)")
	rootCmd.AddCommand(testCmd)
}

// testMirror 测试单个镜像源.
func testMirror(_ *internal.MirrorManager, mirror *internal.MirrorConfig, timeout int) error {
	result := &TestResult{
		Name:      mirror.Name,
		URL:       mirror.BaseURL,
		ToolType:  mirror.ToolType,
		HasAPIKey: mirror.APIKey != "",
	}

	startTime := time.Now()

	// 测试基础连通性
	reachable, statusCode, err := testConnectivity(mirror, timeout)
	result.Latency = time.Since(startTime).Milliseconds()
	result.StatusCode = statusCode

	if err != nil {
		// 网络错误
		result.Success = false
		result.NetworkError = true
		result.Error = fmt.Sprintf("连接失败: %v", err)
		printTestResult(result)
		return nil
	}

	// 网络可达，根据状态码判断
	if !reachable {
		// 理论上不应该走到这里，因为 testConnectivity 已处理
		result.Success = false
		result.NetworkError = true
		result.Error = "网络不可达"
		printTestResult(result)
		return nil
	}

	// 网络可达，判断 HTTP 状态码
	if statusCode == 200 {
		result.Success = true
		result.NetworkError = false
		printTestResult(result)
		return nil
	}

	if statusCode == 401 {
		result.Success = false
		result.NetworkError = false
		if mirror.APIKey != "" {
			result.Error = "API Key 无效 (401)"
		} else {
			result.Error = needAPIKey401Msg
		}
		printTestResult(result)
		return nil
	}

	// 其他非 200 状态码
	result.Success = false
	result.NetworkError = false
	result.Error = fmt.Sprintf("HTTP %d", statusCode)
	printTestResult(result)

	return nil
}

// testAllMirrors 测试所有镜像源.
func testAllMirrors(mm *internal.MirrorManager, parallel bool, timeout int) error {
	mirrors := mm.ListActiveMirrors()

	if len(mirrors) == 0 {
		return fmt.Errorf("未配置任何镜像源")
	}

	fmt.Printf("🧪 开始测试 %d 个镜像源...\n\n", len(mirrors))

	var results []*TestResult

	if parallel {
		// 并行测试
		resultCh := make(chan *TestResult, len(mirrors))
		for i := range mirrors {
			mirror := &mirrors[i] // Create pointer to avoid race condition
			go func() {
				result := runTest(mm, mirror, timeout)
				resultCh <- result
			}()
		}

		for i := 0; i < len(mirrors); i++ {
			results = append(results, <-resultCh)
		}
	} else {
		// 顺序测试
		for i := range mirrors {
			result := runTest(mm, &mirrors[i], timeout)
			results = append(results, result)
			printTestResult(result)
			fmt.Println()
		}
	}

	// 汇总统计
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	fmt.Println("📊 测试结果汇总:")
	fmt.Printf("   成功: %d/%d\n", successCount, len(mirrors))

	if successCount < len(mirrors) {
		fmt.Println("\n❌ 以下镜像源测试失败:")
		for _, r := range results {
			if !r.Success {
				fmt.Printf("   - %s: %s\n", r.Name, r.Error)
			}
		}
	}

	return nil
}

// runTest 执行测试（供并行调用）.
func runTest(_ *internal.MirrorManager, mirror *internal.MirrorConfig, timeout int) *TestResult {
	result := &TestResult{
		Name:      mirror.Name,
		URL:       mirror.BaseURL,
		ToolType:  mirror.ToolType,
		HasAPIKey: mirror.APIKey != "",
	}

	startTime := time.Now()

	// 测试基础连通性
	reachable, statusCode, err := testConnectivity(mirror, timeout)
	result.Latency = time.Since(startTime).Milliseconds()
	result.StatusCode = statusCode

	// 网络错误
	if err != nil {
		result.Success = false
		result.NetworkError = true
		result.Error = fmt.Sprintf("连接失败: %v", err)
		return result
	}

	// 网络不可达（防御性检查）
	if !reachable {
		result.Success = false
		result.NetworkError = true
		result.Error = "网络不可达"
		return result
	}

	// 根据状态码判断
	switch statusCode {
	case 200:
		result.Success = true
		result.NetworkError = false
	case 401:
		result.Success = false
		result.NetworkError = false
		if mirror.APIKey != "" {
			result.Error = "API Key 无效 (401)"
		} else {
			result.Error = "需要 API Key (401)"
		}
	default:
		result.Success = false
		result.NetworkError = false
		result.Error = fmt.Sprintf("HTTP %d", statusCode)
	}

	return result
}

// testConnectivity 测试基础连通性（不验证认证）.
// 返回: reachable (网络是否可达), statusCode (HTTP 状态码), err (错误).
// 注意: statusCode 仅在网络可达时有效.
func testConnectivity(mirror *internal.MirrorConfig, timeout int) (reachable bool, statusCode int, err error) {
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// 测试端点 - Claude 用 messages, Codex 用 models
	var testURL string
	switch mirror.ToolType {
	case internal.ToolTypeClaude:
		testURL = strings.TrimSuffix(mirror.BaseURL, "/") + "/v1/messages"
	default:
		testURL = strings.TrimSuffix(mirror.BaseURL, "/") + "/v1/models"
	}

	// Claude API 必须用 POST，其他用 GET
	var req *http.Request
	var httpErr error

	if mirror.ToolType == internal.ToolTypeClaude {
		// Claude: 发送最小化的 POST 请求
		body := `{"model": "claude-sonnet-4-20250514", "max_tokens": 1, "messages": [{"role": "user", "content": "test"}]}`
		req, httpErr = http.NewRequest("POST", testURL, bytes.NewBufferString(body))
		if httpErr != nil {
			return false, 0, httpErr
		}
		req.Header.Set("Content-Type", "application/json")
		// 如果有 key 就加上，没有也没关系
		if mirror.APIKey != "" {
			req.Header.Set("x-api-key", mirror.APIKey)
		}
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		// Codex/OpenAI: 使用 GET 请求
		req, httpErr = http.NewRequest("GET", testURL, http.NoBody)
		if httpErr != nil {
			return false, 0, httpErr
		}
		if mirror.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+mirror.APIKey)
		}
	}

	resp, httpErr := client.Do(req)
	if httpErr != nil {
		return false, 0, httpErr
	}
	defer func() { _ = resp.Body.Close() }()

	// 网络成功到达，返回状态码和响应体（可能用于错误信息）.
	// 所有 HTTP 状态码都视为网络可达，由调用方判断语义.
	// 注意：这里不返回错误，只用于状态码判断.
	return true, resp.StatusCode, nil
}

// printTestResult 打印测试结果.
func printTestResult(result *TestResult) {
	if result.Success {
		fmt.Printf("✅ %s\n", result.Name)
	} else {
		fmt.Printf("❌ %s\n", result.Name)
	}

	fmt.Printf("   URL: %s\n", result.URL)
	fmt.Printf("   类型: %s\n", result.ToolType)

	if result.HasAPIKey {
		fmt.Printf("   API Key: ✓ 已配置\n")
	} else {
		fmt.Printf("   API Key: ✗ 未配置\n")
	}

	if result.Latency > 0 {
		fmt.Printf("   延迟: %dms\n", result.Latency)
	}

	if result.StatusCode > 0 {
		fmt.Printf("   HTTP 状态: %d\n", result.StatusCode)
	}

	if result.Error != "" {
		fmt.Printf("   错误: %s\n", result.Error)
	}

	if result.NetworkError {
		fmt.Printf("   类型: 网络错误\n")
	}
}

// PrintResultsAsJSON 将结果打印为 JSON 格式.
func PrintResultsAsJSON(results []*TestResult) {
	data, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(data))
}

// GetTestResultsFromAll 测试所有镜像源并返回结果（供程序使用）.
func GetTestResultsFromAll(mm *internal.MirrorManager, timeout int) []*TestResult {
	mirrors := mm.ListActiveMirrors()
	results := make([]*TestResult, 0, len(mirrors))

	for i := range mirrors {
		result := runTest(mm, &mirrors[i], timeout)
		results = append(results, result)
	}

	return results
}

// IsAnyMirrorReachable 检查是否有任何镜像源可达.
func IsAnyMirrorReachable(mm *internal.MirrorManager, timeout int) bool {
	results := GetTestResultsFromAll(mm, timeout)
	for _, r := range results {
		if r.Success {
			return true
		}
	}
	return false
}

// testAndRemoveInvalidKeys 测试并移除无效的 API Key.
func testAndRemoveInvalidKeys(mm *internal.MirrorManager, testAll, removeAll bool, timeout int) error {
	var mirrors []internal.MirrorConfig

	if testAll {
		mirrors = mm.ListActiveMirrors()
	} else {
		// 测试当前激活的镜像
		var currentMirror *internal.MirrorConfig
		switch {
		case mm.GetConfig().CurrentClaude != "":
			currentMirror, _ = mm.GetCurrentClaudeMirror()
		default:
			currentMirror, _ = mm.GetCurrentCodexMirror()
		}
		if currentMirror == nil {
			return fmt.Errorf("未找到当前激活的镜像源")
		}
		mirrors = []internal.MirrorConfig{*currentMirror}
	}

	if len(mirrors) == 0 {
		return fmt.Errorf("未配置任何镜像源")
	}

	fmt.Printf("🔍 开始测试并清理无效 API Key...\n\n")

	var removedKeys []string
	var invalidMirrors []string

	for i := range mirrors {
		mirror := &mirrors[i]
		if mirror.APIKey == "" {
			continue // 跳过没有 API Key 的镜像源
		}

		fmt.Printf("测试: %s (%s)\n", mirror.Name, mirror.ToolType)

		result := runTest(mm, mirror, timeout)

		if result.Success {
			fmt.Printf("   ✅ API Key 有效\n\n")
		} else {
			invalidMirrors = append(invalidMirrors, mirror.Name)

			// 判断是否应该移除
			shouldRemove := false
			reason := ""

			switch {
			case strings.Contains(result.Error, "401") || strings.Contains(result.Error, "Unauthorized"):
				shouldRemove = true
				reason = "API Key 已失效 (401)"
			case strings.Contains(result.Error, "连接失败"):
				if removeAll {
					shouldRemove = true
					reason = "连接失败 (移除全部无效)"
				} else {
					reason = "连接失败 (跳过，仅移除失效的)"
				}
			case result.StatusCode >= 400:
				if removeAll {
					shouldRemove = true
					reason = fmt.Sprintf("HTTP %d (移除全部无效)", result.StatusCode)
				} else {
					reason = fmt.Sprintf("HTTP %d (跳过，仅移除失效的)", result.StatusCode)
				}
			}

			fmt.Printf("   ❌ %s\n", result.Error)

			if shouldRemove {
				// 清除 API Key - 使用新的专用方法
				err := mm.ClearAPIKey(mirror.Name)
				if err != nil {
					fmt.Printf("   ⚠️  清除 API Key 失败: %v\n", err)
				} else {
					removedKeys = append(removedKeys, mirror.Name)
					fmt.Printf("   🗑️  已清除无效的 API Key\n")
				}
			} else {
				fmt.Printf("   ⏭️  %s\n", reason)
			}
			fmt.Println()
		}
	}

	// 输出汇总
	fmt.Println("📊 清理结果汇总:")
	fmt.Printf("   测试镜像源: %d\n", len(mirrors))
	fmt.Printf("   无效镜像源: %d\n", len(invalidMirrors))
	fmt.Printf("   已清除 Key: %d\n", len(removedKeys))

	if len(removedKeys) > 0 {
		fmt.Println("\n🗑️  已清除 API Key 的镜像源:")
		for _, name := range removedKeys {
			fmt.Printf("   - %s\n", name)
		}
		fmt.Println("\n💡 提示: 如需继续使用这些镜像源，请运行:")
		fmt.Printf("   codex-mirror update %s --api-key <new-key>\n", removedKeys[0])
		fmt.Println("   ⚠️  注意：此操作仅清除配置文件中的 API Key，环境变量将在下次 switch 时更新")
	}

	if len(invalidMirrors) > 0 && len(removedKeys) < len(invalidMirrors) {
		fmt.Println("\n⏭️  跳过的镜像源 (连接失败):")
		for _, name := range invalidMirrors {
			found := false
			for _, r := range removedKeys {
				if r == name {
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("   - %s\n", name)
			}
		}
		fmt.Println("\n💡 提示: 使用 --remove-all-invalid 强制清除所有无效的 API Key")
	}

	return nil
}
