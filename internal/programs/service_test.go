package programs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/korean"
)

type fakeStore struct {
	entries []Entry
}

func (f *fakeStore) Load() ([]Entry, error) {
	result := make([]Entry, len(f.entries))
	copy(result, f.entries)
	return result, nil
}

func (f *fakeStore) Save(entries []Entry) error {
	f.entries = make([]Entry, len(entries))
	copy(f.entries, entries)
	return nil
}

func TestServiceCreateProgramPersistsEntry(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "run.bat", "@echo off\r\nexit /b 0\r\n")
	view, err := service.CreateProgram(Input{Name: "Runner", Path: path})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	if view.Name != "Runner" {
		t.Fatalf("CreateProgram().Name = %q, want %q", view.Name, "Runner")
	}

	if view.Path != path {
		t.Fatalf("CreateProgram().Path = %q, want %q", view.Path, path)
	}

	if view.Status != StatusStopped {
		t.Fatalf("CreateProgram().Status = %q, want %q", view.Status, StatusStopped)
	}

	if len(store.entries) != 1 {
		t.Fatalf("saved entries = %d, want 1", len(store.entries))
	}

	if view.RestartPolicy != RestartPolicyNone {
		t.Fatalf("CreateProgram().RestartPolicy = %q, want %q", view.RestartPolicy, RestartPolicyNone)
	}
	if view.RestartLimit != 0 {
		t.Fatalf("CreateProgram().RestartLimit = %d, want 0", view.RestartLimit)
	}
	if view.RestartDelaySeconds != 0 {
		t.Fatalf("CreateProgram().RestartDelaySeconds = %d, want 0", view.RestartDelaySeconds)
	}
	if view.RunAsAdmin {
		t.Fatal("CreateProgram().RunAsAdmin = true, want false")
	}
}

func TestServiceRejectsDuplicateName(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "run.bat", "@echo off\r\nexit /b 0\r\n")
	if _, err := service.CreateProgram(Input{Name: "Runner", Path: path}); err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	_, err = service.CreateProgram(Input{Name: "runner", Path: path})
	if err == nil {
		t.Fatal("CreateProgram() error = nil, want duplicate error")
	}
}

func TestServiceRejectsMissingPath(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.CreateProgram(Input{
		Name: "Runner",
		Path: filepath.Join(t.TempDir(), "missing.bat"),
	})
	if err == nil {
		t.Fatal("CreateProgram() error = nil, want error")
	}
}

func TestServiceRejectsUnsupportedExtension(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createFile(t, "bad.txt", "hello")
	if _, err := service.CreateProgram(Input{Name: "Bad", Path: path}); err == nil {
		t.Fatal("CreateProgram() error = nil, want unsupported extension error")
	}
}

func TestServiceRejectsDuplicateEnvironmentVariableKeys(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "run.bat", "@echo off\r\nexit /b 0\r\n")
	_, err = service.CreateProgram(Input{
		Name: "Runner",
		Path: path,
		Env: []EnvVar{
			{Key: "MODE", Value: "one"},
			{Key: "MODE", Value: "two"},
		},
	})
	if err == nil {
		t.Fatal("CreateProgram() error = nil, want duplicate env error")
	}
}

func TestServiceUpdateProgramRevalidatesInput(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	firstPath := createBatchFile(t, "first.bat", "@echo off\r\nexit /b 0\r\n")
	secondPath := createBatchFile(t, "second.bat", "@echo off\r\nexit /b 0\r\n")

	first, err := service.CreateProgram(Input{Name: "First", Path: firstPath})
	if err != nil {
		t.Fatalf("CreateProgram(first) error = %v", err)
	}

	if _, err := service.CreateProgram(Input{Name: "Second", Path: secondPath}); err != nil {
		t.Fatalf("CreateProgram(second) error = %v", err)
	}

	_, err = service.UpdateProgram(first.ID, Input{Name: "second", Path: firstPath})
	if err == nil {
		t.Fatal("UpdateProgram() error = nil, want duplicate error")
	}
}

func TestServiceDeleteProgramRemovesEntry(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "run.bat", "@echo off\r\nexit /b 0\r\n")
	entry, err := service.CreateProgram(Input{Name: "Runner", Path: path})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	if err := service.DeleteProgram(entry.ID); err != nil {
		t.Fatalf("DeleteProgram() error = %v", err)
	}

	views, err := service.ListPrograms(ListQuery{})
	if err != nil {
		t.Fatalf("ListPrograms() error = %v", err)
	}

	if len(views) != 0 {
		t.Fatalf("ListPrograms() length = %d, want 0", len(views))
	}
}

func TestServiceStartAndStopProgram(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, func(entry Entry) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", "sleep")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "loop.bat", "@echo off\r\nexit /b 0\r\n")
	entry, err := service.CreateProgram(Input{Name: "Looper", Path: path})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	if err := service.StartProgram(entry.ID); err != nil {
		t.Fatalf("StartProgram() error = %v", err)
	}

	waitForStatus(t, service, entry.ID, StatusRunning)

	if err := service.StopProgram(entry.ID); err != nil {
		t.Fatalf("StopProgram() error = %v", err)
	}

	waitForStatus(t, service, entry.ID, StatusStopped)
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	if len(os.Args) < 3 {
		os.Exit(0)
	}

	switch os.Args[len(os.Args)-1] {
	case "sleep":
		for {
			time.Sleep(250 * time.Millisecond)
		}
	case "stderr-euckr":
		encoded, err := korean.EUCKR.NewEncoder().Bytes([]byte(os.Getenv("GO_HELPER_STDERR_TEXT")))
		if err != nil {
			os.Exit(2)
		}
		_, _ = os.Stderr.Write(encoded)
		_, _ = os.Stderr.Write([]byte("\n"))
		os.Exit(1)
	default:
		os.Exit(0)
	}
}

func TestServiceStartProgramForUnknownIDFails(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if err := service.StartProgram("missing"); err == nil {
		t.Fatal("StartProgram() error = nil, want error")
	}

	if err := service.StopProgram("missing"); err == nil {
		t.Fatal("StopProgram() error = nil, want error")
	}
}

func TestServiceStartProgramRecordsLastError(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, func(entry Entry) *exec.Cmd {
		return exec.Command("does-not-exist.exe")
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "run.bat", "@echo off\r\nexit /b 0\r\n")
	entry, err := service.CreateProgram(Input{Name: "Runner", Path: path})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	err = service.StartProgram(entry.ID)
	if err == nil {
		t.Fatal("StartProgram() error = nil, want error")
	}

	views, err := service.ListPrograms(ListQuery{})
	if err != nil {
		t.Fatalf("ListPrograms() error = %v", err)
	}

	if len(views) != 1 {
		t.Fatalf("ListPrograms() length = %d, want 1", len(views))
	}

	if strings.TrimSpace(views[0].LastError) == "" {
		t.Fatal("LastError = empty, want message")
	}
}

func TestServiceStartProgramAppliesEnvironmentVariables(t *testing.T) {
	store := &fakeStore{}
	var captured *exec.Cmd
	service, err := NewService(store, func(entry Entry) *exec.Cmd {
		captured = exec.Command("cmd", "/C", "exit", "0")
		return captured
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "run.bat", "@echo off\r\nexit /b 0\r\n")
	entry, err := service.CreateProgram(Input{
		Name: "Runner",
		Path: path,
		Env: []EnvVar{
			{Key: "CMS_MODE", Value: "prod"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	if err := service.StartProgram(entry.ID); err != nil {
		t.Fatalf("StartProgram() error = %v", err)
	}

	if captured == nil {
		t.Fatal("command factory did not capture a command")
	}

	if !containsEnvValue(captured.Env, "CMS_MODE=prod") {
		t.Fatalf("command environment = %#v, want CMS_MODE=prod", captured.Env)
	}
}

func TestServiceReconnectProgramsAllowsStoppingByPID(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "loop.bat", "@echo off\r\n:loop\r\nping -n 2 127.0.0.1 >nul\r\ngoto loop\r\n")
	entry, err := service.CreateProgram(Input{Name: "Looper", Path: path})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	if err := service.StartProgram(entry.ID); err != nil {
		t.Fatalf("StartProgram() error = %v", err)
	}

	waitForStatus(t, service, entry.ID, StatusRunning)

	view, err := service.GetProgram(entry.ID)
	if err != nil {
		t.Fatalf("GetProgram() error = %v", err)
	}

	runtimeStore := &fakeRuntimeStore{
		state: runtimeDocument{
			Programs: []runtimeEntry{
				{
					ID:           entry.ID,
					PID:          view.PID,
					Path:         path,
					StartedAt:    parseTimeOrZero(t, view.StartedAt),
					CanReconnect: true,
				},
			},
		},
	}

	reconnected, err := NewService(store, nil, WithRuntimeStore(runtimeStore), WithProcessLookup(func(runtimeEntry) bool { return true }))
	if err != nil {
		t.Fatalf("NewService(reconnected) error = %v", err)
	}

	if err := reconnected.ReconnectPrograms(); err != nil {
		t.Fatalf("ReconnectPrograms() error = %v", err)
	}

	if err := reconnected.StopProgram(entry.ID); err != nil {
		t.Fatalf("StopProgram() after reconnect error = %v", err)
	}

	waitForStatus(t, reconnected, entry.ID, StatusStopped)
}

func TestSortViewsKeepsStableOrderForEqualDescendingKeys(t *testing.T) {
	views := []View{
		{ID: "one", Name: "Alpha", CreatedAt: time.Unix(1, 0).UTC().Format(time.RFC3339Nano)},
		{ID: "two", Name: "Beta", CreatedAt: time.Unix(1, 0).UTC().Format(time.RFC3339Nano)},
		{ID: "three", Name: "Gamma", CreatedAt: time.Unix(1, 0).UTC().Format(time.RFC3339Nano)},
	}

	sortViews(views, ListQuery{SortBy: "created", SortDirection: "desc"})

	if views[0].ID != "one" || views[1].ID != "two" || views[2].ID != "three" {
		t.Fatalf("sortViews() reordered equal descending keys: %#v", views)
	}
}

func TestServiceListProgramsFiltersBySearchTagAndStatus(t *testing.T) {
	store := &fakeStore{
		entries: []Entry{
			{ID: "one", Name: "Alpha Sync", Description: "nightly", Tags: []string{"ops"}, Path: `C:\tools\a.bat`, Kind: KindBatch, WorkingDirectory: `C:\tools`, RestartPolicy: RestartPolicyNone},
			{ID: "two", Name: "Beta Report", Description: "finance", Tags: []string{"report"}, Path: `C:\tools\b.bat`, Kind: KindBatch, WorkingDirectory: `C:\tools`, RestartPolicy: RestartPolicyNone},
		},
	}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	service.states["one"].status = StatusRunning

	got, err := service.ListPrograms(ListQuery{Search: "sync", Tag: "ops", Status: StatusRunning})
	if err != nil {
		t.Fatalf("ListPrograms() error = %v", err)
	}

	if len(got) != 1 || got[0].ID != "one" {
		t.Fatalf("ListPrograms() = %#v, want only entry one", got)
	}
}

func TestServiceListProgramsPrefiltersBeforeProcessProbe(t *testing.T) {
	store := &fakeStore{
		entries: []Entry{
			{ID: "one", Name: "Alpha Sync", Description: "nightly", Tags: []string{"ops"}, Path: `C:\tools\a.bat`, Kind: KindBatch, WorkingDirectory: `C:\tools`, RestartPolicy: RestartPolicyNone},
			{ID: "two", Name: "Beta Report", Description: "finance", Tags: []string{"report"}, Path: `C:\tools\b.bat`, Kind: KindBatch, WorkingDirectory: `C:\tools`, RestartPolicy: RestartPolicyNone},
		},
	}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	originalProbe := processProbe
	defer func() {
		processProbe = originalProbe
	}()

	probeCalls := 0
	processProbe = func(entry Entry) (int, bool) {
		probeCalls++
		return 0, false
	}

	got, err := service.ListPrograms(ListQuery{Search: "does-not-match"})
	if err != nil {
		t.Fatalf("ListPrograms() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListPrograms() = %#v, want no results", got)
	}
	if probeCalls != 0 {
		t.Fatalf("processProbe calls = %d, want 0 for prefiltered query", probeCalls)
	}
}

func TestServiceGetProgramReturnsRichView(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "run.bat", "@echo off\r\nexit /b 0\r\n")
	created, err := service.CreateProgram(Input{
		Name:        "Runner",
		Description: "desc",
		Notes:       "memo",
		Tags:        []string{"ops"},
		Path:        path,
	})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	got, err := service.GetProgram(created.ID)
	if err != nil {
		t.Fatalf("GetProgram() error = %v", err)
	}

	if got.Description != "desc" || got.Notes != "memo" || len(got.Tags) != 1 {
		t.Fatalf("GetProgram() = %#v", got)
	}
	if got.LaunchMode != "cmd" {
		t.Fatalf("GetProgram().LaunchMode = %q, want %q", got.LaunchMode, "cmd")
	}
}

func TestServiceUpdateProgramPreservesConfiguredRestartFields(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "run.bat", "@echo off\r\nexit /b 0\r\n")
	created, err := service.CreateProgram(Input{Name: "Runner", Path: path})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	updated, err := service.UpdateProgram(created.ID, Input{
		Name:                "Runner",
		Path:                path,
		RunAsAdmin:          true,
		RestartPolicy:       RestartPolicyOnFailure,
		RestartLimit:        3,
		RestartDelaySeconds: 5,
	})
	if err != nil {
		t.Fatalf("UpdateProgram() error = %v", err)
	}

	if updated.RunAsAdmin != true {
		t.Fatal("UpdateProgram().RunAsAdmin = false, want true")
	}
	if updated.RestartPolicy != RestartPolicyOnFailure {
		t.Fatalf("UpdateProgram().RestartPolicy = %q, want %q", updated.RestartPolicy, RestartPolicyOnFailure)
	}
	if updated.RestartLimit != 3 {
		t.Fatalf("UpdateProgram().RestartLimit = %d, want 3", updated.RestartLimit)
	}
	if updated.RestartDelaySeconds != 5 {
		t.Fatalf("UpdateProgram().RestartDelaySeconds = %d, want 5", updated.RestartDelaySeconds)
	}
	if updated.LaunchMode != "elevated" {
		t.Fatalf("UpdateProgram().LaunchMode = %q, want %q", updated.LaunchMode, "elevated")
	}
}

func TestServiceGetProgramLogsReturnsCapturedOutput(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "echo.bat", "@echo off\r\necho hello\r\n")
	entry, err := service.CreateProgram(Input{Name: "Logger", Path: path})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	if err := service.StartProgram(entry.ID); err != nil {
		t.Fatalf("StartProgram() error = %v", err)
	}
	waitForStatus(t, service, entry.ID, StatusStopped)

	logs, err := service.GetProgramLogs(entry.ID, LogQuery{Limit: 20})
	if err != nil {
		t.Fatalf("GetProgramLogs() error = %v", err)
	}

	if len(logs.Entries) == 0 {
		t.Fatal("GetProgramLogs() returned no entries")
	}
}

func TestServiceIncludesRecentStderrInLastError(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, func(entry Entry) *exec.Cmd {
		return exec.Command("cmd", "/C", "echo missing module express 1>&2 & exit 1")
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "stderr-fail.bat", "@echo off\r\nexit /b 0\r\n")
	entry, err := service.CreateProgram(Input{Name: "Broken", Path: path})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	if err := service.StartProgram(entry.ID); err != nil {
		t.Fatalf("StartProgram() error = %v", err)
	}
	waitForStatus(t, service, entry.ID, StatusStopped)

	view, err := service.GetProgram(entry.ID)
	if err != nil {
		t.Fatalf("GetProgram() error = %v", err)
	}

	if !strings.Contains(view.LastError, "exit status 1") {
		t.Fatalf("LastError = %q, want exit status", view.LastError)
	}
	if !strings.Contains(view.LastError, "missing module express") {
		t.Fatalf("LastError = %q, want stderr context", view.LastError)
	}
}

func TestServicePrefersErrorLineOverTrailingStackFrame(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, func(entry Entry) *exec.Cmd {
		return exec.Command("cmd", "/C", "echo internal/modules/cjs/loader.js:613 1>&2 & echo Error: Cannot find module 'express' 1>&2 & echo at Function.Module.runMain 1>&2 & exit 1")
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "stack-fail.bat", "@echo off\r\nexit /b 0\r\n")
	entry, err := service.CreateProgram(Input{Name: "NodeFail", Path: path})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	if err := service.StartProgram(entry.ID); err != nil {
		t.Fatalf("StartProgram() error = %v", err)
	}
	waitForStatus(t, service, entry.ID, StatusStopped)

	view, err := service.GetProgram(entry.ID)
	if err != nil {
		t.Fatalf("GetProgram() error = %v", err)
	}

	if !strings.Contains(view.LastError, "Error: Cannot find module 'express'") {
		t.Fatalf("LastError = %q, want the error line", view.LastError)
	}
}

func TestServiceDecodesKoreanStderrFromWindowsEncoding(t *testing.T) {
	store := &fakeStore{}
	message := "설치 파일이 아닙니다."

	service, err := NewService(store, func(entry Entry) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", "stderr-euckr")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "GO_HELPER_STDERR_TEXT="+message)
		return cmd
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "stderr-euckr.bat", "@echo off\r\nexit /b 0\r\n")
	entry, err := service.CreateProgram(Input{Name: "BrokenKo", Path: path})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	if err := service.StartProgram(entry.ID); err != nil {
		t.Fatalf("StartProgram() error = %v", err)
	}
	waitForStatus(t, service, entry.ID, StatusStopped)

	view, err := service.GetProgram(entry.ID)
	if err != nil {
		t.Fatalf("GetProgram() error = %v", err)
	}
	if !strings.Contains(view.LastError, message) {
		t.Fatalf("LastError = %q, want %q", view.LastError, message)
	}
}

func TestServiceRestartsProgramOnFailureWhenPolicyRequiresIt(t *testing.T) {
	store := &fakeStore{}
	starts := 0
	service, err := NewService(store, func(entry Entry) *exec.Cmd {
		starts++
		return exec.Command("cmd", "/C", "exit", "1")
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "run.bat", "@echo off\r\nexit /b 1\r\n")
	entry, err := service.CreateProgram(Input{
		Name:          "Retry",
		Path:          path,
		RestartPolicy: RestartPolicyOnFailure,
		RestartLimit:  1,
	})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	if err := service.StartProgram(entry.ID); err != nil {
		t.Fatalf("StartProgram() error = %v", err)
	}

	waitForRestartCount(t, service, entry.ID, 1)
	if starts < 2 {
		t.Fatalf("starts = %d, want at least 2", starts)
	}
}

func TestServiceDoesNotRestartProgramAfterUserStop(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, func(entry Entry) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", "sleep")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "run.bat", "@echo off\r\nexit /b 0\r\n")
	entry, err := service.CreateProgram(Input{
		Name:          "Forever",
		Path:          path,
		RestartPolicy: RestartPolicyAlways,
		RestartLimit:  5,
	})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}

	if err := service.StartProgram(entry.ID); err != nil {
		t.Fatalf("StartProgram() error = %v", err)
	}
	waitForStatus(t, service, entry.ID, StatusRunning)

	if err := service.StopProgram(entry.ID); err != nil {
		t.Fatalf("StopProgram() error = %v", err)
	}
	waitForStatus(t, service, entry.ID, StatusStopped)

	time.Sleep(300 * time.Millisecond)
	view, err := service.GetProgram(entry.ID)
	if err != nil {
		t.Fatalf("GetProgram() error = %v", err)
	}
	if view.RestartCount != 0 {
		t.Fatalf("RestartCount = %d, want 0", view.RestartCount)
	}
	if view.Status != StatusStopped {
		t.Fatalf("Status = %q, want %q", view.Status, StatusStopped)
	}
}

func TestServiceReconnectProgramsMarksMissingProcessOrphaned(t *testing.T) {
	store := &fakeStore{
		entries: []Entry{
			{ID: "one", Name: "Runner", Path: `C:\tools\run.bat`, Kind: KindBatch, WorkingDirectory: `C:\tools`, RestartPolicy: RestartPolicyNone},
		},
	}
	runtimeStore := &fakeRuntimeStore{
		state: runtimeDocument{
			Programs: []runtimeEntry{
				{ID: "one", PID: 999999, CanReconnect: true},
			},
		},
	}

	service, err := NewService(store, nil, WithRuntimeStore(runtimeStore), WithProcessLookup(alwaysMissingProcess))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if err := service.ReconnectPrograms(); err != nil {
		t.Fatalf("ReconnectPrograms() error = %v", err)
	}

	view, err := service.GetProgram("one")
	if err != nil {
		t.Fatalf("GetProgram() error = %v", err)
	}
	if view.Status != StatusOrphaned {
		t.Fatalf("Status = %q, want %q", view.Status, StatusOrphaned)
	}
}

func TestServiceDetectsManuallyStartedProgramByPath(t *testing.T) {
	store := &fakeStore{
		entries: []Entry{
			{ID: "one", Name: "Runner", Path: `C:\tools\run.bat`, Kind: KindBatch, WorkingDirectory: `C:\tools`, RestartPolicy: RestartPolicyNone},
		},
	}
	runtimeStore := &fakeRuntimeStore{
		state: runtimeDocument{
			Programs: []runtimeEntry{
				{ID: "one", PID: 0, Path: `C:\tools\run.bat`, CanReconnect: true},
			},
		},
	}

	service, err := NewService(store, nil, WithRuntimeStore(runtimeStore))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	originalLookup := processPathLookup
	originalProbe := processProbe
	originalSnapshot := processSnapshotByPID
	originalKill := killProcessByPID
	defer func() {
		processPathLookup = originalLookup
		processProbe = originalProbe
		processSnapshotByPID = originalSnapshot
		killProcessByPID = originalKill
	}()

	running := true
	killedPID := 0
	snapshotTime := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	processPathLookup = func(path string) (int, bool) {
		if running && strings.EqualFold(path, `C:\tools\run.bat`) {
			return 4242, true
		}
		return 0, false
	}
	processProbe = func(entry Entry) (int, bool) {
		if running && strings.EqualFold(entry.Path, `C:\tools\run.bat`) {
			return 4242, true
		}
		return 0, false
	}
	processSnapshotByPID = func(pid int) (processSnapshot, bool) {
		if !running || pid != 4242 {
			return processSnapshot{}, false
		}
		return processSnapshot{
			startedAt:             snapshotTime,
			memoryWorkingSetBytes: 128 * 1024 * 1024,
			memoryPrivateBytes:    64 * 1024 * 1024,
		}, true
	}
	killProcessByPID = func(pid int) error {
		killedPID = pid
		running = false
		return nil
	}

	if err := service.ReconnectPrograms(); err != nil {
		t.Fatalf("ReconnectPrograms() error = %v", err)
	}

	view, err := service.GetProgram("one")
	if err != nil {
		t.Fatalf("GetProgram() error = %v", err)
	}
	if view.Status != StatusRunning {
		t.Fatalf("Status = %q, want %q", view.Status, StatusRunning)
	}
	if view.PID != 4242 {
		t.Fatalf("PID = %d, want %d", view.PID, 4242)
	}
	if view.StartedAt != snapshotTime.Format(time.RFC3339Nano) {
		t.Fatalf("StartedAt = %q, want %q", view.StartedAt, snapshotTime.Format(time.RFC3339Nano))
	}
	if view.MemoryWorkingSetBytes != 128*1024*1024 {
		t.Fatalf("MemoryWorkingSetBytes = %d, want %d", view.MemoryWorkingSetBytes, 128*1024*1024)
	}
	if view.MemoryPrivateBytes != 64*1024*1024 {
		t.Fatalf("MemoryPrivateBytes = %d, want %d", view.MemoryPrivateBytes, 64*1024*1024)
	}

	if err := service.StopProgram("one"); err != nil {
		t.Fatalf("StopProgram() error = %v", err)
	}
	if killedPID != 4242 {
		t.Fatalf("killedPID = %d, want %d", killedPID, 4242)
	}

	view, err = service.GetProgram("one")
	if err != nil {
		t.Fatalf("GetProgram() after stop error = %v", err)
	}
	if view.Status != StatusStopped {
		t.Fatalf("Status after stop = %q, want %q", view.Status, StatusStopped)
	}
}

func TestServiceUsesProcessProbeForWrappedBatchPrograms(t *testing.T) {
	store := &fakeStore{
		entries: []Entry{
			{ID: "one", Name: "Runner", Path: `C:\tools\run.bat`, Kind: KindBatch, WorkingDirectory: `C:\tools`, RestartPolicy: RestartPolicyNone},
		},
	}
	service, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	originalProbe := processProbe
	defer func() {
		processProbe = originalProbe
	}()

	processProbe = func(entry Entry) (int, bool) {
		if entry.Kind == KindBatch && entry.Path == `C:\tools\run.bat` {
			return 5150, true
		}
		return 0, false
	}

	service.mu.Lock()
	service.states["one"].status = StatusOrphaned
	service.states["one"].canReconnect = true
	service.mu.Unlock()

	view, err := service.GetProgram("one")
	if err != nil {
		t.Fatalf("GetProgram() error = %v", err)
	}
	if view.Status != StatusRunning {
		t.Fatalf("Status = %q, want %q", view.Status, StatusRunning)
	}
	if view.PID != 5150 {
		t.Fatalf("PID = %d, want 5150", view.PID)
	}
	if view.LaunchMode != "cmd" {
		t.Fatalf("LaunchMode = %q, want %q", view.LaunchMode, "cmd")
	}
}

func TestServiceClassifiesStopPermissionErrorForReconnectedProcess(t *testing.T) {
	store := &fakeStore{}
	service, err := NewService(store, func(entry Entry) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", "sleep")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	path := createBatchFile(t, "run.bat", "@echo off\r\nexit /b 0\r\n")
	entry, err := service.CreateProgram(Input{Name: "Runner", Path: path})
	if err != nil {
		t.Fatalf("CreateProgram() error = %v", err)
	}
	if err := service.StartProgram(entry.ID); err != nil {
		t.Fatalf("StartProgram() error = %v", err)
	}
	waitForStatus(t, service, entry.ID, StatusRunning)

	service.mu.Lock()
	state := service.ensureStateLocked(entry.ID)
	process := state.cmd.Process
	state.cmd = nil
	service.mu.Unlock()

	originalKill := killProcessByPID
	defer func() {
		killProcessByPID = originalKill
		_ = process.Kill()
	}()

	killProcessByPID = func(pid int) error {
		return errors.New("TerminateProcess: Access is denied.")
	}

	err = service.StopProgram(entry.ID)
	if err == nil {
		t.Fatal("StopProgram() error = nil, want permission error")
	}

	view, getErr := service.GetProgram(entry.ID)
	if getErr != nil {
		t.Fatalf("GetProgram() error = %v", getErr)
	}
	if view.Status != StatusOrphaned {
		t.Fatalf("Status = %q, want %q", view.Status, StatusOrphaned)
	}
	if !strings.Contains(view.LastError, "접근이 거부되었습니다") {
		t.Fatalf("LastError = %q, want permission message", view.LastError)
	}
}

func createBatchFile(t *testing.T, name string, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := writeTextFile(path, content); err != nil {
		t.Fatalf("writeTextFile() error = %v", err)
	}
	return path
}

func createFile(t *testing.T, name string, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := writeTextFile(path, content); err != nil {
		t.Fatalf("writeTextFile() error = %v", err)
	}
	return path
}

func waitForStatus(t *testing.T, service *Service, id string, want string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		views, err := service.ListPrograms(ListQuery{})
		if err != nil {
			t.Fatalf("ListPrograms() error = %v", err)
		}

		for _, view := range views {
			if view.ID == id && view.Status == want {
				return
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("status for %q did not become %q in time", id, want)
}

func waitForRestartCount(t *testing.T, service *Service, id string, want int) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		view, err := service.GetProgram(id)
		if err != nil {
			t.Fatalf("GetProgram() error = %v", err)
		}
		if view.RestartCount >= want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("restart count for %q did not become %d in time", id, want)
}

func containsEnvValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func parseTimeOrZero(t *testing.T, value string) time.Time {
	t.Helper()

	if value == "" {
		return time.Time{}
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatalf("time.Parse(%q) error = %v", value, err)
	}

	return parsed
}
