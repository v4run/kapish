package enrich

import (
	"context"
	"sync"
)

type providerState struct {
	rows        map[string]Row
	everFetched bool
	lastErr     error
}

// Manager owns the enrichment providers and a cache of their last results, and
// computes render-ready Cells by joining that cache onto clusters.
type Manager struct {
	specs   map[string]ProviderSpec
	columns []Column

	mu    sync.RWMutex
	state map[string]*providerState

	updates chan struct{}
}

func NewManager(specs []ProviderSpec, columns []Column) *Manager {
	m := &Manager{
		specs:   make(map[string]ProviderSpec, len(specs)),
		columns: columns,
		state:   make(map[string]*providerState, len(specs)),
		updates: make(chan struct{}, 1),
	}
	for _, s := range specs {
		name := s.Provider.Name()
		m.specs[name] = s
		m.state[name] = &providerState{}
	}
	return m
}

// Columns returns the configured columns in display order.
func (m *Manager) Columns() []Column { return m.columns }

// Updates fires (coalesced, non-blocking) whenever a provider's cache changes.
func (m *Manager) Updates() <-chan struct{} { return m.updates }

func (m *Manager) signal() {
	select {
	case m.updates <- struct{}{}:
	default:
	}
}

// Cells returns one Cell per configured column, in column order.
func (m *Manager) Cells(ref ClusterRef) []Cell {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cells := make([]Cell, len(m.columns))
	for i, col := range m.columns {
		st := m.state[col.Provider]
		if st == nil {
			cells[i] = Cell{State: CellError}
			continue
		}
		if !st.everFetched {
			if st.lastErr != nil {
				cells[i] = Cell{State: CellError}
			} else {
				cells[i] = Cell{State: CellPending}
			}
			continue
		}
		key := ref.Name
		if col.JoinOn == "namespace" {
			key = ref.Namespace
		}
		row, ok := st.rows[key]
		if !ok {
			cells[i] = Cell{State: CellEmpty}
			continue
		}
		cells[i] = Cell{Text: row[col.Field], State: CellReady}
	}
	return cells
}

// ProviderErrors returns the last fetch error per provider name (absent if the
// last fetch succeeded).
func (m *Manager) ProviderErrors() map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]error)
	for name, st := range m.state {
		if st.lastErr != nil {
			out[name] = st.lastErr
		}
	}
	return out
}

// refreshOnce fetches a single provider and updates its cache. On error the
// last good rows are kept so stale-but-present data keeps rendering.
func (m *Manager) refreshOnce(ctx context.Context, name string) {
	spec, ok := m.specs[name]
	if !ok {
		return
	}
	fctx := ctx
	if spec.Timeout > 0 {
		var cancel context.CancelFunc
		fctx, cancel = context.WithTimeout(ctx, spec.Timeout)
		defer cancel()
	}
	rows, err := spec.Provider.Fetch(fctx)

	m.mu.Lock()
	st := m.state[name]
	if err != nil {
		st.lastErr = err
	} else {
		st.rows = rows
		st.everFetched = true
		st.lastErr = nil
	}
	m.mu.Unlock()

	m.signal()
}
