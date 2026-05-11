package shell

import "strings"

// RenderPrompt substitutes {cluster}, {ns}, {provider}, {ctx}, {time} in
// tmpl with values from tok. {time} renders as HH:MM (using tok.Now in its
// own location). Unknown tokens are left literal (config-load validation
// rejects unknown tokens at the config layer).
func RenderPrompt(tmpl string, tok PromptTokens) string {
	if tmpl == "" {
		return ""
	}
	r := strings.NewReplacer(
		"{cluster}", tok.Cluster,
		"{ns}", tok.Namespace,
		"{provider}", tok.Provider,
		"{ctx}", tok.Ctx,
		"{time}", tok.Now.Format("15:04"),
	)
	return r.Replace(tmpl)
}
