package programs

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkServiceListPrograms(b *testing.B) {
	store := &fakeStore{entries: makeBenchmarkEntries(500)}
	service, err := NewService(store, nil)
	if err != nil {
		b.Fatalf("NewService() error = %v", err)
	}

	originalProbe := processProbe
	originalSnapshot := processSnapshotByPID
	defer func() {
		processProbe = originalProbe
		processSnapshotByPID = originalSnapshot
	}()

	processProbe = func(entry Entry) (int, bool) {
		return 0, false
	}
	processSnapshotByPID = func(pid int) (processSnapshot, bool) {
		return processSnapshot{}, false
	}

	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		_, err := service.ListPrograms(ListQuery{})
		if err != nil {
			b.Fatalf("ListPrograms() error = %v", err)
		}
	}
}

func BenchmarkServiceListProgramsPrefiltered(b *testing.B) {
	store := &fakeStore{entries: makeBenchmarkEntries(500)}
	service, err := NewService(store, nil)
	if err != nil {
		b.Fatalf("NewService() error = %v", err)
	}

	originalProbe := processProbe
	defer func() {
		processProbe = originalProbe
	}()

	processProbe = func(entry Entry) (int, bool) {
		return 0, false
	}

	query := ListQuery{Search: "program-499", Tag: "tag-4"}

	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		_, err := service.ListPrograms(query)
		if err != nil {
			b.Fatalf("ListPrograms() error = %v", err)
		}
	}
}

func BenchmarkServiceListProgramsRunningMixed(b *testing.B) {
	for _, runningCount := range []int{0, 10, 100} {
		b.Run(fmt.Sprintf("running-%d", runningCount), func(b *testing.B) {
			store := &fakeStore{entries: makeBenchmarkEntries(500)}
			service, err := NewService(store, nil)
			if err != nil {
				b.Fatalf("NewService() error = %v", err)
			}

			configureBenchmarkRunningState(service, runningCount)

			originalSnapshot := processSnapshotByPID
			defer func() {
				processSnapshotByPID = originalSnapshot
			}()

			processSnapshotByPID = func(pid int) (processSnapshot, bool) {
				return processSnapshot{
					startedAt:             time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC),
					memoryWorkingSetBytes: 128 * 1024 * 1024,
					memoryPrivateBytes:    64 * 1024 * 1024,
				}, true
			}

			b.ResetTimer()
			for index := 0; index < b.N; index++ {
				_, err := service.ListPrograms(ListQuery{})
				if err != nil {
					b.Fatalf("ListPrograms() error = %v", err)
				}
			}
		})
	}
}

func BenchmarkServiceReconnectProgramsMany(b *testing.B) {
	entries := makeBenchmarkEntries(250)
	store := &fakeStore{entries: entries}
	runtimePrograms := make([]runtimeEntry, 0, len(entries))
	for index, entry := range entries {
		runtimePrograms = append(runtimePrograms, runtimeEntry{
			ID:           entry.ID,
			PID:          6000 + index,
			Path:         entry.Path,
			StartedAt:    time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC),
			CanReconnect: true,
		})
	}

	service, err := NewService(store, nil, WithRuntimeStore(&fakeRuntimeStore{
		state: runtimeDocument{Programs: runtimePrograms},
	}))
	if err != nil {
		b.Fatalf("NewService() error = %v", err)
	}

	originalProbe := processProbe
	originalCandidatesProvider := processCandidatesProvider
	defer func() {
		processProbe = originalProbe
		processCandidatesProvider = originalCandidatesProvider
	}()

	processProbe = func(entry Entry) (int, bool) {
		return 0, false
	}
	processCandidatesProvider = func() []processCandidate {
		candidates := make([]processCandidate, 0, len(entries))
		for index, entry := range entries {
			candidates = append(candidates, processCandidate{
				pid:         6000 + index,
				imagePath:   `C:\Windows\System32\cmd.exe`,
				commandLine: fmt.Sprintf(`cmd.exe /D /C "%s"`, entry.Path),
			})
		}
		return candidates
	}

	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if err := service.ReconnectPrograms(); err != nil {
			b.Fatalf("ReconnectPrograms() error = %v", err)
		}
	}
}

func makeBenchmarkEntries(count int) []Entry {
	entries := make([]Entry, 0, count)
	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)

	for index := 0; index < count; index++ {
		entries = append(entries, Entry{
			ID:                fmt.Sprintf("program-%d", index),
			Name:              fmt.Sprintf("Program-%d", index),
			Description:       "benchmark program",
			Notes:             "notes",
			Tags:              []string{fmt.Sprintf("tag-%d", index%5)},
			Path:              fmt.Sprintf(`C:\tools\program-%d.bat`, index),
			Kind:              KindBatch,
			WorkingDirectory:  `C:\tools`,
			RestartPolicy:     RestartPolicyNone,
			CreatedAt:         now,
			UpdatedAt:         now,
		})
	}

	return entries
}

func configureBenchmarkRunningState(service *Service, runningCount int) {
	service.mu.Lock()
	defer service.mu.Unlock()

	for index, entry := range service.entries {
		state := service.ensureStateLocked(entry.ID)
		if index < runningCount {
			state.status = StatusRunning
			state.pid = 4000 + index
			state.startedAt = time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
			state.canReconnect = true
			continue
		}

		state.status = StatusStopped
		state.pid = 0
		state.startedAt = time.Time{}
		state.canReconnect = false
	}
}
