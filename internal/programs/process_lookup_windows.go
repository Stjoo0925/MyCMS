//go:build windows

package programs

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

type processBasicInformation struct {
	Reserved1       uintptr
	PebBaseAddress  uintptr
	Reserved2       [2]uintptr
	UniqueProcessID uintptr
	Reserved3       uintptr
}

type peb struct {
	Reserved1         [2]byte
	BeingDebugged     byte
	Reserved2         byte
	Reserved3         [2]uintptr
	Ldr               uintptr
	ProcessParameters uintptr
}

type unicodeString struct {
	Length        uint16
	MaximumLength uint16
	Buffer        uintptr
}

type rtlUserProcessParameters struct {
	Reserved1     [16]byte
	Reserved2     [10]uintptr
	ImagePathName unicodeString
	CommandLine   unicodeString
}

var (
	modntdll                    = windows.NewLazySystemDLL("ntdll.dll")
	modkernel32ProcessLookup    = windows.NewLazySystemDLL("kernel32.dll")
	procNtQueryInformation      = modntdll.NewProc("NtQueryInformationProcess")
	procReadProcessMemoryLookup = modkernel32ProcessLookup.NewProc("ReadProcessMemory")
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
	target := normalizeComparablePath(path)
	if target == "" {
		return 0, false
	}

	return scanProcesses(func(candidate processCandidate) bool {
		return normalizeComparablePath(candidate.imagePath) == target
	})
}

func listProcessCandidates() []processCandidate {
	candidates := make([]processCandidate, 0, 64)

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return candidates
	}
	defer windows.CloseHandle(snapshot)

	var procEntry windows.ProcessEntry32
	procEntry.Size = uint32(unsafe.Sizeof(procEntry))
	if err := windows.Process32First(snapshot, &procEntry); err != nil {
		return candidates
	}

	for {
		handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_VM_READ, false, procEntry.ProcessID)
		if err == nil {
			candidate, ok := readProcessCandidate(handle, int(procEntry.ProcessID))
			windows.CloseHandle(handle)
			if ok {
				candidates = append(candidates, candidate)
			}
		}

		if err := windows.Process32Next(snapshot, &procEntry); err != nil {
			return candidates
		}
	}
}

func probeProcess(entry Entry) (int, bool) {
	if entry.Path == "" {
		return 0, false
	}

	if entry.Kind == KindExecutable {
		return probeProcessByPath(entry.Path)
	}

	return scanProcesses(func(candidate processCandidate) bool {
		return matchProcessCandidate(entry, candidate)
	})
}

func scanProcesses(match func(processCandidate) bool) (int, bool) {
	for _, candidate := range listProcessCandidates() {
		if match(candidate) {
			return candidate.pid, true
		}
	}

	return 0, false
}

func readProcessCandidate(handle windows.Handle, pid int) (processCandidate, bool) {
	exePath, err := queryProcessPath(handle)
	if err != nil {
		return processCandidate{}, false
	}

	commandLine, _ := queryProcessCommandLine(handle)
	return processCandidate{
		pid:         pid,
		imagePath:   exePath,
		commandLine: commandLine,
	}, true
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

func queryProcessCommandLine(handle windows.Handle) (string, error) {
	var info processBasicInformation
	infoSize := uint32(unsafe.Sizeof(info))
	status, _, callErr := procNtQueryInformation.Call(
		uintptr(handle),
		0,
		uintptr(unsafe.Pointer(&info)),
		uintptr(infoSize),
		0,
	)
	if status != 0 {
		return "", callErr
	}

	pebValue, err := readRemoteStruct[peb](handle, info.PebBaseAddress)
	if err != nil {
		return "", err
	}

	params, err := readRemoteStruct[rtlUserProcessParameters](handle, pebValue.ProcessParameters)
	if err != nil {
		return "", err
	}

	return readRemoteUnicodeString(handle, params.CommandLine)
}

func readRemoteStruct[T any](handle windows.Handle, address uintptr) (T, error) {
	var value T
	size := unsafe.Sizeof(value)
	if err := readRemoteMemory(handle, address, uintptr(size), unsafe.Pointer(&value)); err != nil {
		return value, err
	}
	return value, nil
}

func readRemoteUnicodeString(handle windows.Handle, value unicodeString) (string, error) {
	if value.Length == 0 || value.Buffer == 0 {
		return "", nil
	}

	buffer := make([]uint16, int(value.Length)/2)
	if err := readRemoteMemory(handle, value.Buffer, uintptr(value.Length), unsafe.Pointer(&buffer[0])); err != nil {
		return "", err
	}

	return windows.UTF16ToString(buffer), nil
}

func readRemoteMemory(handle windows.Handle, address uintptr, size uintptr, out unsafe.Pointer) error {
	var bytesRead uintptr
	r1, _, err := procReadProcessMemoryLookup.Call(
		uintptr(handle),
		address,
		uintptr(out),
		size,
		uintptr(unsafe.Pointer(&bytesRead)),
	)
	if r1 == 0 {
		return err
	}
	if bytesRead < size {
		return windows.ERROR_PARTIAL_COPY
	}
	return nil
}
