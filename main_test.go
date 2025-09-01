package main

import "testing"

// TestMain 基本测试函数
func TestMain(t *testing.T) {
	// 基本测试，确保 main 包可以正常编译
	if testing.Short() {
		t.Skip("跳过 main 测试")
	}
}