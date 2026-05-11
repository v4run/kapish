package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kconfig "github.com/v4run/kapish/internal/config"
)

func serverWithMgmts(t *testing.T, current string, names ...string) *Server {
	t.Helper()
	entries := make([]kconfig.ManagementClusterEntry, 0, len(names))
	for _, n := range names {
		entries = append(entries, kconfig.ManagementClusterEntry{Name: n})
	}
	cfg := kconfig.Defaults()
	cfg.ManagementClusters = kconfig.ManagementClustersConfig{Current: current, Entries: entries}
	s, err := New(Options{AppConfig: cfg, MgmtContext: current})
	require.NoError(t, err)
	return s
}

func TestGetMgmts_ListsEntriesAndCurrent(t *testing.T) {
	s := serverWithMgmts(t, "b", "a", "b", "c")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/mgmts", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var got struct {
		Current string `json:"current"`
		Entries []struct {
			Name string `json:"name"`
		} `json:"entries"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "b", got.Current)
	require.Len(t, got.Entries, 3)
	assert.Equal(t, "a", got.Entries[0].Name)
}

func TestPutMgmtsCurrent_UnknownNameRejected(t *testing.T) {
	s := serverWithMgmts(t, "a", "a", "b")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/mgmts/current", bytes.NewReader([]byte(`{"name":"nope"}`)))
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
