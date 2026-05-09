package kapishlog

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_LevelFiltersBelow(t *testing.T) {
	var buf bytes.Buffer
	l, err := New(Options{Level: "info", Writer: &buf})
	require.NoError(t, err)
	l.Debug("nope")
	l.Info("yes")

	out := buf.String()
	assert.NotContains(t, out, "nope")
	assert.Contains(t, out, "yes")
}

func TestNew_DebugLevelLetsDebugThrough(t *testing.T) {
	var buf bytes.Buffer
	l, err := New(Options{Level: "debug", Writer: &buf})
	require.NoError(t, err)
	l.Debug("hello-debug")
	assert.Contains(t, buf.String(), "hello-debug")
}

func TestNew_BadLevel(t *testing.T) {
	_, err := New(Options{Level: "loud"})
	require.Error(t, err)
}

func TestNew_ProducesJSON(t *testing.T) {
	var buf bytes.Buffer
	l, err := New(Options{Level: "info", Writer: &buf})
	require.NoError(t, err)
	l.Info("event", "k", "v")

	// Each output line must be a valid JSON object.
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &m), "line: %s", line)
		assert.Equal(t, "INFO", m["level"])
		assert.Equal(t, "event", m["msg"])
		assert.Equal(t, "v", m["k"])
	}
}
