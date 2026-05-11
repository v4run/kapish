package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFishInit_HasStandardLines(t *testing.T) {
	opts := Options{
		Env:            map[string]string{"FOO": "bar"},
		Aliases:        map[string]string{"k": "kubectl"},
		PromptTemplate: "[{cluster}] ",
		Cwd:            "/tmp",
		PromptTokens:   PromptTokens{Cluster: "x"},
	}
	got := fishInit(opts, "/tmp/kapish-abc/kubeconfig")

	assert.Contains(t, got, "set -gx KUBECONFIG '/tmp/kapish-abc/kubeconfig'")
	assert.Contains(t, got, "set -gx FOO 'bar'")
	assert.Contains(t, got, "alias k 'kubectl'")
	assert.Contains(t, got, "function fish_prompt")
	assert.Contains(t, got, "echo -n '[x] '")
	assert.Contains(t, got, "cd '/tmp'")
}
