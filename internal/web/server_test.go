package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kconfig "github.com/v4run/kapish/internal/config"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	s, err := New(Options{
		AppConfig:   kconfig.Defaults(),
		MgmtContext: "test-mgmt",
		// CapiClient nil — endpoints that need it must handle that gracefully or
		// the test only hits ones that don't.
	})
	require.NoError(t, err)
	return s
}

func TestHealth_OK(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	s.Handler().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ok")
}

func TestSecurityHeaders(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	s.Handler().ServeHTTP(rec, req)
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
}

func TestCORS_RejectsCrossOrigin(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	s.Handler().ServeHTTP(rec, req)
	// Either reject with 403 or simply omit the ACAO header — assert the header
	// is NOT echoing the evil origin.
	assert.NotEqual(t, "https://evil.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestUnknownAPIRoute_404(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil)
	s.Handler().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
