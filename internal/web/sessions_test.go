package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostSessions_RequiresCapiClient(t *testing.T) {
	s := newTestServer(t) // CapiClient nil
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewReader([]byte(`{"namespace":"prod","cluster":"prod-eu-1"}`)))
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	// With no capi client we can't fetch the kubeconfig — expect 5xx, not a panic.
	assert.GreaterOrEqual(t, rec.Code, 500)
}

func TestPostSessions_BadBody(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSessionStore_CreateLookupTokenSingleUse(t *testing.T) {
	st := newSessionStore()
	id, tok := st.create(&ptySession{}) // ptySession fields don't matter here
	require.NotEmpty(t, id)
	require.NotEmpty(t, tok)

	// Lookup with the right token works once.
	sess, ok := st.consumeToken(id, tok)
	require.True(t, ok)
	require.NotNil(t, sess)

	// Second use of the same token fails.
	_, ok = st.consumeToken(id, tok)
	assert.False(t, ok)

	// Wrong token fails.
	id2, _ := st.create(&ptySession{})
	_, ok = st.consumeToken(id2, "wrong")
	assert.False(t, ok)
}

func TestSessionStore_JanitorEvictsStaleUnusedSessions(t *testing.T) {
	st := newSessionStore()
	// Create a session and backdate its created time.
	sess := &ptySession{created: time.Now().Add(-2 * time.Minute)}
	id, _ := st.create(sess)
	// Also a fresh one that should survive.
	freshID, _ := st.create(&ptySession{created: time.Now()})

	st.evictStale(time.Minute)

	st.mu.Lock()
	_, oldExists := st.m[id]
	_, freshExists := st.m[freshID]
	st.mu.Unlock()
	assert.False(t, oldExists, "stale unused session should be evicted")
	assert.True(t, freshExists, "fresh session should survive")
}
