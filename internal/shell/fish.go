package shell

import "strings"

func fishInit(opts Options, kubeconfigPath string) string {
	var b strings.Builder

	b.WriteString("set -gx KUBECONFIG " + posixSingleQuote(kubeconfigPath) + "\n")
	for _, k := range sortedKeys(opts.Env) {
		b.WriteString("set -gx " + k + " " + posixSingleQuote(opts.Env[k]) + "\n")
	}
	for _, k := range sortedKeys(opts.Aliases) {
		b.WriteString("alias " + k + " " + posixSingleQuote(opts.Aliases[k]) + "\n")
	}
	if opts.PromptTemplate != "" {
		prefix := RenderPrompt(opts.PromptTemplate, opts.PromptTokens)
		b.WriteString("functions -c fish_prompt _kapish_orig_prompt 2>/dev/null\n")
		b.WriteString("function fish_prompt\n")
		b.WriteString("    echo -n " + posixSingleQuote(prefix) + "\n")
		b.WriteString("    if functions -q _kapish_orig_prompt; _kapish_orig_prompt; end\n")
		b.WriteString("end\n")
	}
	b.WriteString(cdLine(opts.Cwd))
	return b.String()
}
