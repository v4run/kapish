// Package shell builds per-session shell init (rcfile / ZDOTDIR / fish init)
// and exposes a SpawnPlan that callers wrap in os/exec or PTY as appropriate.
package shell

import "time"

// Options describe everything needed to spawn a kapish shell session.
type Options struct {
	PathToShell    string
	Cwd            string
	Env            map[string]string
	Aliases        map[string]string
	PromptTemplate string
	PromptTokens   PromptTokens
}

// PromptTokens are substitution values for the prompt template.
type PromptTokens struct {
	Cluster   string
	Namespace string
	Provider  string
	Ctx       string
	Now       time.Time // used to render {time} as HH:MM
}
