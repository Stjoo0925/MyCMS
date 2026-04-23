package programs

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Store interface {
	Load() ([]Entry, error)
	Save(entries []Entry) error
}

type RuntimeStore interface {
	Load() (runtimeDocument, error)
	Save(runtimeDocument) error
}

type JSONStore struct {
	path string
}

type JSONRuntimeStore struct {
	path string
}

type storeDocument struct {
	Version  int     `json:"version"`
	Programs []Entry `json:"programs"`
}

type runtimeDocument struct {
	Programs []runtimeEntry `json:"programs"`
}

type runtimeEntry struct {
	ID           string    `json:"id"`
	PID          int       `json:"pid"`
	Path         string    `json:"path"`
	StartedAt    time.Time `json:"startedAt"`
	CanReconnect bool      `json:"canReconnect"`
}

func NewJSONStore(path string) *JSONStore {
	return &JSONStore{path: path}
}

func NewJSONRuntimeStore(path string) *JSONRuntimeStore {
	return &JSONRuntimeStore{path: path}
}

func DefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, appConfigDirName, "programs.json"), nil
}

func DefaultRuntimePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, appConfigDirName, "runtime.json"), nil
}

func (s *JSONStore) Load() ([]Entry, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Entry{}, nil
		}
		return nil, err
	}

	var document storeDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}

	if document.Programs == nil {
		return []Entry{}, nil
	}

	for index := range document.Programs {
		upgradeEntry(&document.Programs[index])
	}

	return document.Programs, nil
}

func (s *JSONStore) Save(entries []Entry) error {
	document := storeDocument{
		Version:  2,
		Programs: entries,
	}

	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return writeTextFile(s.path, string(data))
}

func (s *JSONRuntimeStore) Load() (runtimeDocument, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return runtimeDocument{Programs: []runtimeEntry{}}, nil
		}
		return runtimeDocument{}, err
	}

	var document runtimeDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return runtimeDocument{}, err
	}
	if document.Programs == nil {
		document.Programs = []runtimeEntry{}
	}
	return document, nil
}

func (s *JSONRuntimeStore) Save(document runtimeDocument) error {
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeTextFile(s.path, string(data))
}

func upgradeEntry(entry *Entry) {
	now := time.Now().UTC()

	if entry.Kind == "" {
		entry.Kind = inferKindFromPath(entry.Path)
	}
	if entry.RestartPolicy == "" {
		entry.RestartPolicy = RestartPolicyNone
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	if entry.UpdatedAt.IsZero() {
		entry.UpdatedAt = entry.CreatedAt
	}
	if entry.Tags == nil {
		entry.Tags = []string{}
	}
	if entry.Args == nil {
		entry.Args = []string{}
	}
	if entry.Env == nil {
		entry.Env = []EnvVar{}
	}
	if entry.WorkingDirectory == "" && entry.Path != "" {
		entry.WorkingDirectory = filepath.Dir(entry.Path)
	}
}

func inferKindFromPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".bat":
		return KindBatch
	case ".cmd":
		return KindCommand
	case ".exe":
		return KindExecutable
	case ".ps1":
		return KindPowerShell
	case ".py":
		return KindPython
	case ".js":
		return KindNode
	case ".mjs":
		return KindNode
	case ".cjs":
		return KindNode
	case ".jar":
		return KindJar
	default:
		return ""
	}
}

func writeTextFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}

	tempName := temp.Name()
	cleanup := func() {
		temp.Close()
		_ = os.Remove(tempName)
	}

	if _, err := temp.WriteString(content); err != nil {
		cleanup()
		return err
	}
	if err := temp.Close(); err != nil {
		cleanup()
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(tempName)
		return err
	}

	if err := os.Rename(tempName, path); err != nil {
		_ = os.Remove(tempName)
		return err
	}

	return nil
}
