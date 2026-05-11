# Web Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `internal/web` — the localhost HTTP+WebSocket server behind `kapish serve`: a cluster-list cache backed by `capi.WatchClusters`, JSON + SSE endpoints for clusters, config GET/PUT, a mgmt-cluster picker endpoint, and a session/WebSocket-PTY bridge that spawns a shell (via `shell.PrepareSpawn`) and streams it to a browser terminal. Plus the `kapish serve` command. The React frontend is Plan 5; this plan ships a minimal placeholder page so `serve` works standalone.

**Architecture:**
- `internal/web.Server` wraps a `*http.Server` bound to `127.0.0.1` by default. A `clusterCache` holds the current `[]capi.Cluster` (seeded by a LIST, kept fresh by a WATCH goroutine + periodic re-LIST) and fans out events to SSE subscribers. A `sessionStore` maps `sessionID → *ptySession`; `POST /api/v1/sessions` creates one (kubeconfig fetched, `shell.PrepareSpawn` prepared but not started) and returns a one-time WebSocket token; the `/ws` handler validates the token, starts the PTY (`creack/pty.Start`), and bridges PTY↔WebSocket with a tiny framing protocol (1-byte prefix: `0x00` data, `0x01` JSON resize, `0x02` ping/pong). All handlers go through middleware: same-origin CORS, `X-Frame-Options: DENY`, request logging.
- `cmd/kapish/serve.go` adds the `serve` subcommand: resolve+validate config, build a `capi.Client`, construct the `Server`, optionally open the browser, run until SIGINT/SIGTERM, then graceful shutdown (close sessions, stop the server).

**Tech Stack:**
- `github.com/coder/websocket` — WebSocket (clean ctx-based API; was nhooyr/websocket)
- `github.com/creack/pty` — PTY allocation
- `net/http` — server
- `github.com/google/uuid` — session IDs + tokens (already an indirect dep via client-go)
- existing: `internal/capi`, `internal/shell`, `internal/config`, `internal/kapishlog`, `cobra`

**End-state:** `kapish serve --port 0` starts a server on a free localhost port, prints the URL, serves a placeholder page at `/`, and exposes `/api/v1/{health,clusters,clusters/stream,config,mgmts,sessions}` plus the `/ws` PTY bridge — all testable headlessly.

---

## Task 1: Add coder/websocket + creack/pty dependencies

**Files:** `go.mod`, `go.sum`

- [ ] **Step 1:** `go get github.com/coder/websocket@latest github.com/creack/pty@latest github.com/google/uuid@latest`
- [ ] **Step 2:** `go mod tidy && go build ./... && go test ./... -count=1` — all green.
- [ ] **Step 3:** Commit:
```bash
git add go.mod go.sum
git commit -m "chore: add coder/websocket, creack/pty, google/uuid dependencies"
```

---

## Task 2: clusterCache — thread-safe cluster snapshot + event fan-out

**Files:**
- Create: `internal/web/clustercache.go`
- Create: `internal/web/clustercache_test.go`

- [ ] **Step 1: Failing test** — `internal/web/clustercache_test.go`:

```go
package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/v4run/kapish/internal/capi"
)

func TestClusterCache_SnapshotReturnsSortedCopy(t *testing.T) {
	c := newClusterCache()
	c.replaceAll([]capi.Cluster{
		{Name: "b", Namespace: "ns"},
		{Name: "a", Namespace: "ns"},
	})
	snap := c.snapshot()
	require.Len(t, snap, 2)
	assert.Equal(t, "a", snap[0].Name)
	assert.Equal(t, "b", snap[1].Name)

	// Mutating the returned slice must not affect the cache.
	snap[0].Name = "MUTATED"
	assert.Equal(t, "a", c.snapshot()[0].Name)
}

func TestClusterCache_ApplyEventAddModifyDelete(t *testing.T) {
	c := newClusterCache()
	c.applyEvent(capi.Event{Type: capi.EventAdded, Cluster: capi.Cluster{Name: "x", Namespace: "ns", Phase: "Pending"}})
	require.Len(t, c.snapshot(), 1)
	assert.Equal(t, "Pending", c.snapshot()[0].Phase)

	c.applyEvent(capi.Event{Type: capi.EventModified, Cluster: capi.Cluster{Name: "x", Namespace: "ns", Phase: "Provisioned"}})
	assert.Equal(t, "Provisioned", c.snapshot()[0].Phase)

	c.applyEvent(capi.Event{Type: capi.EventDeleted, Cluster: capi.Cluster{Name: "x", Namespace: "ns"}})
	assert.Empty(t, c.snapshot())
}

func TestClusterCache_SubscribeReceivesEvents(t *testing.T) {
	c := newClusterCache()
	sub, unsub := c.subscribe()
	defer unsub()

	go c.applyEvent(capi.Event{Type: capi.EventAdded, Cluster: capi.Cluster{Name: "y", Namespace: "ns"}})

	select {
	case ev := <-sub:
		assert.Equal(t, capi.EventAdded, ev.Type)
		assert.Equal(t, "y", ev.Cluster.Name)
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive event")
	}
}

func TestClusterCache_UnsubStopsDelivery(t *testing.T) {
	c := newClusterCache()
	sub, unsub := c.subscribe()
	unsub()
	// applyEvent must not panic on a closed/removed subscriber.
	c.applyEvent(capi.Event{Type: capi.EventAdded, Cluster: capi.Cluster{Name: "z", Namespace: "ns"}})
	// Channel should be closed; a receive returns zero-value + ok==false eventually.
	select {
	case _, ok := <-sub:
		assert.False(t, ok, "unsubbed channel should be closed")
	case <-time.After(time.Second):
		// also acceptable if implementation just stops sending
	}
}
```

- [ ] **Step 2:** `go test ./internal/web -v` → FAIL (package undefined).
- [ ] **Step 3: Implement** — `internal/web/clustercache.go`:

```go
// Package web implements kapish's localhost HTTP+WebSocket server (kapish serve).
package web

import (
	"sort"
	"sync"

	"github.com/v4run/kapish/internal/capi"
)

// clusterCache holds the current set of CAPI clusters and fans events out to
// SSE subscribers. Safe for concurrent use.
type clusterCache struct {
	mu       sync.Mutex
	byKey    map[string]capi.Cluster
	subs     map[int]chan capi.Event
	nextSubID int
}

func newClusterCache() *clusterCache {
	return &clusterCache{byKey: map[string]capi.Cluster{}, subs: map[int]chan capi.Event{}}
}

func key(c capi.Cluster) string { return c.Namespace + "/" + c.Name }

// replaceAll resets the cache to exactly the given clusters (used after a LIST).
func (c *clusterCache) replaceAll(clusters []capi.Cluster) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byKey = make(map[string]capi.Cluster, len(clusters))
	for _, cl := range clusters {
		c.byKey[key(cl)] = cl
	}
}

// snapshot returns a sorted (namespace, name) copy of the current clusters.
func (c *clusterCache) snapshot() []capi.Cluster {
	c.mu.Lock()
	out := make([]capi.Cluster, 0, len(c.byKey))
	for _, cl := range c.byKey {
		out = append(out, cl)
	}
	c.mu.Unlock()
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// applyEvent updates the cache and notifies subscribers (non-blocking; a slow
// subscriber may drop an event — SSE clients re-fetch the snapshot on connect).
func (c *clusterCache) applyEvent(ev capi.Event) {
	c.mu.Lock()
	switch ev.Type {
	case capi.EventAdded, capi.EventModified:
		c.byKey[key(ev.Cluster)] = ev.Cluster
	case capi.EventDeleted:
		delete(c.byKey, key(ev.Cluster))
	case capi.EventError:
		c.mu.Unlock()
		return
	}
	subs := make([]chan capi.Event, 0, len(c.subs))
	for _, ch := range c.subs {
		subs = append(subs, ch)
	}
	c.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- ev:
		default: // drop on slow subscriber
		}
	}
}

// subscribe returns a receive-only event channel and an unsubscribe func.
// The channel is closed by unsub.
func (c *clusterCache) subscribe() (<-chan capi.Event, func()) {
	c.mu.Lock()
	id := c.nextSubID
	c.nextSubID++
	ch := make(chan capi.Event, 32)
	c.subs[id] = ch
	c.mu.Unlock()
	return ch, func() {
		c.mu.Lock()
		if cur, ok := c.subs[id]; ok {
			delete(c.subs, id)
			close(cur)
		}
		c.mu.Unlock()
	}
}
```

- [ ] **Step 4:** `go test ./internal/web -v` → PASS.
- [ ] **Step 5:** `go vet ./...` clean. Commit:
```bash
git add internal/web/clustercache.go internal/web/clustercache_test.go
git commit -m "feat(web): clusterCache with snapshot + event fan-out"
```

---

## Task 3: HTTP server skeleton + middleware + /health

**Files:**
- Create: `internal/web/server.go`
- Create: `internal/web/middleware.go`
- Create: `internal/web/server_test.go`

- [ ] **Step 1: Failing test** — `internal/web/server_test.go`:

```go
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
```

- [ ] **Step 2:** `go test ./internal/web -v` → FAIL.
- [ ] **Step 3: Implement** — `internal/web/server.go`:

```go
package web

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/v4run/kapish/internal/capi"
	kconfig "github.com/v4run/kapish/internal/config"
)

// Options configure New.
type Options struct {
	CapiClient  *capi.Client
	AppConfig   kconfig.Config
	MgmtContext string
	ConfigPath  string // for config PUT; empty disables persistence
	BindAddr    string // default "127.0.0.1"
	Port        int    // 0 = pick free port at Listen time
}

// Server is the kapish web server.
type Server struct {
	opts  Options
	cache *clusterCache
	mux   *http.ServeMux
	// ln is the listener once Listen() is called.
	ln net.Listener
}

// New constructs a Server (does not start listening).
func New(opts Options) (*Server, error) {
	if opts.BindAddr == "" {
		opts.BindAddr = "127.0.0.1"
	}
	s := &Server{
		opts:  opts,
		cache: newClusterCache(),
		mux:   http.NewServeMux(),
	}
	s.routes()
	return s, nil
}

// Handler returns the fully-wrapped http.Handler (mux + middleware). Exposed
// for tests via httptest.
func (s *Server) Handler() http.Handler {
	return withMiddleware(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	// More routes added in later tasks.
	// Catch-all for unknown /api/v1/ paths -> 404 JSON.
	s.mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// Cache exposes the cluster cache (used by serve to seed/feed it).
func (s *Server) Cache() *clusterCache { return s.cache }

var _ = fmt.Sprintf // placeholder; remove if fmt unused
```

(Remove the `fmt` import + `var _ =` line if `fmt` ends up unused.)

`internal/web/middleware.go`:

```go
package web

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// withMiddleware wraps h with security headers, same-origin CORS, and request
// logging.
func withMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Same-origin CORS: only echo ACAO for requests whose Origin host
		// matches the request Host (loopback). Cross-origin gets no ACAO.
		if origin := r.Header.Get("Origin"); origin != "" {
			if sameOrigin(origin, r.Host) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
		}

		start := time.Now()
		h.ServeHTTP(w, r)
		slog.Info("http", "method", r.Method, "path", r.URL.Path, "dur", time.Since(start).String())
	})
}

func sameOrigin(origin, host string) bool {
	// origin is like "http://127.0.0.1:8080"; strip the scheme.
	i := strings.Index(origin, "://")
	if i < 0 {
		return false
	}
	return origin[i+3:] == host
}
```

- [ ] **Step 4:** `go test ./internal/web -v` → PASS.
- [ ] **Step 5:** `go vet ./...` clean. Commit:
```bash
git add internal/web/server.go internal/web/middleware.go internal/web/server_test.go
git commit -m "feat(web): server skeleton, middleware, /health"
```

---

## Task 4: GET /api/v1/clusters

**Files:**
- Modify: `internal/web/server.go`
- Create: `internal/web/clusters.go`
- Create: `internal/web/clusters_test.go`

- [ ] **Step 1: Failing test** — `internal/web/clusters_test.go`:

```go
package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/v4run/kapish/internal/capi"
)

func TestGetClusters_ReturnsSnapshot(t *testing.T) {
	s := newTestServer(t)
	s.Cache().replaceAll([]capi.Cluster{
		{Name: "prod-eu-1", Namespace: "prod", Phase: "Provisioned", Provider: "aws", K8sVersion: "v1.30.2"},
		{Name: "stg-1", Namespace: "staging", Phase: "Pending"},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters", nil)
	s.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var got struct {
		Clusters []struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
			Phase     string `json:"phase"`
			Provider  string `json:"provider"`
			Version   string `json:"version"`
		} `json:"clusters"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Clusters, 2)
	assert.Equal(t, "prod-eu-1", got.Clusters[0].Name)
	assert.Equal(t, "prod", got.Clusters[0].Namespace)
	assert.Equal(t, "Provisioned", got.Clusters[0].Phase)
	assert.Equal(t, "aws", got.Clusters[0].Provider)
	assert.Equal(t, "v1.30.2", got.Clusters[0].Version)
}

func TestGetClusters_EmptyIsEmptyArrayNotNull(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters", nil)
	s.Handler().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"clusters":[]`)
}
```

- [ ] **Step 2:** `go test ./internal/web -v` → FAIL.
- [ ] **Step 3: Implement** — `internal/web/clusters.go`:

```go
package web

import (
	"net/http"

	"github.com/v4run/kapish/internal/capi"
)

// clusterDTO is the JSON shape returned to the browser.
type clusterDTO struct {
	Name                string `json:"name"`
	Namespace           string `json:"namespace"`
	Phase               string `json:"phase"`
	ControlPlaneReady   bool   `json:"controlPlaneReady"`
	InfrastructureReady bool   `json:"infrastructureReady"`
	Version             string `json:"version"`
	Provider            string `json:"provider"`
	AgeSeconds          int64  `json:"ageSeconds"`
}

func toDTO(c capi.Cluster) clusterDTO {
	d := clusterDTO{
		Name:                c.Name,
		Namespace:           c.Namespace,
		Phase:               c.Phase,
		ControlPlaneReady:   c.ControlPlaneReady,
		InfrastructureReady: c.InfrastructureReady,
		Version:             c.K8sVersion,
		Provider:            c.Provider,
	}
	if !c.CreationTimestamp.IsZero() {
		d.AgeSeconds = int64(timeSince(c.CreationTimestamp).Seconds())
	}
	return d
}

func (s *Server) handleGetClusters(w http.ResponseWriter, r *http.Request) {
	snap := s.cache.snapshot()
	out := make([]clusterDTO, 0, len(snap))
	for _, c := range snap {
		out = append(out, toDTO(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"clusters": out})
}
```

Add a tiny `timeSince` indirection (so tests aren't time-flaky if needed) — actually just use `time.Since` directly; declare:
```go
import "time"
func timeSince(t time.Time) time.Duration { return time.Since(t) }
```
(Or inline `time.Since(c.CreationTimestamp)` and skip the helper. Keep it simple — inline it and drop `timeSince`.)

In `server.go`'s `routes()`, add: `s.mux.HandleFunc("GET /api/v1/clusters", s.handleGetClusters)`.

- [ ] **Step 4:** `go test ./internal/web -v` → PASS. Note the `"clusters":[]` test relies on `make([]clusterDTO, 0, ...)` (not nil) so `json.Marshal` emits `[]` not `null`.
- [ ] **Step 5:** `go vet ./...` clean. Commit:
```bash
git add internal/web/clusters.go internal/web/clusters_test.go internal/web/server.go
git commit -m "feat(web): GET /api/v1/clusters snapshot endpoint"
```

---

## Task 5: GET /api/v1/clusters/stream (SSE)

**Files:**
- Modify: `internal/web/server.go`
- Create: `internal/web/sse.go`
- Create: `internal/web/sse_test.go`

- [ ] **Step 1: Failing test** — `internal/web/sse_test.go`:

```go
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
```

- [ ] **Step 2:** `go test ./internal/web -run TestSSE -v` → FAIL.
- [ ] **Step 3: Implement** — `internal/web/sse.go`:

```go
package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/v4run/kapish/internal/capi"
)

func (s *Server) handleClustersStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	sub, unsub := s.cache.subscribe()
	defer unsub()

	// Initial: tell the client to re-fetch the snapshot (simplest & robust).
	fmt.Fprintf(w, "event: sync\ndata: {}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-sub:
			if !ok {
				return
			}
			b, _ := json.Marshal(sseEventDTO(ev))
			fmt.Fprintf(w, "event: cluster\ndata: %s\n\n", b)
			flusher.Flush()
		}
	}
}

type sseEvent struct {
	Type    string     `json:"type"` // "added" | "modified" | "deleted"
	Cluster clusterDTO `json:"cluster"`
}

func sseEventDTO(ev capi.Event) sseEvent {
	t := "modified"
	switch ev.Type {
	case capi.EventAdded:
		t = "added"
	case capi.EventDeleted:
		t = "deleted"
	}
	return sseEvent{Type: t, Cluster: toDTO(ev.Cluster)}
}
```

In `server.go`'s `routes()`: `s.mux.HandleFunc("GET /api/v1/clusters/stream", s.handleClustersStream)`.

- [ ] **Step 4:** `go test ./internal/web -v` → PASS.
- [ ] **Step 5:** `go vet ./...` clean. Commit:
```bash
git add internal/web/sse.go internal/web/sse_test.go internal/web/server.go
git commit -m "feat(web): GET /api/v1/clusters/stream SSE endpoint"
```

---

## Task 6: GET/PUT /api/v1/config

**Files:**
- Modify: `internal/web/server.go`
- Create: `internal/web/config_api.go`
- Create: `internal/web/config_api_test.go`

- [ ] **Step 1: Failing test** — `internal/web/config_api_test.go`:

```go
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
```

- [ ] **Step 2:** `go test ./internal/web -run TestGetConfig -v` and `TestPutConfig` → FAIL.
- [ ] **Step 3: Implement** — `internal/web/config_api.go`. GET marshals `s.opts.AppConfig` to JSON. PUT decodes JSON into a `kconfig.Config`, runs `kconfig.Validate`, on failure → 400 with the error string; on success, updates `s.opts.AppConfig` in memory and (if `ConfigPath != ""`) writes via `kconfig.WriteToFile`. Note: PUT receives JSON but `WriteToFile` does YAML round-trip — that's fine, `WriteToFile` re-marshals the struct (the comment-preservation only kicks in if the YAML file already exists; the new struct values still get patched in). Wire `s.mux.HandleFunc("GET /api/v1/config", ...)` and `s.mux.HandleFunc("PUT /api/v1/config", ...)`.

```go
package web

import (
	"encoding/json"
	"net/http"

	kconfig "github.com/v4run/kapish/internal/config"
)

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.opts.AppConfig)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var incoming kconfig.Config
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}
	if err := kconfig.Validate(incoming); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	s.opts.AppConfig = incoming
	if s.opts.ConfigPath != "" {
		if err := kconfig.WriteToFile(s.opts.ConfigPath, incoming); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}
```

- [ ] **Step 4:** `go test ./internal/web -v` → PASS.
- [ ] **Step 5:** `go vet ./...` clean. Commit:
```bash
git add internal/web/config_api.go internal/web/config_api_test.go internal/web/server.go
git commit -m "feat(web): GET/PUT /api/v1/config endpoints"
```

---

## Task 7: GET /api/v1/mgmts + PUT /api/v1/mgmts/current

**Files:**
- Modify: `internal/web/server.go`
- Create: `internal/web/mgmts_api.go`
- Create: `internal/web/mgmts_api_test.go`

- [ ] **Step 1: Failing test** — `internal/web/mgmts_api_test.go`:

```go
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
```

> Note: `PUT /api/v1/mgmts/current` with a *valid* name would, in production, rebuild the capi client + reseed the cache. That's hard to unit-test without a live cluster. The test above only checks the rejection path. The implementation of the happy path: if `CapiClient` rebuild fails (e.g. unreachable), return 502; otherwise swap the client, update `Current`, reseed the cache via a LIST, return 200. For Plan 4, if `CapiClient` is nil (test mode), just update `Current` and return 200 without a real reconnect — gate the reconnect on `s.opts.CapiClient != nil`.

- [ ] **Step 2:** `go test ./internal/web -run TestGetMgmts -v` and `TestPutMgmtsCurrent` → FAIL.
- [ ] **Step 3: Implement** — `internal/web/mgmts_api.go`:

```go
package web

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/v4run/kapish/internal/capi"
)

func (s *Server) handleGetMgmts(w http.ResponseWriter, r *http.Request) {
	type entry struct {
		Name      string `json:"name"`
		Context   string `json:"context,omitempty"`
		Namespace string `json:"namespace,omitempty"`
	}
	mc := s.opts.AppConfig.ManagementClusters
	entries := make([]entry, 0, len(mc.Entries))
	for _, e := range mc.Entries {
		entries = append(entries, entry{Name: e.Name, Context: e.Context, Namespace: e.Namespace})
	}
	writeJSON(w, http.StatusOK, map[string]any{"current": mc.Current, "entries": entries})
}

func (s *Server) handlePutMgmtsCurrent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected {\"name\":\"...\"}"})
		return
	}
	mc := s.opts.AppConfig.ManagementClusters
	var picked *struct {
		Kubeconfig, Context, Namespace string
	}
	for _, e := range mc.Entries {
		if e.Name == body.Name {
			picked = &struct{ Kubeconfig, Context, Namespace string }{e.Kubeconfig, e.Context, e.Namespace}
			break
		}
	}
	if picked == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no such mgmt cluster: " + body.Name})
		return
	}

	// Rebuild the capi client if we have one (skip in test mode).
	if s.opts.CapiClient != nil {
		c, err := capi.NewClient(capi.Options{Kubeconfig: picked.Kubeconfig, Context: picked.Context, Namespace: picked.Namespace})
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cs, err := c.ListClusters(ctx)
		cancel()
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		s.opts.CapiClient = c
		s.opts.MgmtContext = c.Context()
		s.cache.replaceAll(cs)
	}
	s.opts.AppConfig.ManagementClusters.Current = body.Name
	writeJSON(w, http.StatusOK, map[string]string{"status": "switched", "current": body.Name})
}
```

In `server.go`: `s.mux.HandleFunc("GET /api/v1/mgmts", s.handleGetMgmts)` and `s.mux.HandleFunc("PUT /api/v1/mgmts/current", s.handlePutMgmtsCurrent)`.

- [ ] **Step 4:** `go test ./internal/web -v` → PASS.
- [ ] **Step 5:** `go vet ./...` clean. Commit:
```bash
git add internal/web/mgmts_api.go internal/web/mgmts_api_test.go internal/web/server.go
git commit -m "feat(web): GET /api/v1/mgmts + PUT /api/v1/mgmts/current"
```

---

## Task 8: sessionStore + POST /api/v1/sessions

**Files:**
- Modify: `internal/web/server.go`
- Create: `internal/web/sessions.go`
- Create: `internal/web/sessions_test.go`

- [ ] **Step 1: Failing test** — `internal/web/sessions_test.go`:

```go
package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

// Confirm the POST response shape when it would succeed — we can't actually
// succeed without a capi client, so this is a structural check only via the
// 5xx test above. Decoding shape is tested in the e2e WS test (Task 9).
var _ = json.Marshal
```

- [ ] **Step 2:** `go test ./internal/web -run "TestPostSessions|TestSessionStore" -v` → FAIL.
- [ ] **Step 3: Implement** — `internal/web/sessions.go`:

```go
package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/v4run/kapish/internal/shell"
)

// ptySession holds a prepared (not yet started) shell for a browser terminal.
type ptySession struct {
	cluster   string
	namespace string
	plan      *shell.SpawnPlan // Cmd not yet started
	created   time.Time
}

type sessionEntry struct {
	sess  *ptySession
	token string // one-time WebSocket token; cleared after consume
	used  bool
}

type sessionStore struct {
	mu sync.Mutex
	m  map[string]*sessionEntry
}

func newSessionStore() *sessionStore { return &sessionStore{m: map[string]*sessionEntry{}} }

func (s *sessionStore) create(sess *ptySession) (id, token string) {
	id = uuid.NewString()
	token = uuid.NewString()
	s.mu.Lock()
	s.m[id] = &sessionEntry{sess: sess, token: token}
	s.mu.Unlock()
	return id, token
}

// consumeToken returns the session if id+token match and the token hasn't been
// used. The token becomes invalid after one successful consume.
func (s *sessionStore) consumeToken(id, token string) (*ptySession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.m[id]
	if !ok || e.used || e.token == "" || e.token != token {
		return nil, false
	}
	e.used = true
	e.token = ""
	return e.sess, true
}

func (s *sessionStore) remove(id string) {
	s.mu.Lock()
	delete(s.m, id)
	s.mu.Unlock()
}

func (s *Server) handlePostSessions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Namespace string `json:"namespace"`
		Cluster   string `json:"cluster"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Cluster == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected {\"namespace\":\"...\",\"cluster\":\"...\"}"})
		return
	}
	if s.opts.CapiClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no management cluster connection"})
		return
	}
	ctx, cancel := contextWithTimeout(15 * time.Second)
	kc, err := s.opts.CapiClient.FetchKubeconfig(ctx, body.Namespace, body.Cluster)
	cancel()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	app := s.opts.AppConfig
	opts := shell.Options{
		PathToShell:    app.Shell.Command,
		Cwd:            app.Shell.Cwd,
		Env:            app.Shell.Env,
		Aliases:        app.Shell.Aliases,
		PromptTemplate: app.Shell.Prompt,
		PromptTokens: shell.PromptTokens{
			Cluster: body.Cluster, Namespace: body.Namespace, Ctx: s.opts.MgmtContext, Now: time.Now(),
		},
	}
	plan, err := shell.PrepareSpawn(opts, kc)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	id, tok := s.sessions.create(&ptySession{cluster: body.Cluster, namespace: body.Namespace, plan: plan, created: time.Now()})
	writeJSON(w, http.StatusOK, map[string]string{
		"sessionId": id,
		"wsUrl":     fmt.Sprintf("/api/v1/sessions/%s/ws", id),
		"wsToken":   tok,
	})
	_ = exec.Cmd{} // keep os/exec import if unused after edits; remove if not needed
}
```

(Drop the trailing `_ = exec.Cmd{}` and the `os/exec` import if unused.)

Add `sessions *sessionStore` to the `Server` struct; init it in `New`. Add `contextWithTimeout` helper somewhere (`internal/web/server.go`):
```go
import "context"
func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
```
(Or just call `context.WithTimeout` inline — keep it simple.)

In `server.go`'s `routes()`: `s.mux.HandleFunc("POST /api/v1/sessions", s.handlePostSessions)`.

- [ ] **Step 4:** `go test ./internal/web -v` → PASS.
- [ ] **Step 5:** `go vet ./...` clean. Commit:
```bash
git add internal/web/sessions.go internal/web/sessions_test.go internal/web/server.go
git commit -m "feat(web): sessionStore + POST /api/v1/sessions"
```

---

## Task 9: WebSocket-PTY bridge — GET /api/v1/sessions/{id}/ws

**Files:**
- Modify: `internal/web/server.go`
- Create: `internal/web/ws.go`
- Create: `internal/web/ws_test.go`

- [ ] **Step 1: Failing test** — `internal/web/ws_test.go`. This test uses a real `/bin/sh` (or `bash`) as the "shell", connects via `coder/websocket`'s client, writes a command, and reads the output back through the PTY:

```go
package web

import (
	"context"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/v4run/kapish/internal/shell"
)

func TestWebSocketPTY_EchoesShellOutput(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not on PATH")
	}

	s := newTestServer(t)
	// Manually create a session whose plan runs bash with our init.
	plan, err := shell.PrepareSpawn(shell.Options{PathToShell: bash}, []byte("# kc\n"))
	require.NoError(t, err)
	id, tok := s.sessions.create(&ptySession{cluster: "c", namespace: "ns", plan: plan, created: time.Now()})

	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/sessions/" + id + "/ws?token=" + tok

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send a stdin frame: prefix 0x00 + "echo KAPISH_OK\n"
	stdin := append([]byte{0x00}, []byte("echo KAPISH_OK\n")...)
	require.NoError(t, conn.Write(ctx, websocket.MessageBinary, stdin))

	// Read frames until we see "KAPISH_OK".
	deadline := time.Now().Add(4 * time.Second)
	found := false
	for time.Now().Before(deadline) {
		readCtx, c2 := context.WithTimeout(ctx, time.Second)
		typ, data, err := conn.Read(readCtx)
		c2()
		if err != nil {
			continue
		}
		if typ == websocket.MessageBinary && len(data) > 0 && data[0] == 0x00 {
			if strings.Contains(string(data[1:]), "KAPISH_OK") {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "expected to see KAPISH_OK in the PTY output")

	// Tell bash to exit so the goroutines wind down.
	_ = conn.Write(ctx, websocket.MessageBinary, append([]byte{0x00}, []byte("exit\n")...))
}

func TestWebSocketPTY_BadTokenRejected(t *testing.T) {
	s := newTestServer(t)
	id, _ := s.sessions.create(&ptySession{created: time.Now()})

	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/sessions/" + id + "/ws?token=wrong"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if err == nil {
		conn.Close(websocket.StatusNormalClosure, "")
		t.Fatalf("expected dial to fail with bad token")
	}
	// coder/websocket returns the HTTP response on a failed handshake.
	if resp != nil {
		assert.GreaterOrEqual(t, resp.StatusCode, 400)
	}
}
```

- [ ] **Step 2:** `go test ./internal/web -run TestWebSocketPTY -v` → FAIL.
- [ ] **Step 3: Implement** — `internal/web/ws.go`. The handler:
  1. Extract `{id}` from the path and `token` from the query.
  2. `s.sessions.consumeToken(id, token)` — on miss, `http.Error(w, "...", 403)` BEFORE accepting the WS (so the dial fails with 403). Use `websocket.Accept` only after the token check.
  3. `pty.Start(sess.plan.Cmd)` → `(*os.File, error)`. On error, accept the WS just to send a close, or just `http.Error` if not yet accepted; simplest: do `pty.Start` before `websocket.Accept` and `http.Error` 500 on failure.
  4. `defer sess.plan.Cleanup()`, `defer ptmx.Close()`, `defer s.sessions.remove(id)`, and `defer func(){ _ = sess.plan.Cmd.Process.Kill() }()` (best-effort).
  5. Two goroutines: PTY→WS (read PTY into a buffer, write `append([]byte{0x00}, buf...)` as a binary message) and WS→PTY (read messages; if `data[0]==0x00` write `data[1:]` to PTY; if `data[0]==0x01` JSON-decode `{cols,rows}` and `pty.Setsize`; if `data[0]==0x02` write a `[]byte{0x02}` pong back).
  6. When either side closes, cancel the other, SIGHUP the process group (`syscall.Kill(-cmd.Process.Pid, syscall.SIGHUP)` — note the negative pid for the group; the cmd was started by `pty.Start` which sets `Setsid`), wait briefly, then `Kill`.

```go
package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/coder/websocket"
	"github.com/creack/pty"
)

func (s *Server) handleSessionWS(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	token := r.URL.Query().Get("token")
	sess, ok := s.sessions.consumeToken(id, token)
	if !ok {
		http.Error(w, "invalid or expired session token", http.StatusForbidden)
		return
	}
	defer s.sessions.remove(id)
	defer func() { _ = sess.plan.Cleanup() }()

	ptmx, err := pty.Start(sess.plan.Cmd)
	if err != nil {
		http.Error(w, "failed to allocate pty: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer ptmx.Close()

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Same-origin is enforced by middleware + InsecureSkipVerify=false default;
		// coder/websocket checks the Origin header against the Host by default.
	})
	if err != nil {
		_ = sess.plan.Cmd.Process.Kill()
		return
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// PTY -> WS
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				msg := append([]byte{0x00}, buf[:n]...)
				if werr := c.Write(ctx, websocket.MessageBinary, msg); werr != nil {
					cancel()
					return
				}
			}
			if err != nil {
				cancel()
				return
			}
		}
	}()

	// WS -> PTY
	for {
		typ, data, err := c.Read(ctx)
		if err != nil {
			break
		}
		if typ != websocket.MessageBinary || len(data) == 0 {
			continue
		}
		switch data[0] {
		case 0x00: // stdin
			_, _ = ptmx.Write(data[1:])
		case 0x01: // resize {cols,rows}
			var sz struct {
				Cols uint16 `json:"cols"`
				Rows uint16 `json:"rows"`
			}
			if json.Unmarshal(data[1:], &sz) == nil {
				_ = pty.Setsize(ptmx, &pty.Winsize{Cols: sz.Cols, Rows: sz.Rows})
			}
		case 0x02: // ping -> pong
			_ = c.Write(ctx, websocket.MessageBinary, []byte{0x02})
		}
	}

	// Wind down the shell.
	if sess.plan.Cmd.Process != nil {
		pid := sess.plan.Cmd.Process.Pid
		_ = syscall.Kill(-pid, syscall.SIGHUP)
		select {
		case <-time.After(2 * time.Second):
			_ = sess.plan.Cmd.Process.Kill()
		case <-waitDone(sess.plan.Cmd):
		}
	}
	_ = io.Discard // keep io import if unused; remove otherwise
	_ = strings.TrimSpace
	_ = os.Stdin
}

func waitDone(cmd interface{ Wait() error }) <-chan struct{} {
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	return done
}
```

> **Clean-up note:** the `_ = io.Discard`, `_ = strings.TrimSpace`, `_ = os.Stdin` lines are placeholders so the imports compile while you wire things — remove them and the unused imports (`io`, `strings`, `os`) once the real code doesn't need them. `pty.Start` already calls `Setsid` so the negative-pid SIGHUP targets the shell's process group. The `waitDone` helper takes the `*exec.Cmd` via an interface to avoid an import; you can also just take `*exec.Cmd` directly and import `os/exec`.

In `server.go`'s `routes()`: `s.mux.HandleFunc("GET /api/v1/sessions/{id}/ws", s.handleSessionWS)`.

> **Important — Origin check & httptest:** `coder/websocket`'s `Accept` checks the `Origin` header against the request `Host` by default and rejects mismatches with 403. In the e2e test, `websocket.Dial` sets `Origin` to the ws:// URL's host automatically, so it matches `httptest.NewServer`'s host — the handshake succeeds. The `TestWebSocketPTY_BadTokenRejected` test fails *before* `Accept` (token check), so it gets a plain 403 from `http.Error`. Both behaviors are what the tests expect. If `Accept` unexpectedly rejects in the echo test, set `AcceptOptions{OriginPatterns: []string{"*"}}` ONLY in a test-mode flag — but try without first.

- [ ] **Step 4:** `go test ./internal/web -run TestWebSocketPTY -v` → PASS (or skip if bash absent). If the echo test is flaky on timing, increase the deadline; don't disable it.
- [ ] **Step 5:** `go vet ./...` clean. `go test ./... -count=1` green. Commit:
```bash
git add internal/web/ws.go internal/web/ws_test.go internal/web/server.go
git commit -m "feat(web): WebSocket-PTY bridge for shell sessions"
```

---

## Task 10: Embed frontend assets (placeholder) + serve at /

**Files:**
- Create: `internal/web/frontend/dist/index.html` (placeholder)
- Create: `internal/web/embed.go`
- Modify: `internal/web/server.go`
- Create: `internal/web/embed_test.go`

- [ ] **Step 1:** Create the placeholder `internal/web/frontend/dist/index.html`:

```html
<!doctype html>
<html>
<head><meta charset="utf-8"><title>kapish</title></head>
<body>
<h1>kapish</h1>
<p>The web UI is built in a later phase. The API is live at <code>/api/v1/</code>.</p>
</body>
</html>
```

- [ ] **Step 2:** `internal/web/embed.go`:

```go
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

// frontendRoot returns an fs.FS rooted at frontend/dist.
func frontendRoot() fs.FS {
	sub, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		panic(err) // build-time guarantee; can't happen if //go:embed succeeded
	}
	return sub
}
```

- [ ] **Step 3:** In `server.go`'s `routes()`, add a fallback handler for `/` (must come after the `/api/v1/` routes so API paths win): `s.mux.Handle("/", http.FileServer(http.FS(frontendRoot())))`. (Go 1.22's `ServeMux` precedence: more-specific patterns win, so `/api/v1/health` etc. take priority over `/`.)

- [ ] **Step 4:** `internal/web/embed_test.go`:

```go
package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServesIndexAtRoot(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "kapish")
}

func TestAPIRoutesStillWinOverRoot(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ok")
}
```

- [ ] **Step 5:** `go test ./internal/web -v` → PASS. `go vet ./...` clean. Commit:
```bash
git add internal/web/embed.go internal/web/embed_test.go internal/web/server.go internal/web/frontend/dist/index.html
git commit -m "feat(web): embed placeholder frontend, serve at /"
```

---

## Task 11: Server.Listen / Run / Shutdown

**Files:**
- Modify: `internal/web/server.go`
- Create: `internal/web/run_test.go`

- [ ] **Step 1: Failing test** — `internal/web/run_test.go`:

```go
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
```

- [ ] **Step 2:** `go test ./internal/web -run TestServer_ListenAndShutdown -v` → FAIL.
- [ ] **Step 3: Implement** — add to `server.go`:

```go
import (
	"context"
	"net"
	"net/http"
)

// Listen binds the configured addr:port. Returns the actual address (useful
// when Port==0). Call before Serve.
func (s *Server) Listen() (string, error) {
	addr := s.opts.BindAddr
	if s.opts.Port != 0 {
		addr = net.JoinHostPort(s.opts.BindAddr, fmt.Sprintf("%d", s.opts.Port))
	} else {
		addr = net.JoinHostPort(s.opts.BindAddr, "0")
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", fmt.Errorf("web: listen %s: %w", addr, err)
	}
	s.ln = ln
	s.httpSrv = &http.Server{Handler: s.Handler()}
	return ln.Addr().String(), nil
}

// Serve runs until Shutdown is called. Returns http.ErrServerClosed on clean shutdown.
func (s *Server) Serve() error {
	if s.ln == nil {
		return fmt.Errorf("web: Serve called before Listen")
	}
	return s.httpSrv.Serve(s.ln)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpSrv == nil {
		return nil
	}
	return s.httpSrv.Shutdown(ctx)
}
```

Add `httpSrv *http.Server` to the `Server` struct.

- [ ] **Step 4:** `go test ./internal/web -v` → PASS. `go vet ./...` clean.
- [ ] **Step 5:** Commit:
```bash
git add internal/web/server.go internal/web/run_test.go
git commit -m "feat(web): Listen/Serve/Shutdown lifecycle"
```

---

## Task 12: `kapish serve` command

**Files:**
- Create: `cmd/kapish/serve.go`
- Modify: `cmd/kapish/root.go`

- [ ] **Step 1: Implement** — `cmd/kapish/serve.go`. `newServeCmd()` returns a cobra command `serve` with flags `--port int` (default 0), `--bind string` (default "127.0.0.1"), `--no-open bool` (default false), `--dev bool` (default false; reserved for Plan 5's Vite proxy — for now just print a note if set). The `RunE`:
  1. `readGlobalFlags`, resolve+load+override+validate config (same as `runTUI`).
  2. If `--bind` is non-loopback (`!= "127.0.0.1" && != "localhost" && != "::1"`), print a warning to stderr.
  3. Build the `capi.Client` from the current mgmt entry / flags (reuse `indexOfCurrentEntry`, `currentNamespace` from `tui.go`).
  4. `web.New(web.Options{CapiClient, AppConfig, MgmtContext: client.Context(), ConfigPath: cfgPath, BindAddr, Port})`.
  5. `addr, err := srv.Listen()`; print `kapish web UI: http://<addr>/`.
  6. Seed the cache: `cs, _ := client.ListClusters(ctx); srv.Cache().replaceAll(cs)`. Start a watch goroutine that pumps `srv.Cache().applyEvent(ev)` for each event from `client.WatchClusters(ctx)`, with a simple reconnect-on-error loop (re-call WatchClusters after a 1s sleep). Also a periodic re-LIST every `AppConfig.UI.RefreshIntervalSec` to catch missed events.
  7. Unless `--no-open` (or `web.openBrowser == false`), open the browser to the URL (use a tiny cross-platform opener: `open` on darwin, `xdg-open` on linux, `cmd /c start` on windows — or just `exec.Command` per `runtime.GOOS`; keep it best-effort, ignore errors).
  8. `go srv.Serve()`; wait for SIGINT/SIGTERM (`signal.NotifyContext`); on signal, `srv.Shutdown(ctx)` with a 5s timeout.

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/v4run/kapish/internal/capi"
	kconfig "github.com/v4run/kapish/internal/config"
	"github.com/v4run/kapish/internal/web"
)

func newServeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "serve",
		Short: "Run the kapish web UI on localhost",
		RunE:  runServe,
	}
	c.Flags().Int("port", 0, "Port to bind (0 = pick a free port)")
	c.Flags().String("bind", "127.0.0.1", "Address to bind (non-loopback prints a warning)")
	c.Flags().Bool("no-open", false, "Don't open the browser automatically")
	c.Flags().Bool("dev", false, "Dev mode (reserved for Vite proxy; currently a no-op)")
	return c
}

func runServe(cmd *cobra.Command, args []string) error {
	g, err := readGlobalFlags(cmd)
	if err != nil {
		return err
	}
	port, _ := cmd.Flags().GetInt("port")
	bind, _ := cmd.Flags().GetString("bind")
	noOpen, _ := cmd.Flags().GetBool("no-open")

	cfgPath, err := kconfig.ResolvePath(kconfig.PathSources{
		Flag: g.ConfigPath, EnvVar: os.Getenv("KAPISH_CONFIG"),
		XDGConfigHome: os.Getenv("XDG_CONFIG_HOME"), Home: os.Getenv("HOME"),
	})
	if err != nil {
		return err
	}
	app, err := kconfig.LoadFromFile(cfgPath)
	if err != nil {
		return err
	}
	app = kconfig.ApplyOverrides(app, kconfig.FlagOverrides{
		Kubeconfig: g.Kubeconfig, Context: g.Context, OneShot: boolPtrIfSet(cmd, "one-shot", g.OneShot),
	})
	if err := kconfig.Validate(app); err != nil {
		return err
	}

	if bind != "127.0.0.1" && bind != "localhost" && bind != "::1" {
		fmt.Fprintf(os.Stderr, "warning: binding to %s exposes kapish with no authentication\n", bind)
	}

	mgmtKubeconfig, mgmtContext := g.Kubeconfig, g.Context
	mgmtNamespace := ""
	if idx := indexOfCurrentEntry(app); idx >= 0 {
		e := app.ManagementClusters.Entries[idx]
		if mgmtKubeconfig == "" {
			mgmtKubeconfig = e.Kubeconfig
		}
		if mgmtContext == "" {
			mgmtContext = e.Context
		}
		mgmtNamespace = e.Namespace
	}
	client, err := capi.NewClient(capi.Options{Kubeconfig: mgmtKubeconfig, Context: mgmtContext, Namespace: mgmtNamespace})
	if err != nil {
		return fmt.Errorf("connect to management cluster: %w", err)
	}

	srv, err := web.New(web.Options{
		CapiClient: client, AppConfig: app, MgmtContext: client.Context(),
		ConfigPath: cfgPath, BindAddr: bind, Port: port,
	})
	if err != nil {
		return err
	}
	addr, err := srv.Listen()
	if err != nil {
		return err
	}
	url := "http://" + addr + "/"
	fmt.Println("kapish web UI:", url)

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Seed + watch + periodic re-list.
	if cs, lerr := client.ListClusters(rootCtx); lerr == nil {
		srv.Cache().replaceAll(cs)
	}
	go func() {
		for rootCtx.Err() == nil {
			evs, werr := client.WatchClusters(rootCtx)
			if werr != nil {
				time.Sleep(time.Second)
				continue
			}
			for ev := range evs {
				srv.Cache().applyEvent(ev)
			}
			// channel closed; loop reconnects unless ctx done.
			time.Sleep(time.Second)
		}
	}()
	go func() {
		d := time.Duration(app.UI.RefreshIntervalSec) * time.Second
		if d <= 0 {
			d = 30 * time.Second
		}
		t := time.NewTicker(d)
		defer t.Stop()
		for {
			select {
			case <-rootCtx.Done():
				return
			case <-t.C:
				if cs, lerr := client.ListClusters(rootCtx); lerr == nil {
					srv.Cache().replaceAll(cs)
				}
			}
		}
	}()

	if !noOpen && app.Web.OpenBrowser {
		_ = openBrowser(url)
	}

	srvErr := make(chan error, 1)
	go func() { srvErr <- srv.Serve() }()

	select {
	case <-rootCtx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutCtx)
	case err := <-srvErr:
		return err
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
```

In `root.go`: `root.AddCommand(newServeCmd())`.

- [ ] **Step 2:** Build + smoke (no real cluster — use bad flags to confirm clean error):
```sh
go build ./cmd/kapish
./bin/kapish serve --kubeconfig /tmp/nope --context nope 2>&1 | head -3
```
Expected: `kapish: connect to management cluster: ...` — clean, no panic.

- [ ] **Step 3:** `go test ./... -count=1` green; `go vet ./...` clean; `go mod tidy` no-op.
- [ ] **Step 4:** Commit:
```bash
git add cmd/kapish/serve.go cmd/kapish/root.go
git commit -m "feat(cli): kapish serve command"
```

---

## Task 13: Full verification + final review

**Files:** none (verification only)

- [ ] **Step 1:** `make test` — all packages green (`cmd/kapish`, `internal/capi`, `internal/config`, `internal/kapishlog`, `internal/shell`, `internal/tui`, `internal/version`, `internal/web`).
- [ ] **Step 2:** `go vet ./... && go mod tidy` — clean, no diff.
- [ ] **Step 3:** `make build && ./bin/kapish --help` — `serve` listed under Available Commands; `./bin/kapish serve --help` shows the serve flags.
- [ ] **Step 4:** `go install ./cmd/kapish && "$(go env GOPATH)/bin/kapish" version` — still works.
- [ ] **Step 5:** Manual smoke (optional, no real cluster): `./bin/kapish serve --port 0 --no-open --kubeconfig /tmp/nope --context nope` should print a clean connect error and exit.
- [ ] **Step 6:** If `go mod tidy` changed anything, commit it:
```bash
git add go.mod go.sum
git commit -m "chore: tidy go.mod after Plan 4"
```

---

## Plan 4 exit criteria

- [ ] `make test` green across all packages, including `internal/web`.
- [ ] `internal/web` exposes `New(Options)`, `Server.{Handler, Listen, Serve, Shutdown, Cache}`.
- [ ] Endpoints work (verified via httptest): `GET /api/v1/health`, `GET /api/v1/clusters`, `GET /api/v1/clusters/stream` (SSE), `GET/PUT /api/v1/config`, `GET /api/v1/mgmts`, `PUT /api/v1/mgmts/current`, `POST /api/v1/sessions`, `GET /api/v1/sessions/{id}/ws` (WebSocket-PTY echoes shell output), `GET /` (placeholder index).
- [ ] Security: binds `127.0.0.1` by default; `X-Frame-Options: DENY`; same-origin CORS (no ACAO echo for cross-origin); one-time WebSocket token (single-use).
- [ ] `kapish serve` connects to the mgmt cluster, starts the server, prints the URL, seeds + watches the cluster cache, opens the browser (unless `--no-open`), and shuts down cleanly on SIGINT/SIGTERM; clean error (not panic) on connect failure.

When all boxes are checked, Plan 4 is done. Plan 5 (web frontend — drop in the Claude Design React components, wire to this API, xterm.js) is the final phase.
