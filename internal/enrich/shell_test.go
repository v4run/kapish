package enrich

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "data.json")
	require.NoError(t, os.WriteFile(p, []byte(body), 0o600))
	return p
}

func TestShell_ParsesJSONArray(t *testing.T) {
	path := writeTemp(t, `[{"id":"c1","cost":"$412","nodes":12,"ok":true},{"id":"c2","cost":"$99"}]`)
	p := NewShellProvider("cost", []string{"cat", path}, "id")
	rows, err := p.Fetch(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "$412", rows["c1"]["cost"])
	assert.Equal(t, "12", rows["c1"]["nodes"], "JSON numbers stringify without .0")
	assert.Equal(t, "true", rows["c1"]["ok"])
	assert.Equal(t, "$99", rows["c2"]["cost"])
}

func TestShell_SkipsRowsMissingKey(t *testing.T) {
	path := writeTemp(t, `[{"id":"c1","v":"a"},{"v":"b"},{"id":"c1","v":"dup"}]`)
	p := NewShellProvider("x", []string{"cat", path}, "id")
	rows, err := p.Fetch(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 1, "missing-key row skipped, duplicate-key row skipped")
	assert.Equal(t, "a", rows["c1"]["v"], "first wins on duplicate key")
}

func TestShell_MalformedJSONIsError(t *testing.T) {
	p := NewShellProvider("x", []string{"printf", "not json"}, "id")
	_, err := p.Fetch(context.Background())
	assert.Error(t, err)
}

func TestShell_NonZeroExitIsError(t *testing.T) {
	p := NewShellProvider("x", []string{"false"}, "id")
	_, err := p.Fetch(context.Background())
	assert.Error(t, err)
}

func TestShell_TimeoutIsError(t *testing.T) {
	p := NewShellProvider("x", []string{"sleep", "5"}, "id")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := p.Fetch(ctx)
	assert.Error(t, err)
}

func TestShell_EmptyCommandIsError(t *testing.T) {
	p := NewShellProvider("x", nil, "id")
	_, err := p.Fetch(context.Background())
	assert.Error(t, err)
}
