package web

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
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

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		_ = sess.plan.Cmd.Process.Kill()
		return
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// PTY -> WS goroutine
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

	// WS -> PTY main loop
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

	// Wind down the shell gracefully.
	if sess.plan.Cmd.Process != nil {
		pid := sess.plan.Cmd.Process.Pid
		_ = syscall.Kill(-pid, syscall.SIGHUP)
		select {
		case <-time.After(2 * time.Second):
			_ = sess.plan.Cmd.Process.Kill()
		case <-waitDone(sess.plan.Cmd):
		}
	}
}

// waitDone returns a channel that is closed when cmd.Wait() returns.
func waitDone(cmd *exec.Cmd) <-chan struct{} {
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	return done
}
