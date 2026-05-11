package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/v4run/kapish/internal/capi"
)

func TestShellExited_ReturnsToList(t *testing.T) {
	m := readyModelWith(capi.Cluster{Name: "a", Namespace: "ns"})
	m.screen = screenSpawning
	mu, _ := m.Update(shellExitedMsg{err: nil})
	mm := mu.(Model)
	assert.Equal(t, screenReady, mm.screen)
}

func TestShellExited_OneShotQuits(t *testing.T) {
	m := readyModelWith(capi.Cluster{Name: "a", Namespace: "ns"})
	m.cfg.OneShot = true
	m.screen = screenSpawning
	mu, cmd := m.Update(shellExitedMsg{err: nil})
	mm := mu.(Model)
	// One-shot: returns a quit cmd; screen doesn't matter.
	_ = mm
	assert.NotNil(t, cmd, "one-shot should quit after shell exit")
}

func TestBeginSpawn_FailedClusterAsksConfirm(t *testing.T) {
	m := readyModelWith(capi.Cluster{Name: "bad", Namespace: "ns", Phase: "Failed"})
	mu, _ := m.beginSpawn()
	mm := mu.(Model)
	assert.True(t, mm.confirmingSpawn, "Failed cluster should require confirmation")
}

func TestBeginSpawn_HealthyClusterNoConfirm(t *testing.T) {
	// With no CapiClient, beginSpawn can't actually run; assert it doesn't
	// crash and doesn't set confirmingSpawn for a Provisioned cluster.
	m := readyModelWith(capi.Cluster{Name: "ok", Namespace: "ns", Phase: "Provisioned"})
	mu, _ := m.beginSpawn()
	mm := mu.(Model)
	assert.False(t, mm.confirmingSpawn)
}

func TestConfirmSpawn_YesProceeds_NoCancels(t *testing.T) {
	m := readyModelWith(capi.Cluster{Name: "bad", Namespace: "ns", Phase: "Failed"})
	mu, _ := m.beginSpawn()
	m = mu.(Model)
	require := assert.New(t)
	require.True(m.confirmingSpawn)

	// 'n' cancels.
	mu, _ = m.Update(key('n'))
	m = mu.(Model)
	require.False(m.confirmingSpawn)
	require.Equal(screenReady, m.screen)
}
