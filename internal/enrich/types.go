// Package enrich joins extra columns onto the cluster list from configurable
// providers. It depends on neither the capi package nor any UI package: the
// Manager is fed a lightweight ClusterRef and returns render-ready Cells.
package enrich

import (
	"context"
	"time"
)

// ClusterRef is the minimal cluster identity the manager needs to join on.
type ClusterRef struct {
	Name      string
	Namespace string
}

// Row is one provider output record: field name -> stringified value.
type Row map[string]string

// Provider fetches all rows for one source, once per refresh, keyed by the
// join value.
type Provider interface {
	Name() string
	Fetch(ctx context.Context) (map[string]Row, error)
}

// ProviderSpec pairs a provider with its refresh cadence and per-fetch timeout.
type ProviderSpec struct {
	Provider Provider
	Interval time.Duration
	Timeout  time.Duration
}

// Column is one configured enrichment column.
type Column struct {
	Header   string // display header
	Provider string // provider name this column reads from
	Field    string // which field of the joined Row to display
	JoinOn   string // cluster field matched against the provider row key: name|namespace
	Width    int    // optional min width; 0 = auto-size
}

// CellState classifies an enrichment cell. CellPending is the zero value so an
// unfetched provider naturally reads as pending.
type CellState int

const (
	CellPending CellState = iota
	CellReady
	CellError
	CellEmpty
)

// Cell is a render-ready enrichment value for one cluster/column.
type Cell struct {
	Text  string
	State CellState
}
