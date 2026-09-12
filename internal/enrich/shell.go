package enrich

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// ShellProvider execs a configured argv and parses its stdout as a JSON array
// of flat objects. It never runs through a shell — argv is executed directly,
// so there is no shell-injection surface.
type ShellProvider struct {
	name     string
	argv     []string
	keyField string
}

func NewShellProvider(name string, argv []string, keyField string) *ShellProvider {
	return &ShellProvider{name: name, argv: argv, keyField: keyField}
}

func (p *ShellProvider) Name() string { return p.name }

func (p *ShellProvider) Fetch(ctx context.Context) (map[string]Row, error) {
	if len(p.argv) == 0 {
		return nil, fmt.Errorf("enrich: shell provider %q: empty command", p.name)
	}
	cmd := exec.CommandContext(ctx, p.argv[0], p.argv[1:]...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("enrich: shell provider %q: %w", p.name, err)
	}

	var records []map[string]any
	if err := json.Unmarshal(out, &records); err != nil {
		return nil, fmt.Errorf("enrich: shell provider %q: parse json: %w", p.name, err)
	}

	rows := make(map[string]Row, len(records))
	for _, rec := range records {
		kv, ok := rec[p.keyField]
		if !ok {
			continue // missing key: skip
		}
		key := stringify(kv)
		if key == "" {
			continue
		}
		if _, dup := rows[key]; dup {
			continue // duplicate key: first wins
		}
		row := make(Row, len(rec))
		for k, v := range rec {
			row[k] = stringify(v)
		}
		rows[key] = row
	}
	return rows, nil
}

// stringify renders a decoded JSON value as display text. JSON numbers decode
// as float64; integral values render without a trailing ".0".
func stringify(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}
