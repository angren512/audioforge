//go:build !windows

package selftest

import "os/exec"

func hideConsole(cmd *exec.Cmd) {}
