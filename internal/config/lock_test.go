package config

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Concurrent writes must serialize: the final file must contain valid YAML
// with one of the writers' values, not interleaved bytes from both.
func TestWriteToFile_ConcurrentWritesSerialize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	c := Defaults()
	require.NoError(t, WriteToFile(path, c))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cc := Defaults()
			cc.UI.RefreshIntervalSec = 100 + i
			_ = WriteToFile(path, cc)
		}(i)
	}
	wg.Wait()

	// File must still parse — not corrupted by interleaving.
	got, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, got.UI.RefreshIntervalSec, 100)
	assert.Less(t, got.UI.RefreshIntervalSec, 120)
}
