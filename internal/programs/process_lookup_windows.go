//go:build windows

package programs

import (
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func defaultProcessLookup(entry runtimeEntry) bool {
	if entry.PID <= 0 {
		return false
	}

	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(entry.PID))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	return true
}

func probeProcessByPath(path string) (int, bool) {
	target := strings.ToLower(filepath.Clean(path))
	if target == "" || target == "." {
		return 0, false
	}

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, false
	}
	defer windows.CloseHandle(snapshot)

	var procEntry windows.ProcessEntry32
	procEntry.Size = uint32(unsafe.Sizeof(procEntry))
	if err := windows.Process32First(snapshot, &procEntry); err != nil {
		return 0, false
	}

	for {
		handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, procEntry.ProcessID)
		if err == nil {
			exePath, queryErr := queryProcessPath(handle)
			windows.CloseHandle(handle)
			if queryErr == nil {
				if strings.EqualFold(filepath.Clean(exePath), filepath.Clean(path)) {
					return int(procEntry.ProcessID), true
				}
			}
		}

		if err := windows.Process32Next(snapshot, &procEntry); err != nil {
			return 0, false
		}
	}
}

func queryProcessPath(handle windows.Handle) (string, error) {
	size := uint32(260)
	for {
		buffer := make([]uint16, size)
		current := size
		err := windows.QueryFullProcessImageName(handle, 0, &buffer[0], &current)
		if err == windows.ERROR_INSUFFICIENT_BUFFER {
			size *= 2
			continue
		}
		if err != nil {
			return "", err
		}
		return windows.UTF16ToString(buffer[:current]), nil
	}
}
