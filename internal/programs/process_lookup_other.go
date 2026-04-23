//go:build !windows

package programs

func probeProcessByPath(path string) (int, bool) {
	return 0, false
}

func listProcessCandidates() []processCandidate {
	return nil
}

func probeProcess(entry Entry) (int, bool) {
	return 0, false
}
