//go:build windows

package programs

import (
	"os/exec"
	"strconv"
	"syscall"

	"golang.org/x/sys/windows"
)

func init() {
	killProcessByPID = terminateProcessByPID
}

func terminateProcessByPID(pid int) error {
	cmd := exec.Command("taskkill.exe", "/PID", strconv.Itoa(pid), "/F")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	taskkillErr := cmd.Run()
	if taskkillErr == nil {
		return nil
	}

	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err == nil {
		defer windows.CloseHandle(handle)
		if termErr := windows.TerminateProcess(handle, 1); termErr == nil {
			return nil
		}
	}

	if !defaultProcessLookup(runtimeEntry{PID: pid}) {
		return nil
	}

	return taskkillErr
}
