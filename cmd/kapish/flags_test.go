package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGlobalFlags_DefaultsAreSane(t *testing.T) {
	cmd := newRootCmd()
	require.NoError(t, cmd.ParseFlags([]string{}))

	g, err := readGlobalFlags(cmd)
	require.NoError(t, err)
	assert.Equal(t, "", g.ConfigPath)
	assert.Equal(t, "", g.Kubeconfig)
	assert.Equal(t, "", g.Context)
	assert.Equal(t, "info", g.LogLevel)
	assert.Equal(t, "", g.LogFile)
	assert.False(t, g.OneShot)
}

func TestParseGlobalFlags_AllValuesSet(t *testing.T) {
	cmd := newRootCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--config", "/tmp/c.yaml",
		"--kubeconfig", "/tmp/k",
		"--context", "ctx",
		"--log-level", "debug",
		"--log-file", "/tmp/k.log",
		"--one-shot",
	}))

	g, err := readGlobalFlags(cmd)
	require.NoError(t, err)
	assert.Equal(t, "/tmp/c.yaml", g.ConfigPath)
	assert.Equal(t, "/tmp/k", g.Kubeconfig)
	assert.Equal(t, "ctx", g.Context)
	assert.Equal(t, "debug", g.LogLevel)
	assert.Equal(t, "/tmp/k.log", g.LogFile)
	assert.True(t, g.OneShot)
}
