//go:build !windows

package programs

import "os/exec"

func attachConsoleWindow(cmd *exec.Cmd) {}

func buildBatchCommand(entry Entry, args []string) *exec.Cmd {
	cmd := exec.Command("sh", append([]string{entry.Path}, args...)...)
	cmd.Dir = entry.WorkingDirectory
	return cmd
}

func batchCommandArgs(path string, args []string) []string {
	return append([]string{path}, args...)
}
