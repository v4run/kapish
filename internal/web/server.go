package web

import (
	"context"
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
	opts     Options
	cache    *clusterCache
	sessions *sessionStore
	mux      *http.ServeMux
	// ln is the listener once Listen() is called.
	ln      net.Listener
	httpSrv *http.Server
}

// New constructs a Server (does not start listening).
func New(opts Options) (*Server, error) {
	if opts.BindAddr == "" {
		opts.BindAddr = "127.0.0.1"
	}
	s := &Server{
		opts:     opts,
		cache:    newClusterCache(),
		sessions: newSessionStore(),
		mux:      http.NewServeMux(),
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
	s.mux.HandleFunc("GET /api/v1/clusters", s.handleGetClusters)
	s.mux.HandleFunc("GET /api/v1/clusters/stream", s.handleClustersStream)
	s.mux.HandleFunc("GET /api/v1/config", s.handleGetConfig)
	s.mux.HandleFunc("PUT /api/v1/config", s.handlePutConfig)
	s.mux.HandleFunc("GET /api/v1/mgmts", s.handleGetMgmts)
	s.mux.HandleFunc("PUT /api/v1/mgmts/current", s.handlePutMgmtsCurrent)
	s.mux.HandleFunc("POST /api/v1/sessions", s.handlePostSessions)
	s.mux.HandleFunc("GET /api/v1/sessions/{id}/ws", s.handleSessionWS)
	// More routes added in later tasks.
	// Catch-all for unknown /api/v1/ paths -> 404 JSON.
	s.mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	})
	// Fallback: serve embedded frontend. API routes above are more specific and
	// win via Go 1.22 ServeMux precedence.
	s.mux.Handle("/", http.FileServer(http.FS(frontendRoot())))
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

// ReplaceAll resets the cluster cache to exactly the given clusters (used
// after a LIST from cmd/kapish/serve).
func (s *Server) ReplaceAll(clusters []capi.Cluster) { s.cache.replaceAll(clusters) }

// ApplyEvent applies a single cluster watch event to the cache (used by the
// watch goroutine in cmd/kapish/serve).
func (s *Server) ApplyEvent(ev capi.Event) { s.cache.applyEvent(ev) }

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
