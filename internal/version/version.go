package version

import (
	"fmt"
	"runtime/debug"
)

// Set via -ldflags in Makefile builds; ignored under `go install`.
var (
	Version = "dev"
	Commit  = "unknown"
)

// String returns a short version string suitable for `kapish version`.
// Order of preference:
//  1. info.Main.Version (when installed via `go install <module>@<tag>`)
//  2. VCS revision (+"+dirty" if dirty) from BuildInfo
//  3. ldflags-injected Version (Makefile builds)
//  4. "dev"
func String() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
		var rev, dirty string
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
				if len(rev) > 7 {
					rev = rev[:7]
				}
			case "vcs.modified":
				if s.Value == "true" {
					dirty = "+dirty"
				}
			}
		}
		if rev != "" {
			return rev + dirty
		}
	}
	if Version != "" && Version != "dev" {
		return Version
	}
	return "dev"
}

// Long is the human-readable version line for `kapish version`.
func Long() string {
	return fmt.Sprintf("kapish %s (commit %s)", String(), Commit)
}
