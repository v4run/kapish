package shell

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRenderPrompt_AllTokens(t *testing.T) {
	tok := PromptTokens{
		Cluster:   "prod-eu-1",
		Namespace: "prod",
		Provider:  "aws",
		Ctx:       "mgmt-eu",
		Now:       time.Date(2026, 5, 10, 14, 30, 0, 0, time.UTC),
	}
	got := RenderPrompt("[{cluster}/{ns}] {provider}@{ctx} {time} ", tok)
	assert.Equal(t, "[prod-eu-1/prod] aws@mgmt-eu 14:30 ", got)
}

func TestRenderPrompt_EmptyTemplate(t *testing.T) {
	got := RenderPrompt("", PromptTokens{Cluster: "x"})
	assert.Equal(t, "", got)
}

func TestRenderPrompt_UnknownTokenLeftLiteral(t *testing.T) {
	got := RenderPrompt("[{nope}] ", PromptTokens{Cluster: "x"})
	assert.True(t, strings.Contains(got, "{nope}"), "got: %q", got)
}

func TestRenderPrompt_TimeIsHHMM(t *testing.T) {
	got := RenderPrompt("{time}", PromptTokens{Now: time.Date(2026, 1, 1, 9, 5, 0, 0, time.UTC)})
	assert.Equal(t, "09:05", got)
}
