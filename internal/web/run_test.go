package web

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kconfig "github.com/v4run/kapish/internal/config"
)

func TestServer_ListenAndShutdown(t *testing.T) {
	s, err := New(Options{AppConfig: kconfig.Defaults(), MgmtContext: "m", BindAddr: "127.0.0.1", Port: 0})
	require.NoError(t, err)

	addr, err := s.Listen()
	require.NoError(t, err)
	assert.Contains(t, addr, "127.0.0.1:")

	errCh := make(chan error, 1)
	go func() { errCh <- s.Serve() }()

	// Hit /health.
	resp, err := http.Get("http://" + addr + "/api/v1/health")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	require.NoError(t, s.Shutdown(ctx))

	select {
	case err := <-errCh:
		// http.ErrServerClosed is the expected return from Serve after Shutdown.
		assert.True(t, err == nil || err == http.ErrServerClosed, "got: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return after Shutdown")
	}
}
