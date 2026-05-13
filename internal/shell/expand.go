package shell

import (
	"os"
	"strings"
)

// expandCwd expands a leading ~ or ~/ to the user's home directory, then
// expands $VAR / ${VAR} via the process environment. An empty string is
// returned unchanged, as is a bare ~user form (rare; not worth an os/user
// lookup). If the home directory can't be determined, the ~ is left as-is.
func expandCwd(p string) string {
	if p == "" {
		return ""
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = home + p[1:]
		}
	}
	return os.ExpandEnv(p)
}

// cdLine returns a shell line that cds into the (expanded) working directory,
// or "" when cwd is empty. The same `cd 'dir'` syntax works in zsh, bash and
// fish.
func cdLine(cwd string) string {
	cwd = expandCwd(cwd)
	if cwd == "" {
		return ""
	}
	return "cd " + posixSingleQuote(cwd) + "\n"
}
