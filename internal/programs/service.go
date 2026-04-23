package programs

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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

	views := make([]View, 0, len(s.entries))
	for _, entry := range s.entries {
		view := s.viewForEntryLocked(entry)
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

	return s.viewForEntryLocked(s.entries[index]), nil
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

	for _, item := range document.Programs {
		index := s.indexByIDLocked(item.ID)
		if index < 0 {
			continue
		}

		state := s.ensureStateLocked(item.ID)
		if item.CanReconnect {
			if pid, ok := processPathLookup(item.Path); ok {
				state.cmd = nil
				state.pid = pid
				state.startedAt = item.StartedAt
				state.status = StatusRunning
				state.lastError = ""
				state.canReconnect = true
				state.elevated = false
				continue
			}
		}

		if item.CanReconnect && s.processLookup(item) {
			state.cmd = nil
			state.pid = item.PID
			state.startedAt = item.StartedAt
			state.status = StatusRunning
			state.lastError = ""
			state.canReconnect = true
			state.elevated = false
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

	return s.viewForEntryLocked(entry), nil
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

	return s.viewForEntryLocked(entry), nil
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
		return errors.New("program is already running")
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

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		state.status = StatusStopped
		state.lastError = err.Error()
		s.mu.Unlock()
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		state.status = StatusStopped
		state.lastError = err.Error()
		s.mu.Unlock()
		return err
	}

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

	go s.capturePipe(entry.ID, "stdout", stdout)
	go s.capturePipe(entry.ID, "stderr", stderr)
	go s.waitForExit(entry.ID, cmd)
	return nil
}

func (s *Service) StopProgram(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByIDLocked(id)
	if index < 0 {
		return fmt.Errorf("program %q was not found", id)
	}

	entry := s.entries[index]
	state := s.ensureStateLocked(entry.ID)
	if state.pid <= 0 {
		if pid, ok := processPathLookup(entry.Path); ok {
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
		state.status = StatusOrphaned
		state.stopping = false
		state.lastError = err.Error()
		return err
	}

	state.cmd = nil
	state.pid = 0
	state.canReconnect = false
	state.elevated = false
	state.lastExitAt = time.Now().UTC()
	state.status = StatusStopped
	state.stopping = false
	state.lastError = ""
	if err := s.persistRuntimeLocked(); err != nil {
		state.lastError = err.Error()
	}
	return nil
}

func (s *Service) waitForExit(id string, cmd *exec.Cmd) {
	waitErr := cmd.Wait()

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

	state.status = StatusStopped

	if waitErr != nil && !state.stopping {
		state.lastError = summarizeProcessExit(waitErr, state.logs)
	}
	if state.stopping {
		state.lastError = ""
	}
	state.stopping = false
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
		return Entry{}, errors.New("작업 디렉터리는 존재하는 폴더여야 합니다")
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

func (s *Service) viewForEntryLocked(entry Entry) View {
	state := s.stateForEntryLocked(entry.ID)
	if state == nil {
		state = &programState{status: StatusStopped}
	}
	status := state.status
	pid := state.pid
	canReconnect := state.canReconnect
	lastError := state.lastError
	startedAt := state.startedAt
	memoryWorkingSetBytes := int64(0)
	memoryPrivateBytes := int64(0)

	if status == StatusRunning || status == StatusStarting || status == StatusStopping || status == StatusStopped || status == StatusOrphaned {
		if pid <= 0 {
			if foundPID, ok := processPathLookup(entry.Path); ok {
				pid = foundPID
				if status == StatusStopped || status == StatusOrphaned {
					status = StatusRunning
					canReconnect = true
					lastError = ""
				}
			}
		}

		if pid > 0 {
			if snapshot, ok := processSnapshotByPID(pid); ok {
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

func (s *Service) capturePipe(id string, stream string, reader io.ReadCloser) {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		s.mu.Lock()
		state := s.ensureStateLocked(id)
		if state.logs == nil {
			state.logs = newLogBuffer(500)
		}

		now := time.Now().UTC()
		state.logs.Append(stream, scanner.Text(), now)
		state.lastLogAt = now
		s.mu.Unlock()
	}

	if err := scanner.Err(); err != nil {
		s.mu.Lock()
		state := s.ensureStateLocked(id)
		state.lastError = err.Error()
		s.mu.Unlock()
	}
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
