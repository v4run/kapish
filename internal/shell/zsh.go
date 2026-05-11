package shell

import "strings"

func zshInit(opts Options, kubeconfigPath string) string {
	var b strings.Builder

	b.WriteString(`[ -f "$HOME/.zshrc" ] && . "$HOME/.zshrc"` + "\n")
	b.WriteString("export KUBECONFIG=" + posixSingleQuote(kubeconfigPath) + "\n")
	for _, k := range sortedKeys(opts.Env) {
		b.WriteString("export " + k + "=" + posixSingleQuote(opts.Env[k]) + "\n")
	}
	for _, k := range sortedKeys(opts.Aliases) {
		b.WriteString("alias " + k + "=" + posixSingleQuote(opts.Aliases[k]) + "\n")
	}
	if opts.PromptTemplate != "" {
		prefix := RenderPrompt(opts.PromptTemplate, opts.PromptTokens)
		b.WriteString("PROMPT=" + posixSingleQuote(prefix) + `"$PROMPT"` + "\n")
	}
	if opts.Cwd != "" {
		b.WriteString("cd " + posixSingleQuote(opts.Cwd) + "\n")
	}
	return b.String()
}
