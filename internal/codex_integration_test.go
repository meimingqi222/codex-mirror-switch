package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// TestRealWorldMCPServersConfig 测试真实的MCP服务器配置读写场景.
func TestRealWorldMCPServersConfig(t *testing.T) {
	tempDir := t.TempDir()
	// 模拟Codex配置路径
	os.Setenv("HOME", tempDir)
	defer os.Unsetenv("HOME")

	configDir := filepath.Join(tempDir, ".codex")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")

	// 创建一个包含MCP服务器配置的真实config.toml
	initialConfig := `model_provider = "test"
model = "gpt-5"
model_reasoning_effort = "high"
disable_response_storage = true

[mcp_servers.zai-mcp-server]
command = "bunx"
args = ["@z_ai/mcp-server"]
env = { Z_AI_API_KEY = "a8d43a16951d497d88551eaaa9f8e582.0Vm4cwmvKECfeuIO", Z_AI_MODE = "ZHIPU" }

[mcp_servers.another-server]
command = "node"
args = ["server.js"]
env = { API_TOKEN = "token123", DEBUG = "true" }
`

	if err := os.WriteFile(configPath, []byte(initialConfig), 0o644); err != nil {
		t.Fatalf("写入初始配置失败: %v", err)
	}

	// 读取配置
	var config map[string]interface{}
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		t.Fatalf("解析初始配置失败: %v", err)
	}

	// 验证读取的配置结构正确
	mcpServers, ok := config["mcp_servers"].(map[string]interface{})
	if !ok {
		t.Fatal("mcp_servers不是map类型")
	}

	zaiServer, ok := mcpServers["zai-mcp-server"].(map[string]interface{})
	if !ok {
		t.Fatal("zai-mcp-server不是map类型")
	}

	env, ok := zaiServer["env"].(map[string]interface{})
	if !ok {
		t.Fatal("env不是map类型")
	}

	if env["Z_AI_API_KEY"] != "a8d43a16951d497d88551eaaa9f8e582.0Vm4cwmvKECfeuIO" {
		t.Errorf("API_KEY值不正确: %v", env["Z_AI_API_KEY"])
	}

	// 现在使用CodexConfigManager重写配置
	ccm := &CodexConfigManager{configPath: configPath}

	// 构建rawConfig，模拟从文件读取后的状态
	rawConfig := map[string]interface{}{
		"model_provider":           "test",
		"model":                    "gpt-5",
		"model_reasoning_effort":   "high",
		"disable_response_storage": true,
		"mcp_servers.zai-mcp-server": map[string]interface{}{
			"command": "bunx",
			"args":    []interface{}{"@z_ai/mcp-server"},
			"env": map[string]interface{}{
				"Z_AI_API_KEY": testAPIKey,
				"Z_AI_MODE":    "ZHIPU",
			},
		},
		"mcp_servers.another-server": map[string]interface{}{
			"command": "node",
			"args":    []interface{}{"server.js"},
			"env": map[string]interface{}{
				"API_TOKEN": "token123",
				"DEBUG":     "true",
			},
		},
	}

	// 重写配置文件
	if err := ccm.writeConfigFile(rawConfig); err != nil {
		t.Fatalf("重写配置文件失败: %v", err)
	}

	// 读取重写后的配置内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取重写后的配置失败: %v", err)
	}
	contentStr := string(content)
	t.Logf("重写后的配置内容:\n%s", contentStr)

	// 验证格式正确：不应该包含内联表
	if strings.Contains(contentStr, "env = {") {
		t.Errorf("配置文件包含错误的内联表格式")
	}

	// 验证包含正确的env节
	if !strings.Contains(contentStr, "[mcp_servers.zai-mcp-server.env]") {
		t.Errorf("配置文件缺少zai-mcp-server的env节")
	}
	if !strings.Contains(contentStr, "[mcp_servers.another-server.env]") {
		t.Errorf("配置文件缺少another-server的env节")
	}

	// 最重要的：验证能被TOML解析器正确解析
	var reloadedConfig map[string]interface{}
	if _, err := toml.DecodeFile(configPath, &reloadedConfig); err != nil {
		t.Fatalf("重写后的配置无法被TOML解析: %v", err)
	}

	// 验证解析后的数据完整性
	reloadedMcpServers, ok := reloadedConfig["mcp_servers"].(map[string]interface{})
	if !ok {
		t.Fatal("重新加载的mcp_servers不是map类型")
	}

	reloadedZaiServer, ok := reloadedMcpServers["zai-mcp-server"].(map[string]interface{})
	if !ok {
		t.Fatal("重新加载的zai-mcp-server不是map类型")
	}

	reloadedEnv, ok := reloadedZaiServer["env"].(map[string]interface{})
	if !ok {
		t.Fatal("重新加载的env不是map类型")
	}

	if reloadedEnv["Z_AI_API_KEY"] != testAPIKey {
		t.Errorf("重新加载后API_KEY值不正确: %v", reloadedEnv["Z_AI_API_KEY"])
	}
	if reloadedEnv["Z_AI_MODE"] != "ZHIPU" {
		t.Errorf("重新加载后MODE值不正确: %v", reloadedEnv["Z_AI_MODE"])
	}

	t.Log("✅ 真实场景的配置读写测试通过")
}
