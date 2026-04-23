package programs

import (
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestDetectKindFromSupportedExtensions(t *testing.T) {
	cases := map[string]string{
		`C:\tools\run.bat`: KindBatch,
		`C:\tools\run.cmd`: KindCommand,
		`C:\tools\run.exe`: KindExecutable,
		`C:\tools\run.ps1`: KindPowerShell,
		`C:\tools\run.py`:  KindPython,
		`C:\tools\run.js`:  KindNode,
		`C:\tools\run.mjs`: KindNode,
		`C:\tools\run.cjs`: KindNode,
		`C:\tools\run.jar`: KindJar,
	}

	for path, want := range cases {
		got, err := detectKind(path)
		if err != nil {
			t.Fatalf("detectKind(%q) error = %v", path, err)
		}
		if got != want {
			t.Fatalf("detectKind(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestBuildCommandUsesExpectedLauncher(t *testing.T) {
	entry := Entry{
		Path:             `C:\tools\job.ps1`,
		Kind:             KindPowerShell,
		Args:             []string{"-Mode", "Prod"},
		WorkingDirectory: `C:\tools`,
	}

	cmd, elevated, err := buildCommand(entry)
	if err != nil {
		t.Fatalf("buildCommand() error = %v", err)
	}

	if elevated {
		t.Fatal("buildCommand() elevated = true, want false")
	}

	got := strings.Join(cmd.Args, " ")
	want := "powershell.exe -NoProfile -ExecutionPolicy Bypass -File C:\\tools\\job.ps1 -Mode Prod"
	if got != want {
		t.Fatalf("cmd.Args = %q, want %q", got, want)
	}
}

func TestBuildCommandUsesCmdCallForBatchPrograms(t *testing.T) {
	entry := Entry{
		Path:             `C:\Program Files\tools\run.bat`,
		Kind:             KindBatch,
		Args:             []string{"--mode", "prod"},
		WorkingDirectory: `C:\Program Files\tools`,
	}

	cmd, elevated, err := buildCommand(entry)
	if err != nil {
		t.Fatalf("buildCommand() error = %v", err)
	}

	if elevated {
		t.Fatal("buildCommand() elevated = true, want false")
	}

	if len(cmd.Args) != 4 {
		t.Fatalf("len(cmd.Args) = %d, want 4 (%#v)", len(cmd.Args), cmd.Args)
	}
	if cmd.Args[0] != "cmd.exe" || cmd.Args[1] != "/D" || cmd.Args[2] != "/C" {
		t.Fatalf("cmd prefix = %#v, want cmd.exe /D /C", cmd.Args[:3])
	}
	want := `"C:\Program Files\tools\run.bat" "--mode" "prod"`
	if cmd.Args[3] != want {
		t.Fatalf("cmd.Args[3] = %q, want %q", cmd.Args[3], want)
	}
}

func TestBuildCommandOpensNewConsoleForBatchPrograms(t *testing.T) {
	entry := Entry{
		Path:             `C:\tools\run.bat`,
		Kind:             KindBatch,
		WorkingDirectory: `C:\tools`,
	}

	cmd, elevated, err := buildCommand(entry)
	if err != nil {
		t.Fatalf("buildCommand() error = %v", err)
	}

	if elevated {
		t.Fatal("buildCommand() elevated = true, want false")
	}

	if cmd.SysProcAttr == nil {
		t.Fatal("buildCommand() SysProcAttr = nil, want new console flags")
	}

	if cmd.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatalf("CreationFlags = %#x, want CREATE_NO_WINDOW", cmd.SysProcAttr.CreationFlags)
	}
}
