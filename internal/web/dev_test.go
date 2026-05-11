package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kconfig "github.com/v4run/kapish/internal/config"
)

// TestDevMode_ProxiesRootButNotAPI verifies that in dev mode:
//   - /api/v1/health is still served by the Go handler (not proxied) → 200
//   - / is proxied to the (intentionally-dead) dev target → some non-200 that
//     proves the request was handed to the proxy, not the embedded file server.
func TestDevMode_ProxiesRootButNotAPI(t *testing.T) {
	// Use an intentionally-dead target so we can tell that / was proxied
	// (the proxy returns a 502 Bad Gateway when it can't reach the upstream).
	s, err := New(Options{
		AppConfig:   kconfig.Defaults(),
		MgmtContext: "test-mgmt",
		Dev:         true,
		DevTarget:   "http://127.0.0.1:1", // port 1 is always unreachable
	})
	require.NoError(t, err)

	h := s.Handler()

	// /api/v1/health must be served by Go, not proxied.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, "/api/v1/health should return 200 in dev mode")

	// / must be proxied (not served from the embedded file server).
	// A 5xx from the dead upstream proves the proxy is active.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec2, req2)
	assert.GreaterOrEqual(t, rec2.Code, 500, "/ should be proxied in dev mode (dead target → 5xx)")
}
