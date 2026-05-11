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
