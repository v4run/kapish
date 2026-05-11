package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// sweepStaleTempDirs removes directories under base named "kapish-*" whose
// mtime is older than maxAge. Returns the count removed. Errors removing an
// individual dir are ignored (best-effort); a fatal error reading base is
// returned.
func sweepStaleTempDirs(base string, maxAge time.Duration) (int, error) {
	entries, err := os.ReadDir(base)
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "kapish-") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.RemoveAll(filepath.Join(base, e.Name())); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}
