package programs

import "time"

type processSnapshot struct {
	startedAt             time.Time
	memoryWorkingSetBytes int64
	memoryPrivateBytes    int64
}

var processSnapshotByPID = func(pid int) (processSnapshot, bool) {
	return processSnapshot{}, false
}
