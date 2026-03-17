//go:build windows

package handler

import "os/exec"

func setPanelChatProcessGroup(cmd *exec.Cmd) {}

func killPanelChatProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
