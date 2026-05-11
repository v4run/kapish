package web

import (
	"encoding/json"
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
