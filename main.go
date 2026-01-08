//go:build cli
// +build cli

package main

import (
	"codex-mirror/cmd"
)

// 程序入口 - CLI 模式
func main() {
	cmd.Execute()
}
