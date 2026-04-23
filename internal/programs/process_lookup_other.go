//go:build !windows

package programs

func probeProcessByPath(path string) (int, bool) {
	return 0, false
}
