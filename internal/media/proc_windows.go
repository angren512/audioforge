//go:build windows

package media

import (
	"os/exec"
	"syscall"
)

// hideWindow 让子进程不弹出黑色控制台窗口。
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
