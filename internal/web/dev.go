package web

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// devProxy returns a reverse proxy that forwards all requests to target (the
// Vite dev server URL, e.g. "http://127.0.0.1:5173"). Used in dev mode so that
// HMR and Vite's module graph are served directly; the Go server's /api routes
// remain more specific and are not proxied.
func devProxy(target string) (http.Handler, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("devProxy: parse %q: %w", target, err)
	}
	return httputil.NewSingleHostReverseProxy(u), nil
}
