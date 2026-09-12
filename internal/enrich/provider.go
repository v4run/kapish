package enrich

import "context"

// StaticProvider returns a fixed set of rows. Useful for tests and as the shape
// the (future) labels provider will follow.
type StaticProvider struct {
	name string
	rows map[string]Row
}

func NewStaticProvider(name string, rows map[string]Row) *StaticProvider {
	return &StaticProvider{name: name, rows: rows}
}

func (p *StaticProvider) Name() string { return p.name }

func (p *StaticProvider) Fetch(context.Context) (map[string]Row, error) {
	return p.rows, nil
}
