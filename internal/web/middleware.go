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
