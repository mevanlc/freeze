//go:build windows

package main

import "os/exec"

func configureChromiumProcess(_ *exec.Cmd) {}

func signalChromiumProcess(cmd *exec.Cmd, _ bool) error {
	return cmd.Process.Kill()
}
