package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/v4run/kapish/internal/enrich"
)

func readyManager() *enrich.Manager {
	specs := []enrich.ProviderSpec{{
		Provider: enrich.NewStaticProvider("cost", map[string]enrich.Row{
			"prod-eu-1": {"monthlyCost": "$412"},
		}),
		Interval: time.Hour, Timeout: time.Second,
	}}
	cols := []enrich.Column{{Header: "COST", Provider: "cost", Field: "monthlyCost", JoinOn: "name"}}
	m := enrich.NewManager(specs, cols)
	m.RefreshAll(context.Background())
	return m
}

func TestUpdate_EnrichUpdatedMsgReRenders(t *testing.T) {
	m := New(Config{MgmtContext: "mgmt-eu", Enrichment: readyManager()})
	// enrichUpdatedMsg must be handled and return a command to keep listening.
	mu, cmd := m.Update(enrichUpdatedMsg{})
	_ = mu.(Model)
	assert.NotNil(t, cmd, "should re-subscribe after an enrich update")
}

func TestEnrichUpdatesCmd_NilWithoutManager(t *testing.T) {
	m := New(Config{})
	assert.Nil(t, m.enrichUpdatesCmd())
}

var _ tea.Msg = enrichUpdatedMsg{}
