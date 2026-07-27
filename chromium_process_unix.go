//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func configureChromiumProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalChromiumProcess(cmd *exec.Cmd, force bool) error {
	signal := syscall.SIGTERM
	if force {
		signal = syscall.SIGKILL
	}
	return syscall.Kill(-cmd.Process.Pid, signal)
}
