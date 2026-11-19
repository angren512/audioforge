//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// revealInExplorer 在资源管理器中选中指定文件。
func revealInExplorer(path string) {
	cmd := exec.Command("explorer.exe", "/select,", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	_ = cmd.Start()
}
