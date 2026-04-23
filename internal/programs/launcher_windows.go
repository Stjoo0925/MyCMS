//go:build windows

package programs

import (
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func attachConsoleWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}

func buildBatchCommand(entry Entry, args []string) *exec.Cmd {
	cmd := exec.Command("cmd.exe", batchCommandArgs(entry.Path, args)...)
	cmd.Dir = entry.WorkingDirectory
	attachConsoleWindow(cmd)
	return cmd
}

func batchCommandArgs(path string, args []string) []string {
	command := make([]string, 0, len(args)+1)
	command = append(command, quoteBatchToken(path))
	for _, arg := range args {
		command = append(command, quoteBatchToken(arg))
	}
	return []string{"/D", "/C", strings.Join(command, " ")}
}

func quoteBatchToken(value string) string {
	escaped := strings.ReplaceAll(value, `"`, `""`)
	return `"` + escaped + `"`
}
