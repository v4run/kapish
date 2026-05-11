// Package tui implements kapish's terminal UI: a cluster list with live
// updates, filter, management-cluster picker, settings screen, and the
// spawn-a-shell flow.
package tui

import (
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
}

// Model is the root bubbletea model.
type Model struct {
	cfg Config

	screen screen
	err    error

	clusters []capi.Cluster // sorted
	filter   textinput.Model
	filtered []capi.Cluster // filter applied
	cursor   int            // index into filtered
	// stickyKey lets the cursor follow the same cluster across re-sorts.
	stickyKey string

	mgmtContext string

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

func (m Model) Init() tea.Cmd {
	// Real entry wires the LIST+WATCH cmds here; tests drive Update directly.
	return nil
}

// View renders the current screen. Full rendering is implemented in Task 6.
func (m Model) View() string {
	return ""
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
	}
	return m, nil
}

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
