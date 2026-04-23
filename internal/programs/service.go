package programs

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/text/encoding/korean"
)

const appConfigDirName = "mycms"

type CommandFactory func(entry Entry) *exec.Cmd
type ProcessLookupFunc func(runtimeEntry) bool
type Option func(*Service)

var killProcessByPID = func(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}

var processPathLookup = probeProcessByPath

type Service struct {
	mu             sync.RWMutex
	store          Store
	runtimeStore   RuntimeStore
	entries        []Entry
	states         map[string]*programState
	commandFactory CommandFactory
	processLookup  ProcessLookupFunc
}

type programState struct {
	cmd          *exec.Cmd
	status       string
	lastError    string
	stopping     bool
	pid          int
	startedAt    time.Time
	lastExitAt   time.Time
	restartCount int
	elevated     bool
	canReconnect bool
	lastLogAt    time.Time
	logs         *logBuffer
}

type processProbeResult struct {
	pid int
	ok  bool
}

type processScanContext struct {
	candidates []processCandidate
	candidateByPID map[int]processCandidate
	candidatesByBase map[string][]processCandidate
	candidatesByImagePath map[string][]processCandidate
	probes     map[string]processProbeResult
	snapshots  map[int]processSnapshot
}

type lineLogWriter struct {
	service *Service
	id      string
	stream  string
	mu      sync.Mutex
	pending string
}

func WithRuntimeStore(store RuntimeStore) Option {
	return func(s *Service) {
		s.runtimeStore = store
	}
}

func WithProcessLookup(fn ProcessLookupFunc) Option {
	return func(s *Service) {
		s.processLookup = fn
	}
}

func NewService(store Store, commandFactory CommandFactory, opts ...Option) (*Service, error) {
	entries, err := store.Load()
	if err != nil {
		return nil, err
	}

	service := &Service{
		store:          store,
		entries:        make([]Entry, len(entries)),
		states:         make(map[string]*programState, len(entries)),
		commandFactory: commandFactory,
	}
	copy(service.entries, entries)

	if service.commandFactory == nil {
		service.commandFactory = defaultCommandFactory
	}
	if service.processLookup == nil {
		service.processLookup = defaultProcessLookup
	}

	for _, opt := range opts {
		opt(service)
	}

	for _, entry := range service.entries {
		service.states[entry.ID] = &programState{status: StatusStopped}
	}

	return service, nil
}

func (s *Service) ListPrograms(query ListQuery) ([]View, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newProcessScanContext()
	views := make([]View, 0, len(s.entries))
	for _, entry := range s.entries {
		state := s.stateForEntryLocked(entry.ID)
		if !matchesEntryQuery(entry, state, query) {
			continue
		}

		view := s.viewForEntryLocked(entry, ctx)
		if !matchesQuery(view, query) {
			continue
		}
		views = append(views, view)
	}

	sortViews(views, query)
	return views, nil
}

func (s *Service) GetProgram(id string) (View, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return View{}, fmt.Errorf("프로그램 %q을(를) 찾을 수 없습니다", id)
	}

	return s.viewForEntryLocked(s.entries[index], newProcessScanContext()), nil
}

func (s *Service) ReconnectPrograms() error {
	if s.runtimeStore == nil {
		return nil
	}

	document, err := s.runtimeStore.Load()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newProcessScanContext()
	for _, item := range document.Programs {
		index := s.indexByIDLocked(item.ID)
		if index < 0 {
			continue
		}

		entry := s.entries[index]
		state := s.ensureStateLocked(item.ID)
		if item.CanReconnect {
			if pid, ok := ctx.probe(entry); ok {
				s.applyRunningStateLocked(state, entry, pid, item.StartedAt, true)
				continue
			}
		}

		if item.CanReconnect && item.PID > 0 && s.processLookup(item) && ctx.matchesPID(entry, item.PID) {
			s.applyRunningStateLocked(state, entry, item.PID, item.StartedAt, false)
			continue
		}

		state.cmd = nil
		state.status = StatusOrphaned
		state.canReconnect = false
		state.pid = 0
		state.elevated = false
		state.lastError = "프로세스를 다시 연결할 수 없습니다"
	}

	return s.persistRuntimeLocked()
}

func (s *Service) CreateProgram(input Input) (View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, err := s.normalizeInputLocked("", input)
	if err != nil {
		return View{}, err
	}

	entry.ID = uuid.NewString()
	entry.CreatedAt = entry.UpdatedAt
	s.entries = append(s.entries, entry)
	s.states[entry.ID] = &programState{status: StatusStopped}

	if err := s.persistLocked(); err != nil {
		delete(s.states, entry.ID)
		s.entries = s.entries[:len(s.entries)-1]
		return View{}, err
	}
	if err := s.persistRuntimeLocked(); err != nil {
		return View{}, err
	}

	return s.viewForEntryLocked(entry, newProcessScanContext()), nil
}

func (s *Service) UpdateProgram(id string, input Input) (View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return View{}, fmt.Errorf("프로그램 %q을(를) 찾을 수 없습니다", id)
	}

	if state := s.ensureStateLocked(id); state.cmd != nil {
		return View{}, errors.New("수정하기 전에 프로그램을 먼저 중지하세요")
	}

	entry, err := s.normalizeInputLocked(id, input)
	if err != nil {
		return View{}, err
	}
	entry.ID = id
	entry.CreatedAt = s.entries[index].CreatedAt

	s.entries[index] = entry
	if err := s.persistLocked(); err != nil {
		return View{}, err
	}
	if err := s.persistRuntimeLocked(); err != nil {
		return View{}, err
	}

	return s.viewForEntryLocked(entry, newProcessScanContext()), nil
}

func (s *Service) DeleteProgram(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return fmt.Errorf("프로그램 %q을(를) 찾을 수 없습니다", id)
	}

	if state := s.ensureStateLocked(id); state.cmd != nil {
		return errors.New("삭제하기 전에 프로그램을 먼저 중지하세요")
	}

	s.entries = append(s.entries[:index], s.entries[index+1:]...)
	delete(s.states, id)

	if err := s.persistLocked(); err != nil {
		return err
	}
	return s.persistRuntimeLocked()
}

func (s *Service) StartProgram(id string) error {
	s.mu.Lock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		s.mu.Unlock()
		return fmt.Errorf("프로그램 %q을(를) 찾을 수 없습니다", id)
	}

	entry := s.entries[index]
	state := s.ensureStateLocked(id)
	if state.cmd != nil {
		s.mu.Unlock()
		return errors.New("프로그램이 이미 실행 중입니다")
	}

	cmd := s.commandFactory(entry)
	if cmd == nil {
		state.lastError = "명령 생성기가 nil을 반환했습니다"
		s.mu.Unlock()
		return errors.New(state.lastError)
	}

	if cmd.Dir == "" {
		cmd.Dir = filepath.Dir(entry.Path)
	}
	applyCommandEnvironment(cmd, entry.Env)

	stdoutWriter := &lineLogWriter{service: s, id: entry.ID, stream: "stdout"}
	stderrWriter := &lineLogWriter{service: s, id: entry.ID, stream: "stderr"}
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	state.status = StatusStarting
	if err := cmd.Start(); err != nil {
		state.status = StatusStopped
		state.lastError = err.Error()
		s.mu.Unlock()
		return err
	}

	state.cmd = cmd
	state.pid = cmd.Process.Pid
	state.startedAt = time.Now().UTC()
	state.lastExitAt = time.Time{}
	state.status = StatusRunning
	state.lastError = ""
	state.stopping = false
	state.canReconnect = true
	state.elevated = entry.RunAsAdmin
	state.logs = newLogBuffer(500)
	if err := s.persistRuntimeLocked(); err != nil {
		state.lastError = err.Error()
	}
	s.mu.Unlock()

	go s.waitForExit(entry.ID, cmd, stdoutWriter, stderrWriter)
	return nil
}

func (s *Service) StopProgram(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return fmt.Errorf("프로그램 %q을(를) 찾을 수 없습니다", id)
	}

	entry := s.entries[index]
	state := s.ensureStateLocked(entry.ID)
	if state.pid <= 0 {
		ctx := newProcessScanContext()
		if pid, ok := ctx.probe(entry); ok {
			state.pid = pid
		}
	}

	if state.cmd != nil && state.cmd.Process != nil {
		state.status = StatusStopping
		state.stopping = true
		if err := state.cmd.Process.Kill(); err != nil {
			state.status = StatusRunning
			state.stopping = false
			state.lastError = err.Error()
			return err
		}

		return nil
	}

	if state.pid <= 0 {
		return errors.New("프로그램이 실행 중이 아닙니다")
	}

	state.status = StatusStopping
	state.stopping = true
	if err := killProcessByPID(state.pid); err != nil {
		if !defaultProcessLookup(runtimeEntry{PID: state.pid}) {
			state.lastExitAt = time.Now().UTC()
			s.applyStoppedStateLocked(state, "", false)
			if err := s.persistRuntimeLocked(); err != nil {
				state.lastError = err.Error()
			}
			return nil
		}

		state.status = StatusOrphaned
		state.stopping = false
		state.lastError = classifyStopError(err)
		return err
	}

	state.lastExitAt = time.Now().UTC()
	s.applyStoppedStateLocked(state, "", false)
	if err := s.persistRuntimeLocked(); err != nil {
		state.lastError = err.Error()
	}
	return nil
}

func (s *Service) waitForExit(id string, cmd *exec.Cmd, stdoutWriter *lineLogWriter, stderrWriter *lineLogWriter) {
	waitErr := cmd.Wait()
	stdoutWriter.Flush()
	stderrWriter.Flush()

	s.mu.Lock()
	state := s.ensureStateLocked(id)
	if state.cmd != cmd {
		s.mu.Unlock()
		return
	}

	state.cmd = nil
	state.pid = 0
	state.lastExitAt = time.Now().UTC()
	state.canReconnect = false
	state.elevated = false

	entryIndex := s.indexByIDLocked(id)
	if entryIndex < 0 {
		s.mu.Unlock()
		return
	}
	entry := s.entries[entryIndex]

	if s.shouldRestartLocked(entry, state, waitErr) {
		state.restartCount++
		state.status = StatusStopped
		state.stopping = false
		if err := s.persistRuntimeLocked(); err != nil {
			state.lastError = err.Error()
		}
		delay := time.Duration(entry.RestartDelaySeconds) * time.Second
		s.mu.Unlock()
		time.AfterFunc(delay, func() {
			_ = s.StartProgram(id)
		})
		return
	}

	if waitErr != nil && !state.stopping {
		s.applyStoppedStateLocked(state, summarizeProcessExit(waitErr, state.logs), false)
	} else {
		s.applyStoppedStateLocked(state, "", false)
	}
	if err := s.persistRuntimeLocked(); err != nil {
		state.lastError = err.Error()
	}
	s.mu.Unlock()
}

func summarizeProcessExit(waitErr error, logs *logBuffer) string {
	if waitErr == nil {
		return ""
	}

	message := waitErr.Error()
	if logs == nil {
		return message
	}

	stderrLine := logs.LastNonEmptyLine("stderr")
	if stderrLine != "" && !strings.Contains(message, stderrLine) {
		return message + ": " + stderrLine
	}

	stdoutLine := logs.LastNonEmptyLine("stdout")
	if stdoutLine != "" && !strings.Contains(message, stdoutLine) {
		return message + ": " + stdoutLine
	}

	return message
}

func (s *Service) normalizeInputLocked(excludeID string, input Input) (Entry, error) {
	now := time.Now().UTC()
	name := strings.TrimSpace(input.Name)
	path := strings.TrimSpace(input.Path)
	workingDirectory := strings.TrimSpace(input.WorkingDirectory)

	if name == "" {
		return Entry{}, errors.New("이름은 필수입니다")
	}
	if path == "" {
		return Entry{}, errors.New("경로는 필수입니다")
	}
	if !filepath.IsAbs(path) {
		return Entry{}, errors.New("경로는 절대 경로여야 합니다")
	}
	if info, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Entry{}, errors.New("경로가 존재하지 않습니다")
		}
		return Entry{}, err
	} else if info.IsDir() {
		return Entry{}, errors.New("경로는 파일을 가리켜야 합니다")
	}
	kind, err := detectKind(path)
	if err != nil {
		return Entry{}, err
	}
	if workingDirectory == "" {
		workingDirectory = filepath.Dir(path)
	}
	if stat, err := os.Stat(workingDirectory); err != nil || !stat.IsDir() {
		return Entry{}, errors.New("작업 디렉터리는 존재하는 디렉터리여야 합니다")
	}
	if err := validateEnv(input.Env); err != nil {
		return Entry{}, err
	}

	for _, entry := range s.entries {
		if entry.ID == excludeID {
			continue
		}
		if strings.EqualFold(entry.Name, name) {
			return Entry{}, errors.New("이름은 고유해야 합니다")
		}
	}

	return Entry{
		Name:                name,
		Description:         strings.TrimSpace(input.Description),
		Notes:               strings.TrimSpace(input.Notes),
		Tags:                normalizeTags(input.Tags),
		Path:                path,
		Kind:                kind,
		WorkingDirectory:    workingDirectory,
		Args:                append([]string(nil), input.Args...),
		Env:                 cloneEnv(input.Env),
		RunAsAdmin:          input.RunAsAdmin,
		RestartPolicy:       normalizeRestartPolicy(input.RestartPolicy),
		RestartLimit:        maxInt(input.RestartLimit, 0),
		RestartDelaySeconds: maxInt(input.RestartDelaySeconds, 0),
		UpdatedAt:           now,
	}, nil
}

func (s *Service) persistLocked() error {
	entries := make([]Entry, len(s.entries))
	copy(entries, s.entries)
	return s.store.Save(entries)
}

func (s *Service) persistRuntimeLocked() error {
	if s.runtimeStore == nil {
		return nil
	}

	document := runtimeDocument{Programs: make([]runtimeEntry, 0, len(s.entries))}
	for _, entry := range s.entries {
		state := s.ensureStateLocked(entry.ID)
		if state.pid == 0 && !state.canReconnect {
			continue
		}

		document.Programs = append(document.Programs, runtimeEntry{
			ID:           entry.ID,
			PID:          state.pid,
			Path:         entry.Path,
			StartedAt:    state.startedAt,
			CanReconnect: state.canReconnect,
		})
	}

	return s.runtimeStore.Save(document)
}

func (s *Service) indexByIDLocked(id string) int {
	for index, entry := range s.entries {
		if entry.ID == id {
			return index
		}
	}
	return -1
}

func (s *Service) stateForEntryLocked(id string) *programState {
	return s.states[id]
}

func (s *Service) ensureStateLocked(id string) *programState {
	state, ok := s.states[id]
	if ok {
		return state
	}

	state = &programState{status: StatusStopped}
	s.states[id] = state
	return state
}

func (s *Service) shouldRestartLocked(entry Entry, state *programState, waitErr error) bool {
	if state.stopping {
		return false
	}
	if state.restartCount >= entry.RestartLimit {
		return false
	}

	switch entry.RestartPolicy {
	case RestartPolicyAlways:
		return true
	case RestartPolicyOnFailure:
		return waitErr != nil
	default:
		return false
	}
}

func newProcessScanContext() *processScanContext {
	candidates := processCandidatesProvider()
	candidateByPID := make(map[int]processCandidate, len(candidates))
	candidatesByBase := make(map[string][]processCandidate, 8)
	candidatesByImagePath := make(map[string][]processCandidate, len(candidates))
	for _, candidate := range candidates {
		candidateByPID[candidate.pid] = candidate
		base := candidateBaseName(candidate)
		candidatesByBase[base] = append(candidatesByBase[base], candidate)
		imagePath := normalizeComparablePath(candidate.imagePath)
		if imagePath != "" {
			candidatesByImagePath[imagePath] = append(candidatesByImagePath[imagePath], candidate)
		}
	}

	return &processScanContext{
		candidates: candidates,
		candidateByPID: candidateByPID,
		candidatesByBase: candidatesByBase,
		candidatesByImagePath: candidatesByImagePath,
		probes:     make(map[string]processProbeResult),
		snapshots:  make(map[int]processSnapshot),
	}
}

func (c *processScanContext) probe(entry Entry) (int, bool) {
	if cached, ok := c.probes[entry.ID]; ok {
		return cached.pid, cached.ok
	}

	if len(c.candidates) > 0 {
		for _, candidate := range c.candidatesForEntry(entry) {
			if matchProcessCandidate(entry, candidate) {
				result := processProbeResult{pid: candidate.pid, ok: true}
				c.probes[entry.ID] = result
				return result.pid, true
			}
		}
	}

	pid, ok := processProbe(entry)
	c.probes[entry.ID] = processProbeResult{pid: pid, ok: ok}
	return pid, ok
}

func (c *processScanContext) snapshot(pid int) (processSnapshot, bool) {
	if cached, ok := c.snapshots[pid]; ok {
		return cached, true
	}

	snapshot, ok := processSnapshotByPID(pid)
	if ok {
		c.snapshots[pid] = snapshot
	}
	return snapshot, ok
}

func (c *processScanContext) candidatesForEntry(entry Entry) []processCandidate {
	switch entry.Kind {
	case KindExecutable:
		path := normalizeComparablePath(entry.Path)
		if path == "" {
			return nil
		}
		return c.candidatesByImagePath[path]
	case KindBatch, KindCommand:
		return c.candidatesByBase["cmd.exe"]
	case KindPowerShell:
		candidates := append([]processCandidate(nil), c.candidatesByBase["powershell.exe"]...)
		candidates = append(candidates, c.candidatesByBase["pwsh.exe"]...)
		return candidates
	default:
		return c.candidates
	}
}

func (c *processScanContext) matchesPID(entry Entry, pid int) bool {
	candidate, ok := c.candidateByPID[pid]
	if !ok {
		return true
	}
	return matchProcessCandidate(entry, candidate)
}

func (s *Service) shouldProbeEntryLocked(state *programState) bool {
	if state == nil {
		return false
	}
	if state.cmd != nil || state.pid > 0 || state.canReconnect {
		return true
	}
	switch state.status {
	case StatusRunning, StatusStarting, StatusStopping, StatusOrphaned:
		return true
	default:
		return false
	}
}

func (s *Service) applyRunningStateLocked(state *programState, entry Entry, pid int, startedAt time.Time, managed bool) {
	state.cmd = nil
	state.pid = pid
	state.startedAt = startedAt
	state.status = StatusRunning
	state.lastError = ""
	state.canReconnect = true
	state.elevated = entry.RunAsAdmin && managed
	state.stopping = false
}

func (s *Service) applyStoppedStateLocked(state *programState, lastError string, keepReconnect bool) {
	state.cmd = nil
	state.pid = 0
	state.canReconnect = keepReconnect
	state.elevated = false
	state.status = StatusStopped
	state.stopping = false
	state.lastError = lastError
	if state.lastExitAt.IsZero() {
		state.lastExitAt = time.Now().UTC()
	}
}

func classifyStopError(err error) string {
	if err == nil {
		return ""
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "access is denied") {
		return "프로세스를 종료할 수 없습니다. 접근이 거부되었습니다."
	}
	return err.Error()
}

func (s *Service) viewForEntryLocked(entry Entry, ctx *processScanContext) View {
	state := s.stateForEntryLocked(entry.ID)
	if state == nil {
		state = &programState{status: StatusStopped}
	}
	status := state.status
	pid := state.pid
	canReconnect := state.canReconnect
	lastError := state.lastError
	startedAt := state.startedAt
	launchMode := launchModeForEntry(entry)
	memoryWorkingSetBytes := int64(0)
	memoryPrivateBytes := int64(0)

	if s.shouldProbeEntryLocked(state) {
		if pid <= 0 {
			if foundPID, ok := ctx.probe(entry); ok {
				pid = foundPID
				if status == StatusStopped || status == StatusOrphaned {
					status = StatusRunning
					canReconnect = true
					lastError = ""
				}
			}
		}

		if pid > 0 && (status == StatusRunning || status == StatusStarting || status == StatusStopping) {
			if snapshot, ok := ctx.snapshot(pid); ok {
				if !snapshot.startedAt.IsZero() {
					startedAt = snapshot.startedAt
				}
				memoryWorkingSetBytes = snapshot.memoryWorkingSetBytes
				memoryPrivateBytes = snapshot.memoryPrivateBytes
			}
		}
	}

	return View{
		ID:                    entry.ID,
		Name:                  entry.Name,
		Description:           entry.Description,
		Notes:                 entry.Notes,
		Tags:                  append([]string(nil), entry.Tags...),
		Path:                  entry.Path,
		Kind:                  entry.Kind,
		LaunchMode:            launchMode,
		WorkingDirectory:      entry.WorkingDirectory,
		Args:                  append([]string(nil), entry.Args...),
		Env:                   append([]EnvVar(nil), entry.Env...),
		RunAsAdmin:            entry.RunAsAdmin,
		RestartPolicy:         entry.RestartPolicy,
		RestartLimit:          entry.RestartLimit,
		RestartDelaySeconds:   entry.RestartDelaySeconds,
		Status:                status,
		LastError:             lastError,
		PID:                   pid,
		StartedAt:             formatTimestamp(startedAt),
		LastExitAt:            formatTimestamp(state.lastExitAt),
		MemoryWorkingSetBytes: memoryWorkingSetBytes,
		MemoryPrivateBytes:    memoryPrivateBytes,
		RestartCount:          state.restartCount,
		Elevated:              state.elevated,
		CanReconnect:          canReconnect,
		LastLogAt:             formatTimestamp(state.lastLogAt),
		CreatedAt:             formatTimestamp(entry.CreatedAt),
		UpdatedAt:             formatTimestamp(entry.UpdatedAt),
	}
}

func defaultCommandFactory(entry Entry) *exec.Cmd {
	cmd, _, err := buildCommand(entry)
	if err != nil {
		return nil
	}
	return cmd
}

func validateEnv(env []EnvVar) error {
	seen := make(map[string]struct{}, len(env))
	for _, item := range env {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			return errors.New("환경 변수 키는 필수입니다")
		}
		upperKey := strings.ToUpper(key)
		if _, exists := seen[upperKey]; exists {
			return errors.New("환경 변수 키는 중복될 수 없습니다")
		}
		seen[upperKey] = struct{}{}
	}
	return nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}

		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func cloneEnv(env []EnvVar) []EnvVar {
	if len(env) == 0 {
		return []EnvVar{}
	}

	result := make([]EnvVar, 0, len(env))
	for _, item := range env {
		result = append(result, EnvVar{
			Key:   strings.TrimSpace(item.Key),
			Value: item.Value,
		})
	}
	return result
}

func normalizeRestartPolicy(policy string) string {
	switch strings.TrimSpace(strings.ToLower(policy)) {
	case RestartPolicyOnFailure:
		return RestartPolicyOnFailure
	case RestartPolicyAlways:
		return RestartPolicyAlways
	case "", RestartPolicyNone:
		return RestartPolicyNone
	default:
		return RestartPolicyNone
	}
}

func maxInt(value int, floor int) int {
	if value < floor {
		return floor
	}
	return value
}

func (s *Service) GetProgramLogs(id string, query LogQuery) (LogView, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return LogView{}, fmt.Errorf("프로그램 %q을(를) 찾을 수 없습니다", id)
	}

	state := s.stateForEntryLocked(id)
	if state == nil || state.logs == nil {
		return LogView{
			ProgramID: id,
			Entries:   []LogEntry{},
		}, nil
	}

	view := state.logs.View(query)
	view.ProgramID = id
	return view, nil
}

func (s *Service) ClearProgramLogs(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return fmt.Errorf("프로그램 %q을(를) 찾을 수 없습니다", id)
	}

	state := s.ensureStateLocked(s.entries[index].ID)
	state.logs = newLogBuffer(500)
	state.lastLogAt = time.Time{}
	return nil
}

func matchesEntryQuery(entry Entry, state *programState, query ListQuery) bool {
	search := strings.TrimSpace(strings.ToLower(query.Search))
	if search != "" {
		haystacks := []string{
			entry.Name,
			entry.Description,
			entry.Notes,
			entry.Path,
			strings.Join(entry.Tags, " "),
		}

		matched := false
		for _, candidate := range haystacks {
			if strings.Contains(strings.ToLower(candidate), search) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if status := strings.TrimSpace(query.Status); status != "" {
		currentStatus := StatusStopped
		if state != nil && state.status != "" {
			currentStatus = state.status
		}
		if !strings.EqualFold(currentStatus, status) {
			return false
		}
	}

	if tag := strings.TrimSpace(strings.ToLower(query.Tag)); tag != "" {
		matched := false
		for _, current := range entry.Tags {
			if strings.EqualFold(current, tag) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

func matchesQuery(view View, query ListQuery) bool {
	search := strings.TrimSpace(strings.ToLower(query.Search))
	if search != "" {
		haystacks := []string{
			view.Name,
			view.Description,
			view.Notes,
			view.Path,
			strings.Join(view.Tags, " "),
		}

		matched := false
		for _, candidate := range haystacks {
			if strings.Contains(strings.ToLower(candidate), search) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if status := strings.TrimSpace(query.Status); status != "" && !strings.EqualFold(view.Status, status) {
		return false
	}

	if tag := strings.TrimSpace(strings.ToLower(query.Tag)); tag != "" {
		matched := false
		for _, current := range view.Tags {
			if strings.EqualFold(current, tag) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

func sortViews(views []View, query ListQuery) {
	sortBy := strings.TrimSpace(strings.ToLower(query.SortBy))
	desc := strings.EqualFold(strings.TrimSpace(query.SortDirection), "desc")

	sort.SliceStable(views, func(i int, j int) bool {
		comparison := compareViewValues(views[i], views[j], sortBy)
		if desc {
			return comparison > 0
		}
		return comparison < 0
	})
}

func compareViewValues(left View, right View, sortBy string) int {
	switch sortBy {
	case "status":
		return strings.Compare(strings.ToLower(left.Status), strings.ToLower(right.Status))
	case "created":
		return compareTimestamp(left.CreatedAt, right.CreatedAt)
	case "updated":
		return compareTimestamp(left.UpdatedAt, right.UpdatedAt)
	case "laststarted":
		return compareTimestamp(left.StartedAt, right.StartedAt)
	default:
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	}
}

func compareTimestamp(left string, right string) int {
	switch {
	case left == right:
		return 0
	case left == "":
		return -1
	case right == "":
		return 1
	default:
		return strings.Compare(left, right)
	}
}

func (w *lineLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	chunk := strings.ReplaceAll(decodeConsoleBytes(p), "\r\n", "\n")
	chunk = strings.ReplaceAll(chunk, "\r", "\n")
	w.pending += chunk

	for {
		newlineIndex := strings.IndexByte(w.pending, '\n')
		if newlineIndex < 0 {
			break
		}

		line := w.pending[:newlineIndex]
		w.pending = w.pending[newlineIndex+1:]
		w.service.appendLogLine(w.id, w.stream, line)
	}

	return len(p), nil
}

func decodeConsoleBytes(p []byte) string {
	if len(p) == 0 {
		return ""
	}
	if utf8.Valid(p) {
		return string(p)
	}

	decoded, err := korean.EUCKR.NewDecoder().Bytes(p)
	if err == nil && utf8.Valid(decoded) {
		return string(decoded)
	}

	return string(p)
}

func (w *lineLogWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.pending == "" {
		return
	}

	w.service.appendLogLine(w.id, w.stream, w.pending)
	w.pending = ""
}

func (s *Service) appendLogLine(id string, stream string, line string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.ensureStateLocked(id)
	if state.logs == nil {
		state.logs = newLogBuffer(500)
	}

	now := time.Now().UTC()
	state.logs.Append(stream, line, now)
	state.lastLogAt = now
}

func applyCommandEnvironment(cmd *exec.Cmd, env []EnvVar) {
	if len(env) == 0 {
		return
	}

	base := cmd.Env
	if len(base) == 0 {
		base = os.Environ()
	}

	merged := make([]string, 0, len(base)+len(env))
	indexByKey := make(map[string]int, len(base))
	for _, item := range base {
		key := envKeyName(item)
		indexByKey[key] = len(merged)
		merged = append(merged, item)
	}

	for _, item := range env {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}

		encoded := key + "=" + item.Value
		upper := strings.ToUpper(key)
		if index, ok := indexByKey[upper]; ok {
			merged[index] = encoded
			continue
		}

		indexByKey[upper] = len(merged)
		merged = append(merged, encoded)
	}

	cmd.Env = merged
}

func envKeyName(item string) string {
	key, _, ok := strings.Cut(item, "=")
	if !ok {
		return strings.ToUpper(strings.TrimSpace(item))
	}
	return strings.ToUpper(strings.TrimSpace(key))
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(time.RFC3339Nano)
}
