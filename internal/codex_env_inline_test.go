package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnvFieldInlineFormat 测试env字段使用内联表格式而不是独立节.
func TestEnvFieldInlineFormat(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// 创建初始配置，包含mcp_servers的env字段
	initialConfig := `model_provider = "test"

[mcp_servers.zai-mcp-server]
command = "bunx"
args = ["@z_ai/mcp-server"]
env = { "Z_AI_API_KEY" = "test_key", "Z_AI_MODE" = "ZHIPU" }
`

	if err := os.WriteFile(configPath, []byte(initialConfig), 0o644); err != nil {
		t.Fatalf("写入初始配置失败: %v", err)
	}

	// 使用CodexConfigManager读取并重写配置
	ccm := &CodexConfigManager{configPath: configPath}

	// 加载配置
	_, rawConfig, err := ccm.loadExistingConfig()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 重写配置文件
	if err := ccm.writeConfigFile(rawConfig); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 读取生成的配置内容
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}
	contentStr := string(content)
	t.Logf("生成的配置内容:\n%s", contentStr)

	// 验证使用内联表格式
	if !strings.Contains(contentStr, `env = {`) {
		t.Errorf("配置文件中env字段未使用内联表格式")
	}

	// 验证不应该有独立的env节
	if strings.Contains(contentStr, "[mcp_servers.zai-mcp-server.env]") {
		t.Errorf("配置文件错误地使用了独立的env节")
	}

	// 验证env内容正确
	if !strings.Contains(contentStr, `"Z_AI_API_KEY"`) && !strings.Contains(contentStr, `Z_AI_API_KEY`) {
		t.Errorf("配置文件缺少Z_AI_API_KEY")
	}
	if !strings.Contains(contentStr, `"Z_AI_MODE"`) && !strings.Contains(contentStr, `Z_AI_MODE`) {
		t.Errorf("配置文件缺少Z_AI_MODE")
	}
}
