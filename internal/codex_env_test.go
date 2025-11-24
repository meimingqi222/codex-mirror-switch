package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

const testAPIKey = "a8d43a16951d497d88551eaaa9f8e582.0Vm4cwmvKECfeuIO"

// TestEnvFieldFormatting 测试env字段是否被正确格式化为嵌套节而非内联表.
func TestEnvFieldFormatting(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// 模拟原始配置，包含mcp_servers及其env字段
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
	}

	// 创建配置文件
	file, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("创建配置文件失败: %v", err)
	}

	// 使用writeConfigFile的逻辑来写入
	ccm := &CodexConfigManager{configPath: configPath}
	if err := ccm.writeConfigFile(rawConfig); err != nil {
		file.Close()
		t.Fatalf("写入配置文件失败: %v", err)
	}
	file.Close()

	// 读取生成的配置文件内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}
	contentStr := string(content)
	t.Logf("生成的配置文件内容:\n%s", contentStr)

	// 验证不应该包含内联表格式 env = { ... }
	if strings.Contains(contentStr, "env = {") {
		t.Errorf("配置文件包含错误的内联表格式: env = { ... }")
	}

	// 验证应该包含独立的env节 [mcp_servers.zai-mcp-server.env]
	if !strings.Contains(contentStr, "[mcp_servers.zai-mcp-server.env]") {
		t.Errorf("配置文件缺少独立的env节: [mcp_servers.zai-mcp-server.env]")
	}

	// 验证env节的内容格式正确
	if !strings.Contains(contentStr, `Z_AI_API_KEY = "`+testAPIKey+`"`) {
		t.Errorf("配置文件中env节的API_KEY格式不正确")
	}
	if !strings.Contains(contentStr, `Z_AI_MODE = "ZHIPU"`) {
		t.Errorf("配置文件中env节的MODE格式不正确")
	}

	// 尝试解析配置文件以确保格式正确
	var parsedConfig map[string]interface{}
	if _, err := toml.DecodeFile(configPath, &parsedConfig); err != nil {
		t.Fatalf("TOML解析失败: %v", err)
	}

	// 验证解析后的结构
	mcpServers, ok := parsedConfig["mcp_servers"].(map[string]interface{})
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

	if env["Z_AI_API_KEY"] != testAPIKey {
		t.Errorf("API_KEY值不正确: %v", env["Z_AI_API_KEY"])
	}
	if env["Z_AI_MODE"] != "ZHIPU" {
		t.Errorf("MODE值不正确: %v", env["Z_AI_MODE"])
	}
}

// TestComplexNestedStructure 测试更复杂的嵌套结构.
func TestComplexNestedStructure(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// 模拟包含多个嵌套层级的配置
	rawConfig := map[string]interface{}{
		"model_provider": "test",
		"mcp_servers.server1": map[string]interface{}{
			"command": "cmd1",
			"port":    8080,
			"env": map[string]interface{}{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
		"mcp_servers.server2": map[string]interface{}{
			"command": "cmd2",
			"args":    []interface{}{"arg1", "arg2"},
			"env": map[string]interface{}{
				"KEY3": "value3",
			},
		},
	}

	ccm := &CodexConfigManager{configPath: configPath}
	file, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("创建配置文件失败: %v", err)
	}

	if err := ccm.writeConfigFile(rawConfig); err != nil {
		file.Close()
		t.Fatalf("写入配置文件失败: %v", err)
	}
	file.Close()

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}
	contentStr := string(content)
	t.Logf("生成的配置文件内容:\n%s", contentStr)

	// 验证不应该包含内联表格式
	if strings.Contains(contentStr, "env = {") {
		t.Errorf("配置文件包含错误的内联表格式")
	}

	// 验证应该包含独立的env节
	if !strings.Contains(contentStr, "[mcp_servers.server1.env]") {
		t.Errorf("配置文件缺少server1的env节")
	}
	if !strings.Contains(contentStr, "[mcp_servers.server2.env]") {
		t.Errorf("配置文件缺少server2的env节")
	}

	// 验证能正确解析
	var parsedConfig map[string]interface{}
	if _, err := toml.DecodeFile(configPath, &parsedConfig); err != nil {
		t.Fatalf("TOML解析失败: %v", err)
	}
}
