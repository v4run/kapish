package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZshInit_HasStandardLines(t *testing.T) {
	opts := Options{
		Env:            map[string]string{"FOO": "bar"},
		Aliases:        map[string]string{"k": "kubectl"},
		PromptTemplate: "[{cluster}] ",
		PromptTokens:   PromptTokens{Cluster: "x"},
	}
	got := zshInit(opts, "/tmp/kapish-abc/kubeconfig")

	assert.Contains(t, got, `[ -f "$HOME/.zshrc" ] && . "$HOME/.zshrc"`)
	assert.Contains(t, got, `export KUBECONFIG='/tmp/kapish-abc/kubeconfig'`)
	assert.Contains(t, got, `export FOO='bar'`)
	assert.Contains(t, got, `alias k='kubectl'`)
	assert.Contains(t, got, `PROMPT='[x] '"$PROMPT"`)
}
