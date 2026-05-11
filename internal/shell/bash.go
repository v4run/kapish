package shell

import (
	"sort"
	"strings"
)

func bashInit(opts Options, kubeconfigPath string) string {
	var b strings.Builder

	b.WriteString(`[ -f "$HOME/.bashrc" ] && . "$HOME/.bashrc"` + "\n")
	b.WriteString(`export KUBECONFIG=` + posixSingleQuote(kubeconfigPath) + "\n")

	for _, k := range sortedKeys(opts.Env) {
		b.WriteString("export " + k + "=" + posixSingleQuote(opts.Env[k]) + "\n")
	}
	for _, k := range sortedKeys(opts.Aliases) {
		b.WriteString("alias " + k + "=" + posixSingleQuote(opts.Aliases[k]) + "\n")
	}

	if opts.PromptTemplate != "" {
		prefix := RenderPrompt(opts.PromptTemplate, opts.PromptTokens)
		b.WriteString("PS1=" + posixSingleQuote(prefix) + `"$PS1"` + "\n")
	}

	if opts.Cwd != "" {
		b.WriteString("cd " + posixSingleQuote(opts.Cwd) + "\n")
	}

	return b.String()
}

// posixSingleQuote wraps s in single quotes, ANSI-C-escaping embedded single
// quotes via the standard '\'' trick. Used for bash/zsh/fish rcfile gen.
func posixSingleQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
