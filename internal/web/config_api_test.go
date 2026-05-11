package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kconfig "github.com/v4run/kapish/internal/config"
)

func TestGetConfig_ReturnsEffectiveConfig(t *testing.T) {
	cfg := kconfig.Defaults()
	cfg.UI.Theme = "light"
	s, err := New(Options{AppConfig: cfg, MgmtContext: "m"})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/config", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	ui := got["ui"].(map[string]any)
	assert.Equal(t, "light", ui["theme"])
}

func TestPutConfig_ValidatesAndPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("ui:\n  theme: dark\n"), 0o600))

	s, err := New(Options{AppConfig: kconfig.Defaults(), MgmtContext: "m", ConfigPath: path})
	require.NoError(t, err)

	body := []byte(`{"ui":{"theme":"light","refreshIntervalSec":30,"oneShot":false},"shell":{"prompt":"[{cluster}] "},"web":{"defaultPort":0,"openBrowser":true,"bindAddr":"127.0.0.1"},"managementClusters":{}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// File updated.
	c, err := kconfig.LoadFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, "light", c.UI.Theme)
}

func TestPutConfig_RejectsInvalid(t *testing.T) {
	s, err := New(Options{AppConfig: kconfig.Defaults(), MgmtContext: "m"})
	require.NoError(t, err)
	// Unknown prompt token -> validation failure.
	body := []byte(`{"ui":{"theme":"dark","refreshIntervalSec":30,"oneShot":false},"shell":{"prompt":"{nope}"},"web":{"defaultPort":0,"openBrowser":true,"bindAddr":"127.0.0.1"},"managementClusters":{}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "nope")
}
