package web

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/v4run/kapish/internal/capi"
)

func TestSSE_StreamsClusterEvents(t *testing.T) {
	s := newTestServer(t)
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/v1/clusters/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", strings.Split(resp.Header.Get("Content-Type"), ";")[0])

	// Push an event after the client connects.
	time.Sleep(50 * time.Millisecond)
	s.cache.applyEvent(capi.Event{Type: capi.EventAdded, Cluster: capi.Cluster{Name: "new", Namespace: "ns", Phase: "Pending"}})

	// Read lines until we see a data: line containing "new".
	br := bufio.NewReader(resp.Body)
	deadline := time.Now().Add(2 * time.Second)
	found := false
	for time.Now().Before(deadline) {
		line, err := br.ReadString('\n')
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "data:") && strings.Contains(line, "new") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected an SSE data line mentioning the new cluster")
}
