package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSweepStaleTempDirs_RemovesOldOnly(t *testing.T) {
	base := t.TempDir()

	old := filepath.Join(base, "kapish-old")
	require.NoError(t, os.MkdirAll(old, 0o700))
	past := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(old, past, past))

	recent := filepath.Join(base, "kapish-recent")
	require.NoError(t, os.MkdirAll(recent, 0o700))

	unrelated := filepath.Join(base, "not-kapish")
	require.NoError(t, os.MkdirAll(unrelated, 0o700))

	n, err := sweepStaleTempDirs(base, 24*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	_, err = os.Stat(old)
	assert.True(t, os.IsNotExist(err), "old kapish dir should be removed")
	_, err = os.Stat(recent)
	assert.NoError(t, err, "recent kapish dir should remain")
	_, err = os.Stat(unrelated)
	assert.NoError(t, err, "non-kapish dir should remain")
}
