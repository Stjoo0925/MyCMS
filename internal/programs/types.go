package programs

import "time"

const (
	StatusStarting = "STARTING"
	StatusRunning  = "RUNNING"
	StatusStopping = "STOPPING"
	StatusStopped  = "STOPPED"
	StatusOrphaned = "ORPHANED"
)

const (
	KindBatch      = "bat"
	KindCommand    = "cmd"
	KindExecutable = "exe"
	KindPowerShell = "ps1"
	KindPython     = "py"
	KindNode       = "js"
	KindJar        = "jar"
)

const (
	RestartPolicyNone      = "none"
	RestartPolicyOnFailure = "on-failure"
	RestartPolicyAlways    = "always"
)

type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ListQuery struct {
	Search        string `json:"search"`
	Status        string `json:"status"`
	Tag           string `json:"tag"`
	SortBy        string `json:"sortBy"`
	SortDirection string `json:"sortDirection"`
}

type Input struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Notes               string   `json:"notes"`
	Tags                []string `json:"tags"`
	Path                string   `json:"path"`
	WorkingDirectory    string   `json:"workingDirectory"`
	Args                []string `json:"args"`
	Env                 []EnvVar `json:"env"`
	RunAsAdmin          bool     `json:"runAsAdmin"`
	RestartPolicy       string   `json:"restartPolicy"`
	RestartLimit        int      `json:"restartLimit"`
	RestartDelaySeconds int      `json:"restartDelaySeconds"`
}

type Entry struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	Notes               string    `json:"notes"`
	Tags                []string  `json:"tags"`
	Path                string    `json:"path"`
	Kind                string    `json:"kind"`
	WorkingDirectory    string    `json:"workingDirectory"`
	Args                []string  `json:"args"`
	Env                 []EnvVar  `json:"env"`
	RunAsAdmin          bool      `json:"runAsAdmin"`
	RestartPolicy       string    `json:"restartPolicy"`
	RestartLimit        int       `json:"restartLimit"`
	RestartDelaySeconds int       `json:"restartDelaySeconds"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type View struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Notes                 string   `json:"notes"`
	Tags                  []string `json:"tags"`
	Path                  string   `json:"path"`
	Kind                  string   `json:"kind"`
	LaunchMode            string   `json:"launchMode"`
	WorkingDirectory      string   `json:"workingDirectory"`
	Args                  []string `json:"args"`
	Env                   []EnvVar `json:"env"`
	RunAsAdmin            bool     `json:"runAsAdmin"`
	RestartPolicy         string   `json:"restartPolicy"`
	RestartLimit          int      `json:"restartLimit"`
	RestartDelaySeconds   int      `json:"restartDelaySeconds"`
	Status                string   `json:"status"`
	LastError             string   `json:"lastError"`
	PID                   int      `json:"pid"`
	StartedAt             string   `json:"startedAt"`
	LastExitAt            string   `json:"lastExitAt"`
	MemoryWorkingSetBytes int64    `json:"memoryWorkingSetBytes"`
	MemoryPrivateBytes    int64    `json:"memoryPrivateBytes"`
	RestartCount          int      `json:"restartCount"`
	Elevated              bool     `json:"elevated"`
	CanReconnect          bool     `json:"canReconnect"`
	LastLogAt             string   `json:"lastLogAt"`
	CreatedAt             string   `json:"createdAt"`
	UpdatedAt             string   `json:"updatedAt"`
}
