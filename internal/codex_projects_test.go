package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProjectsPathQuoting 测试projects路径键的引号保留.
func TestProjectsPathQuoting(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// 创建包含projects配置的初始文件
	initialConfig := `model_provider = "test"

[projects."/Users/yuqiang/work/code/bpt-all"]
  trust_level = "trusted"

[projects."/Users/yuqiang/work/code/another-project"]
  trust_level = "untrusted"
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

	// 验证路径键保留引号
	if !strings.Contains(contentStr, `[projects."/Users/yuqiang/work/code/bpt-all"]`) {
		t.Errorf("配置文件中projects路径键丢失了引号")
	}

	if !strings.Contains(contentStr, `[projects."/Users/yuqiang/work/code/another-project"]`) {
		t.Errorf("配置文件中another-project路径键丢失了引号")
	}

	// 验证不应该有错误的格式（缺少引号）
	if strings.Contains(contentStr, "[projects./Users/") {
		t.Errorf("配置文件中路径键缺少了开头的引号")
	}
}
