package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	kconfig "github.com/v4run/kapish/internal/config"
)

func TestSettings_OpensShowsConfig(t *testing.T) {
	cfg := Config{AppConfig: kconfig.Defaults()}
	m := New(cfg)
	mu, _ := m.Update(clustersLoadedMsg{})
	m = mu.(Model)
	mu, _ = m.Update(key('s'))
	m = mu.(Model)
	assert.Equal(t, screenSettings, m.screen)
	out := m.View()
	assert.Contains(t, out, "theme")
	assert.Contains(t, out, "dark") // default
}

func TestSettings_EscReturns(t *testing.T) {
	m := New(Config{AppConfig: kconfig.Defaults()})
	mu, _ := m.Update(clustersLoadedMsg{})
	m = mu.(Model)
	mu, _ = m.Update(key('s'))
	m = mu.(Model)
	mu, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mu.(Model)
	assert.Equal(t, screenReady, m.screen)
}

func TestSettings_ToggleThemeFlipsValue(t *testing.T) {
	m := New(Config{AppConfig: kconfig.Defaults(), ConfigPath: ""})
	mu, _ := m.Update(clustersLoadedMsg{})
	m = mu.(Model)
	mu, _ = m.Update(key('s'))
	m = mu.(Model)
	mu, _ = m.Update(key('t')) // 't' toggles theme
	m = mu.(Model)
	assert.Equal(t, "light", m.cfg.AppConfig.UI.Theme)
	mu, _ = m.Update(key('t'))
	m = mu.(Model)
	assert.Equal(t, "dark", m.cfg.AppConfig.UI.Theme)
}
