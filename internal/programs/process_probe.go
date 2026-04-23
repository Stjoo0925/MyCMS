package programs

import (
	"path/filepath"
	"strings"
)

type processCandidate struct {
	pid         int
	imagePath   string
	commandLine string
}

var processProbe = probeProcess
var processCandidatesProvider = listProcessCandidates

func normalizeComparablePath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	replaced := strings.ReplaceAll(trimmed, "/", `\`)
	return strings.ToLower(filepath.Clean(replaced))
}

func normalizeComparableCommandLine(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	replaced := strings.ReplaceAll(trimmed, "/", `\`)
	return strings.ToLower(trimmed + " " + replaced)
}

func matchProcessCandidate(entry Entry, candidate processCandidate) bool {
	targetPath := normalizeComparablePath(entry.Path)
	if targetPath == "" {
		return false
	}

	imagePath := normalizeComparablePath(candidate.imagePath)
	commandLine := normalizeComparableCommandLine(candidate.commandLine)
	imageBase := strings.ToLower(filepath.Base(candidate.imagePath))

	switch entry.Kind {
	case KindExecutable:
		return imagePath == targetPath
	case KindBatch, KindCommand:
		return imageBase == "cmd.exe" && strings.Contains(commandLine, targetPath)
	case KindPowerShell:
		return (imageBase == "powershell.exe" || imageBase == "pwsh.exe") && strings.Contains(commandLine, targetPath)
	case KindPython, KindNode, KindJar:
		return strings.Contains(commandLine, targetPath)
	default:
		return imagePath == targetPath || strings.Contains(commandLine, targetPath)
	}
}

func candidateBaseName(candidate processCandidate) string {
	return strings.ToLower(filepath.Base(candidate.imagePath))
}
