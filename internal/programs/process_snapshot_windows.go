//go:build windows

package programs

import (
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type processMemoryCountersEx struct {
	cb                         uint32
	pageFaultCount             uint32
	peakWorkingSetSize         uintptr
	workingSetSize             uintptr
	quotaPeakPagedPoolUsage    uintptr
	quotaPagedPoolUsage        uintptr
	quotaPeakNonPagedPoolUsage uintptr
	quotaNonPagedPoolUsage     uintptr
	pagefileUsage              uintptr
	peakPagefileUsage          uintptr
	privateUsage               uintptr
}

var (
	modkernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procGetProcessMemoryInfo = modkernel32.NewProc("K32GetProcessMemoryInfo")
)

func init() {
	processSnapshotByPID = readProcessSnapshotByPID
}

func readProcessSnapshotByPID(pid int) (processSnapshot, bool) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return processSnapshot{}, false
	}
	defer windows.CloseHandle(handle)

	snapshot := processSnapshot{}
	ok := false

	if startedAt, err := queryProcessStartTime(handle); err == nil {
		snapshot.startedAt = startedAt
		ok = true
	}

	if workingSet, privateBytes, err := queryProcessMemory(handle); err == nil {
		snapshot.memoryWorkingSetBytes = workingSet
		snapshot.memoryPrivateBytes = privateBytes
		ok = true
	}

	return snapshot, ok
}

func queryProcessStartTime(handle windows.Handle) (time.Time, error) {
	var creation windows.Filetime
	var exit windows.Filetime
	var kernel windows.Filetime
	var user windows.Filetime
	if err := windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, creation.Nanoseconds()).UTC(), nil
}

func queryProcessMemory(handle windows.Handle) (int64, int64, error) {
	info := processMemoryCountersEx{cb: uint32(unsafe.Sizeof(processMemoryCountersEx{}))}
	r1, _, err := procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&info)),
		uintptr(info.cb),
	)
	if r1 == 0 {
		if err != windows.ERROR_SUCCESS {
			return 0, 0, err
		}
		return 0, 0, err
	}
	return int64(info.workingSetSize), int64(info.privateUsage), nil
}
