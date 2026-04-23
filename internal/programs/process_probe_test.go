package programs

import "testing"

func TestMatchProcessCandidateUsesCommandLineForWrapperProcesses(t *testing.T) {
	cases := []struct {
		name      string
		entry     Entry
		candidate processCandidate
		want      bool
	}{
		{
			name: "batch matches cmd command line",
			entry: Entry{
				Path: `C:\Users\J\Desktop\MyCMS\ghost-app.bat`,
				Kind: KindBatch,
			},
			candidate: processCandidate{
				imagePath:   `C:\Windows\System32\cmd.exe`,
				commandLine: `cmd.exe /D /C "C:\Users\J\Desktop\MyCMS\ghost-app.bat"`,
			},
			want: true,
		},
		{
			name: "powershell matches script command line",
			entry: Entry{
				Path: `C:\tools\job.ps1`,
				Kind: KindPowerShell,
			},
			candidate: processCandidate{
				imagePath:   `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
				commandLine: `powershell.exe -File C:\tools\job.ps1`,
			},
			want: true,
		},
		{
			name: "node matches wrapper command line",
			entry: Entry{
				Path: `C:\apps\server.js`,
				Kind: KindNode,
			},
			candidate: processCandidate{
				imagePath:   `C:\Program Files\nodejs\node.exe`,
				commandLine: `node C:\apps\server.js`,
			},
			want: true,
		},
		{
			name: "exe requires image path match",
			entry: Entry{
				Path: `C:\apps\worker.exe`,
				Kind: KindExecutable,
			},
			candidate: processCandidate{
				imagePath:   `C:\Windows\System32\cmd.exe`,
				commandLine: `cmd.exe /c C:\apps\worker.exe`,
			},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchProcessCandidate(tc.entry, tc.candidate); got != tc.want {
				t.Fatalf("matchProcessCandidate() = %v, want %v", got, tc.want)
			}
		})
	}
}
