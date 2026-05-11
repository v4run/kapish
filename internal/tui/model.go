// Package tui implements kapish's terminal UI: a cluster list with live
// updates, filter, management-cluster picker, settings screen, and the
// spawn-a-shell flow.
package tui

import (
	keybind "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/v4run/kapish/internal/capi"
	kconfig "github.com/v4run/kapish/internal/config"
)

type screen int

const (
	screenLoading screen = iota
	screenReady
	screenSpawning
	screenSettings
	screenMgmtPicker
	screenError
)

// Config is the input to New — everything the TUI needs that it doesn't
// derive itself.
type Config struct {
	// CapiClient may be nil in tests; the real entry point sets it.
	CapiClient *capi.Client
	// AppConfig is the merged kapish config (used for shell spawn + settings).
	AppConfig kconfig.Config
	// MgmtContext is the resolved mgmt cluster context name (for the header).
	MgmtContext string
	// OneShot: quit after the first spawned shell exits.
	OneShot bool
	// ConfigPath is the absolute path to the config file on disk. When
	// non-empty, the settings screen persists theme changes back to disk.
	// Tests pass "" to disable the write.
	ConfigPath string
}

// Model is the root bubbletea model.
type Model struct {
	cfg Config

	screen   screen
	err      error
	fatalErr error // set on spawn-prep failure; non-nil causes non-zero exit in one-shot mode

	clusters []capi.Cluster // sorted
	filter   textinput.Model
	filtered []capi.Cluster // filter applied
	cursor   int            // index into filtered
	// stickyKey lets the cursor follow the same cluster across re-sorts.
	stickyKey string

	mgmtContext string
	mgmtCursor  int // cursor index into ManagementClusters.Entries

	// confirmingSpawn is true when a Failed/Deleting cluster was selected and
	// the user must confirm before spawning.
	confirmingSpawn bool
	spawnTarget     capi.Cluster

	width, height int
}

// New builds the initial model.
func New(cfg Config) Model {
	fi := textinput.New()
	fi.Placeholder = "filter…"
	fi.Prompt = "/ "
	return Model{
		cfg:         cfg,
		screen:      screenLoading,
		filter:      fi,
		mgmtContext: cfg.MgmtContext,
	}
}

// FatalErr returns the error that caused a spawn-prep failure (kubeconfig
// fetch, shell binary missing, etc.). Non-nil only after a spawnFailedMsg in
// one-shot mode; callers use it to exit with a non-zero status.
func (m Model) FatalErr() error { return m.fatalErr }

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadCmd(), m.refreshTickCmd())
}

// recomputeFiltered re-applies the filter and clamps the cursor, keeping it
// on the sticky cluster if still present.
func (m *Model) recomputeFiltered() {
	m.filtered = filterClusters(m.clusters, m.filter.Value())
	if m.stickyKey != "" {
		for i, c := range m.filtered {
			if clusterKey(c) == m.stickyKey {
				m.cursor = i
				return
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor < len(m.filtered) {
		m.stickyKey = clusterKey(m.filtered[m.cursor])
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case clustersLoadedMsg:
		m.clusters = sortClusters(msg.clusters)
		m.recomputeFiltered()
		m.screen = screenReady
		return m, nil

	case clusterEventMsg:
		m.applyEvent(msg.ev)
		m.recomputeFiltered()
		if m.screen == screenLoading {
			m.screen = screenReady
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		m.screen = screenError
		return m, nil

	case shellExitedMsg:
		if m.cfg.OneShot {
			return m, tea.Quit
		}
		m.screen = screenReady
		return m, nil

	case spawnFailedMsg:
		m.err = msg.err
		m.screen = screenError
		if m.cfg.OneShot {
			m.fatalErr = msg.err
			return m, tea.Quit
		}
		return m, nil

	case refreshTickMsg:
		return m, tea.Batch(m.loadCmd(), m.refreshTickCmd())

	case spawnReadyMsg:
		return m, tea.ExecProcess(msg.plan.Cmd, func(err error) tea.Msg {
			_ = msg.plan.Cleanup()
			return shellExitedMsg{err: err}
		})

	case mgmtSwitchedMsg:
		m.cfg.CapiClient = msg.client
		m.mgmtContext = msg.mgmtContext
		m.cfg.AppConfig.ManagementClusters.Current = msg.entryName
		m.screen = screenLoading
		return m, m.loadCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Spawn confirmation: intercept all keys before any other handling.
	if m.confirmingSpawn {
		switch msg.String() {
		case "y", "Y":
			m.confirmingSpawn = false
			m.screen = screenSpawning
			return m, m.spawnCmd(m.spawnTarget)
		default:
			m.confirmingSpawn = false
			return m, nil
		}
	}

	// Filter mode: typing goes to the textinput; Esc/Enter exit it.
	if m.filter.Focused() {
		switch msg.Type {
		case tea.KeyEsc:
			m.filter.SetValue("")
			m.filter.Blur()
			m.recomputeFiltered()
			return m, nil
		case tea.KeyEnter:
			m.filter.Blur()
			return m, nil
		default:
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.recomputeFiltered()
			return m, cmd
		}
	}

	switch m.screen {
	case screenSettings:
		return m.handleSettingsKey(msg)

	case screenMgmtPicker:
		return m.handleMgmtPickerKey(msg)

	case screenError:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.screen = screenLoading
			m.err = nil
			return m, m.loadCmd()
		}
		return m, nil

	case screenReady:
		switch {
		case keybind.Matches(msg, keys.Quit):
			return m, tea.Quit
		case keybind.Matches(msg, keys.Down):
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				m.stickyKeyFromCursor()
			}
			return m, nil
		case keybind.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
				m.stickyKeyFromCursor()
			}
			return m, nil
		case keybind.Matches(msg, keys.Top):
			m.cursor = 0
			m.stickyKeyFromCursor()
			return m, nil
		case keybind.Matches(msg, keys.Bottom):
			if len(m.filtered) > 0 {
				m.cursor = len(m.filtered) - 1
				m.stickyKeyFromCursor()
			}
			return m, nil
		case keybind.Matches(msg, keys.Filter):
			m.filter.Focus()
			return m, nil
		case keybind.Matches(msg, keys.Refresh):
			m.screen = screenLoading
			return m, m.loadCmd()
		case keybind.Matches(msg, keys.Spawn):
			return m.beginSpawn()
		case keybind.Matches(msg, keys.Mgmt):
			return m.openMgmtPicker()
		case keybind.Matches(msg, keys.Settings):
			return m.openSettings()
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) stickyKeyFromCursor() {
	if m.cursor >= 0 && m.cursor < len(m.filtered) {
		m.stickyKey = clusterKey(m.filtered[m.cursor])
	}
}

func (m Model) beginSpawn() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return m, nil
	}
	c := m.filtered[m.cursor]
	if c.Phase == "Failed" || c.Phase == "Deleting" {
		m.confirmingSpawn = true
		m.spawnTarget = c
		return m, nil
	}
	m.screen = screenSpawning
	return m, m.spawnCmd(c)
}

// openMgmtPicker is defined in mgmtpicker.go.
// openSettings is defined in settings.go.

func (m *Model) applyEvent(ev capi.Event) {
	switch ev.Type {
	case capi.EventAdded, capi.EventModified:
		replaced := false
		for i := range m.clusters {
			if clusterKey(m.clusters[i]) == clusterKey(ev.Cluster) {
				m.clusters[i] = ev.Cluster
				replaced = true
				break
			}
		}
		if !replaced {
			m.clusters = append(m.clusters, ev.Cluster)
		}
		m.clusters = sortClusters(m.clusters)
	case capi.EventDeleted:
		out := m.clusters[:0]
		for _, c := range m.clusters {
			if clusterKey(c) != clusterKey(ev.Cluster) {
				out = append(out, c)
			}
		}
		m.clusters = out
	case capi.EventError:
		// Non-fatal: keep last-known clusters. Higher layer reconnects.
	}
}
