package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codex-mirror/internal"

	"github.com/spf13/cobra"
)

// setupTestEnvironment 设置测试环境.
func setupTestEnvironment(t *testing.T) (string, func()) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 保存原始环境变量
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")

	// 设置临时home目录
	os.Setenv("HOME", tempDir)
	os.Setenv("USERPROFILE", tempDir)

	// 创建清理函数
	cleanup := func() {
		if oldHome != "" {
			os.Setenv("HOME", oldHome)
		} else {
			os.Unsetenv("HOME")
		}

		if oldUserProfile != "" {
			os.Setenv("USERPROFILE", oldUserProfile)
		} else {
			os.Unsetenv("USERPROFILE")
		}
	}

	return tempDir, cleanup
}

// executeCommand 执行命令并捕获输出.
func executeCommand(rootCmd *cobra.Command, args ...string) (string, string, error) {
	// 捕获标准输出
	oldStdout := os.Stdout
	stdoutReader, stdoutWriter, _ := os.Pipe()
	os.Stdout = stdoutWriter

	// 捕获标准错误
	oldStderr := os.Stderr
	stderrReader, stderrWriter, _ := os.Pipe()
	os.Stderr = stderrWriter

	// 创建新的命令实例以避免状态污染
	cmd := &cobra.Command{
		Use: rootCmd.Use,
	}
	for _, subCmd := range rootCmd.Commands() {
		cmd.AddCommand(subCmd)
	}

	// 设置参数
	cmd.SetArgs(args)

	// 执行命令
	err := cmd.Execute()

	// 恢复输出
	stdoutWriter.Close()
	stderrWriter.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// 读取输出
	var stdoutBuf, stderrBuf bytes.Buffer
	io.Copy(&stdoutBuf, stdoutReader)
	io.Copy(&stderrBuf, stderrReader)

	return stdoutBuf.String(), stderrBuf.String(), err
}

// TestAddCommand 测试add命令.
func TestAddCommand(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkOutput func(t *testing.T, stdout, stderr string)
	}{
		{
			name:        "添加基本Codex镜像源",
			args:        []string{"add", "test-codex", "https://api.test.com", "sk-test123"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stdout, "成功添加镜像源") {
					t.Errorf("Expected success message, got stdout: %s", stdout)
				}
			},
		},
		{
			name:        "添加Claude镜像源",
			args:        []string{"add", "test-claude", "https://api.anthropic.com", "sk-ant-test", "--type", "claude"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stdout, "成功添加镜像源") {
					t.Errorf("Expected success message, got stdout: %s", stdout)
				}
			},
		},
		{
			name:        "添加带模型名称的Claude镜像源",
			args:        []string{"add", "test-claude-model", "https://api.custom.com", "sk-test", "--type", "claude", "--model", "claude-3-5-sonnet-20241022"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stdout, "成功添加镜像源") {
					t.Errorf("Expected success message, got stdout: %s", stdout)
				}
			},
		},
		{
			name:        "添加重复镜像源",
			args:        []string{"add", "test-codex", "https://api.another.com", "sk-another"},
			expectError: false, // 会被特殊处理
			checkOutput: func(t *testing.T, stdout, stderr string) {
				// 检查第二个添加操作是否失败
				if strings.Contains(stdout, "成功添加") {
					t.Errorf("Should not succeed when adding duplicate mirror")
				}
				if !strings.Contains(stderr, "已存在") {
					t.Errorf("Expected error about existing mirror in stderr, got stderr: %s", stderr)
				}
			},
		},
		{
			name:        "参数不足",
			args:        []string{"add", "onlyname"},
			expectError: true,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				// Cobra会处理参数验证错误
			},
		},
		{
			name:        "无效的工具类型",
			args:        []string{"add", "invalid-type", "https://api.test.com", "sk-test", "--type", "invalid"},
			expectError: true,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stderr, "不支持的工具类型") {
					t.Errorf("Expected error about invalid tool type, got stderr: %s", stderr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 对于重复镜像源测试，我们需要先添加镜像源，再测试重复添加
			if tt.name == "添加重复镜像源" {
				// 先添加一个镜像源
				_, _, err1 := executeCommand(rootCmd, "add", "test-codex-dup", "https://api.dup.com", "sk-dup")
				if err1 != nil {
					t.Fatalf("Failed to add initial mirror: %v", err1)
				}

				// 再添加同名镜像源，应该失败
				stdout, stderr, err := executeCommand(rootCmd, "add", "test-codex-dup", "https://api.another.com", "sk-another")
				if err == nil {
					t.Errorf("Expected error when adding duplicate mirror, but got none")
					t.Errorf("stdout: %s", stdout)
					t.Errorf("stderr: %s", stderr)
				}

				if tt.checkOutput != nil {
					tt.checkOutput(t, stdout, stderr)
				}
				return
			}

			stdout, stderr, err := executeCommand(rootCmd, tt.args...)

			if (err != nil) != tt.expectError {
				t.Errorf("executeCommand() error = %v, expectError %v", err, tt.expectError)
				t.Errorf("stdout: %s", stdout)
				t.Errorf("stderr: %s", stderr)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout, stderr)
			}
		})
	}
}

// TestListCommand 测试list命令.
func TestListCommand(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 先添加一些镜像源
	executeCommand(rootCmd, "add", "test1", "https://api.test1.com", "sk-test1")
	executeCommand(rootCmd, "add", "test2", "https://api.test2.com", "sk-test2", "--type", "claude")

	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkOutput func(t *testing.T, stdout, stderr string)
	}{
		{
			name:        "列出所有镜像源",
			args:        []string{"list"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stdout, "test1") {
					t.Errorf("Expected test1 in output, got: %s", stdout)
				}
				if !strings.Contains(stdout, "test2") {
					t.Errorf("Expected test2 in output, got: %s", stdout)
				}
				if !strings.Contains(stdout, "https://api.test1.com") {
					t.Errorf("Expected test1 URL in output, got: %s", stdout)
				}
			},
		},
		{
			name:        "列出Codex镜像源",
			args:        []string{"list", "--type", "codex"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stdout, "test1") {
					t.Errorf("Expected test1 in filtered output, got: %s", stdout)
				}
				// 不应该包含Claude类型的镜像源
				lines := strings.Split(stdout, "\n")
				for _, line := range lines {
					if strings.Contains(line, "test2") && strings.Contains(line, "claude") {
						t.Errorf("Should not include Claude mirrors in Codex filter, got line: %s", line)
					}
				}
			},
		},
		{
			name:        "列出Claude镜像源",
			args:        []string{"list", "--type", "claude"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stdout, "test2") {
					t.Errorf("Expected test2 in filtered output, got: %s", stdout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeCommand(rootCmd, tt.args...)

			if (err != nil) != tt.expectError {
				t.Errorf("executeCommand() error = %v, expectError %v", err, tt.expectError)
				t.Errorf("stdout: %s", stdout)
				t.Errorf("stderr: %s", stderr)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout, stderr)
			}
		})
	}
}

// TestSwitchCommand 测试switch命令.
func TestSwitchCommand(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 先添加测试镜像源
	executeCommand(rootCmd, "add", "switch-test", "https://api.switch.com", "sk-switch")

	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkOutput func(t *testing.T, stdout, stderr string)
	}{
		{
			name:        "切换到存在的镜像源",
			args:        []string{"switch", "switch-test"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				combined := stdout + stderr
				if !strings.Contains(combined, "成功切换") {
					t.Errorf("Expected success message, got output: %s", combined)
				}
			},
		},
		{
			name:        "切换到不存在的镜像源",
			args:        []string{"switch", "nonexistent"},
			expectError: true,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stderr, "不存在") {
					t.Errorf("Expected error about non-existent mirror, got stderr: %s", stderr)
				}
			},
		},
		{
			name:        "参数不足",
			args:        []string{"switch"},
			expectError: true,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				// Cobra会处理参数验证
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeCommand(rootCmd, tt.args...)

			if (err != nil) != tt.expectError {
				t.Errorf("executeCommand() error = %v, expectError %v", err, tt.expectError)
				t.Errorf("stdout: %s", stdout)
				t.Errorf("stderr: %s", stderr)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout, stderr)
			}
		})
	}
}

// TestRemoveCommand 测试remove命令.
func TestRemoveCommand(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 先添加测试镜像源
	executeCommand(rootCmd, "add", "remove-test", "https://api.remove.com", "sk-remove")

	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkOutput func(t *testing.T, stdout, stderr string)
	}{
		{
			name:        "删除存在的镜像源",
			args:        []string{"remove", "remove-test"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stdout, "成功删除") {
					t.Errorf("Expected success message, got stdout: %s", stdout)
				}
			},
		},
		{
			name:        "删除不存在的镜像源",
			args:        []string{"remove", "nonexistent"},
			expectError: true,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stderr, "不存在") {
					t.Errorf("Expected error about non-existent mirror, got stderr: %s", stderr)
				}
			},
		},
		{
			name:        "尝试删除官方镜像源",
			args:        []string{"remove", "official"},
			expectError: true,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stderr, "不能删除官方镜像源") {
					t.Errorf("Expected error about removing official mirror, got stderr: %s", stderr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeCommand(rootCmd, tt.args...)

			if (err != nil) != tt.expectError {
				t.Errorf("executeCommand() error = %v, expectError %v", err, tt.expectError)
				t.Errorf("stdout: %s", stdout)
				t.Errorf("stderr: %s", stderr)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout, stderr)
			}
		})
	}
}

// TestStatusCommand 测试status命令.
func TestStatusCommand(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkOutput func(t *testing.T, stdout, stderr string)
	}{
		{
			name:        "查看状态",
			args:        []string{"status"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				// 应该包含基本的状态信息
				if !strings.Contains(stdout, "镜像源状态") && !strings.Contains(stdout, "当前") {
					t.Errorf("Expected status information, got stdout: %s", stdout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeCommand(rootCmd, tt.args...)

			if (err != nil) != tt.expectError {
				t.Errorf("executeCommand() error = %v, expectError %v", err, tt.expectError)
				t.Errorf("stdout: %s", stdout)
				t.Errorf("stderr: %s", stderr)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout, stderr)
			}
		})
	}
}

// TestHelpCommand 测试help命令.
func TestHelpCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkOutput func(t *testing.T, stdout, stderr string)
	}{
		{
			name:        "根命令帮助",
			args:        []string{"--help"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stdout, "Codex镜像切换工具") {
					t.Errorf("Expected root help content, got stdout: %s", stdout)
				}
				if !strings.Contains(stdout, "add") || !strings.Contains(stdout, "list") {
					t.Errorf("Expected subcommands in help, got stdout: %s", stdout)
				}
			},
		},
		{
			name:        "add命令帮助",
			args:        []string{"add", "--help"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stdout, "添加新的镜像源") {
					t.Errorf("Expected add help content, got stdout: %s", stdout)
				}
				if !strings.Contains(stdout, "--type") {
					t.Errorf("Expected --type flag in add help, got stdout: %s", stdout)
				}
			},
		},
		{
			name:        "switch命令帮助",
			args:        []string{"switch", "--help"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stdout, "切换到指定的镜像源") {
					t.Errorf("Expected switch help content, got stdout: %s", stdout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeCommand(rootCmd, tt.args...)

			if (err != nil) != tt.expectError {
				t.Errorf("executeCommand() error = %v, expectError %v", err, tt.expectError)
				t.Errorf("stdout: %s", stdout)
				t.Errorf("stderr: %s", stderr)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout, stderr)
			}
		})
	}
}

// TestCommandIntegration 测试命令集成流程.
func TestCommandIntegration(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 集成测试：完整的操作流程
	t.Run("完整操作流程", func(t *testing.T) {
		// 1. 添加第一个镜像源
		stdout, stderr, err := executeCommand(rootCmd, "add", "integration1", "https://api.int1.com", "sk-int1")
		if err != nil {
			t.Fatalf("Failed to add first mirror: %v, stderr: %s", err, stderr)
		}
		if !strings.Contains(stdout, "成功添加") {
			t.Errorf("Expected success message for first add, got: %s", stdout)
		}

		// 2. 添加第二个镜像源
		stdout, stderr, err = executeCommand(rootCmd, "add", "integration2", "https://api.int2.com", "sk-int2", "--type", "claude")
		if err != nil {
			t.Fatalf("Failed to add second mirror: %v, stderr: %s", err, stderr)
		}

		// 3. 列出所有镜像源
		stdout, stderr, err = executeCommand(rootCmd, "list")
		if err != nil {
			t.Fatalf("Failed to list mirrors: %v, stderr: %s", err, stderr)
		}
		if !strings.Contains(stdout, "integration1") || !strings.Contains(stdout, "integration2") {
			t.Errorf("List should contain both mirrors, got: %s", stdout)
		}

		// 4. 切换镜像源
		stdout, stderr, err = executeCommand(rootCmd, "switch", "integration1")
		if err != nil {
			t.Fatalf("Failed to switch mirror: %v, stderr: %s", err, stderr)
		}

		// 5. 查看状态
		stdout, stderr, err = executeCommand(rootCmd, "status")
		if err != nil {
			t.Fatalf("Failed to get status: %v, stderr: %s", err, stderr)
		}

		// 6. 删除一个镜像源
		stdout, stderr, err = executeCommand(rootCmd, "remove", "integration2")
		if err != nil {
			t.Fatalf("Failed to remove mirror: %v, stderr: %s", err, stderr)
		}

		// 7. 再次列出确认删除
		stdout, stderr, err = executeCommand(rootCmd, "list")
		if err != nil {
			t.Fatalf("Failed to list after removal: %v, stderr: %s", err, stderr)
		}
		if strings.Contains(stdout, "integration2") {
			t.Errorf("integration2 should be removed, but still in list: %s", stdout)
		}
		if !strings.Contains(stdout, "integration1") {
			t.Errorf("integration1 should still exist, got: %s", stdout)
		}
	})
}

// TestConfigurationPersistence 测试配置持久化.
func TestConfigurationPersistence(t *testing.T) {
	tempDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 添加镜像源
	_, stderr, err := executeCommand(rootCmd, "add", "persist-test", "https://api.persist.com", "sk-persist")
	if err != nil {
		t.Fatalf("Failed to add mirror: %v, stderr: %s", err, stderr)
	}

	// 验证配置文件是否创建
	configPath := filepath.Join(tempDir, ".codex-mirror", "mirrors.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Configuration file should be created")
	}

	// 验证配置文件内容
	mm, err := internal.NewMirrorManager()
	if err != nil {
		t.Fatalf("Failed to create mirror manager: %v", err)
	}

	mirrors := mm.ListMirrors()
	found := false
	for _, mirror := range mirrors {
		if mirror.Name == "persist-test" {
			found = true
			if mirror.BaseURL != "https://api.persist.com" {
				t.Errorf("Expected BaseURL https://api.persist.com, got %s", mirror.BaseURL)
			}
			if mirror.APIKey != "sk-persist" {
				t.Errorf("Expected APIKey sk-persist, got %s", mirror.APIKey)
			}
			break
		}
	}

	if !found {
		t.Error("Added mirror should be found in configuration")
	}
}

// TestErrorHandling 测试错误处理.
func TestErrorHandling(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorCheck  func(t *testing.T, stderr string)
	}{
		{
			name:        "无效命令",
			args:        []string{"invalid-command"},
			expectError: true,
			errorCheck: func(t *testing.T, stderr string) {
				// Cobra会处理无效命令
			},
		},
		{
			name:        "切换到不存在的镜像源",
			args:        []string{"switch", "does-not-exist"},
			expectError: true,
			errorCheck: func(t *testing.T, stderr string) {
				if !strings.Contains(stderr, "不存在") {
					t.Errorf("Expected error about non-existent mirror, got: %s", stderr)
				}
			},
		},
		{
			name:        "删除不存在的镜像源",
			args:        []string{"remove", "does-not-exist"},
			expectError: true,
			errorCheck: func(t *testing.T, stderr string) {
				if !strings.Contains(stderr, "不存在") {
					t.Errorf("Expected error about non-existent mirror, got: %s", stderr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeCommand(rootCmd, tt.args...)

			if (err != nil) != tt.expectError {
				t.Errorf("executeCommand() error = %v, expectError %v", err, tt.expectError)
				t.Errorf("stdout: %s", stdout)
				t.Errorf("stderr: %s", stderr)
			}

			if tt.expectError && tt.errorCheck != nil {
				tt.errorCheck(t, stderr)
			}
		})
	}
}

// TestCommandFlags 测试命令标志.
func TestCommandFlags(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkOutput func(t *testing.T, stdout, stderr string)
	}{
		{
			name:        "add命令with type标志",
			args:        []string{"add", "flag-test", "https://api.flag.com", "sk-flag", "--type", "claude"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !strings.Contains(stdout, "成功添加") {
					t.Errorf("Expected success message, got: %s", stdout)
				}
			},
		},
		{
			name:        "list命令with type过滤",
			args:        []string{"list", "--type", "claude"},
			expectError: false,
			checkOutput: func(t *testing.T, stdout, stderr string) {
				// 应该只显示Claude类型的镜像源
				if strings.Contains(stdout, "codex") && !strings.Contains(stdout, "claude") {
					t.Errorf("Should filter by Claude type, got: %s", stdout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := executeCommand(rootCmd, tt.args...)

			if (err != nil) != tt.expectError {
				t.Errorf("executeCommand() error = %v, expectError %v", err, tt.expectError)
				t.Errorf("stdout: %s", stdout)
				t.Errorf("stderr: %s", stderr)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout, stderr)
			}
		})
	}
}
