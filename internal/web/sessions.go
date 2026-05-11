package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

// evictStale removes sessions that were never connected (used==false) and are
// older than ttl. The SpawnPlan's temp dir is cleaned up best-effort.
func (s *sessionStore) evictStale(ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, e := range s.m {
		if !e.used && time.Since(e.sess.created) > ttl {
			if e.sess != nil && e.sess.plan != nil {
				_ = e.sess.plan.Cleanup()
			}
			delete(s.m, id)
		}
	}
}

// janitor runs a background loop that calls evictStale on each tick. It exits
// when ctx is cancelled. tickInterval is ttl/2, floored at 10s.
func (s *sessionStore) janitor(ctx context.Context, ttl time.Duration) {
	interval := ttl / 2
	if interval < 10*time.Second {
		interval = 10 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.evictStale(ttl)
		}
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
}
