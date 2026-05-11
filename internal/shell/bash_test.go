package shell

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBashInit_HasStandardLines(t *testing.T) {
	opts := Options{
		Env:            map[string]string{"FOO": "bar"},
		Aliases:        map[string]string{"k": "kubectl"},
		PromptTemplate: "[{cluster}] ",
		Cwd:            "/tmp",
		PromptTokens:   PromptTokens{Cluster: "x"},
	}
	got := bashInit(opts, "/tmp/kapish-abc/kubeconfig")

	assert.Contains(t, got, `[ -f "$HOME/.bashrc" ] && . "$HOME/.bashrc"`)
	assert.Contains(t, got, `export KUBECONFIG='/tmp/kapish-abc/kubeconfig'`)
	assert.Contains(t, got, `export FOO='bar'`)
	assert.Contains(t, got, `alias k='kubectl'`)
	assert.Contains(t, got, `PS1='[x] '"$PS1"`)
	assert.Contains(t, got, `cd '/tmp'`)
}

func TestBashInit_EscapesSingleQuotes(t *testing.T) {
	opts := Options{
		Env:            map[string]string{"X": "it's tricky"},
		Aliases:        map[string]string{"a": "echo 'hi'"},
		PromptTemplate: "",
	}
	got := bashInit(opts, "/k")
	assert.True(t, strings.Contains(got, `export X='it'\''s tricky'`), "got: %s", got)
	assert.True(t, strings.Contains(got, `alias a='echo '\''hi'\'''`), "got: %s", got)
}
