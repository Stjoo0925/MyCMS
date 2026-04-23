package programs

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigPathUsesMyCMSDirectory(t *testing.T) {
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error = %v", err)
	}

	if !strings.Contains(strings.ToLower(path), `mycms`) {
		t.Fatalf("DefaultConfigPath() = %q, want path containing mycms", path)
	}
}

func TestJSONStoreLoadReturnsEmptyWhenFileMissing(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "programs.json"))

	programs, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(programs) != 0 {
		t.Fatalf("Load() length = %d, want 0", len(programs))
	}
}

func TestJSONStoreSaveAndLoad(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "programs.json"))
	want := []Entry{
		{ID: "one", Name: "Alpha", Path: `C:\tools\alpha.bat`},
		{ID: "two", Name: "Beta", Path: `C:\tools\beta.bat`},
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("Load() length = %d, want %d", len(got), len(want))
	}

	for index := range want {
		if got[index].ID != want[index].ID || got[index].Name != want[index].Name || got[index].Path != want[index].Path {
			t.Fatalf("Load()[%d] = %#v, want %#v", index, got[index], want[index])
		}
	}
}

func TestJSONStoreSaveAndLoadVersion2Document(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "programs.json"))
	want := []Entry{
		{
			ID:                  "one",
			Name:                "Alpha",
			Description:         "primary job",
			Notes:               "nightly run",
			Tags:                []string{"ops", "nightly"},
			Path:                `C:\tools\alpha.bat`,
			Kind:                KindBatch,
			WorkingDirectory:    `C:\tools`,
			Args:                []string{"--sync"},
			Env:                 []EnvVar{{Key: "MODE", Value: "prod"}},
			RunAsAdmin:          true,
			RestartPolicy:       RestartPolicyOnFailure,
			RestartLimit:        3,
			RestartDelaySeconds: 5,
		},
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("Load() length = %d, want 1", len(got))
	}

	if got[0].Description != want[0].Description || got[0].Kind != want[0].Kind {
		t.Fatalf("Load() = %#v, want %#v", got[0], want[0])
	}
}

func TestJSONStoreLoadUpgradesLegacyDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "programs.json")
	if err := writeTextFile(path, "{\n  \"programs\": [{\"id\":\"one\",\"name\":\"Legacy\",\"path\":\"C:\\\\tools\\\\legacy.bat\"}]\n}\n"); err != nil {
		t.Fatalf("writeTextFile() error = %v", err)
	}

	store := NewJSONStore(path)
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("Load() length = %d, want 1", len(got))
	}

	if got[0].Kind != KindBatch {
		t.Fatalf("Kind = %q, want %q", got[0].Kind, KindBatch)
	}
	if got[0].RestartPolicy != RestartPolicyNone {
		t.Fatalf("RestartPolicy = %q, want %q", got[0].RestartPolicy, RestartPolicyNone)
	}
}

func TestJSONStoreLoadRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "programs.json")
	store := NewJSONStore(path)

	if err := writeTextFile(path, "{not-json"); err != nil {
		t.Fatalf("writeTextFile() error = %v", err)
	}

	if _, err := store.Load(); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}
