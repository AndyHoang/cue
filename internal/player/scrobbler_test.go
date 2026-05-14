package player

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/SuperCoolPencil/cue/internal/domain"
)

type mockPlaybackClient struct {
	domain.PlaybackClient
	marks    []string
	progress map[string]int64
	mu       sync.Mutex
}

func (m *mockPlaybackClient) MarkPlayed(ctx context.Context, itemID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.marks = append(m.marks, itemID)
	return nil
}

func (m *mockPlaybackClient) UpdateProgress(ctx context.Context, itemID string, pos int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.progress == nil {
		m.progress = make(map[string]int64)
	}
	m.progress[itemID] = pos
	return nil
}

func (m *mockPlaybackClient) GetMarks() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.marks...)
}

func (m *mockPlaybackClient) GetProgress(id string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.progress[id]
}

func TestScrobbler(t *testing.T) {
	// Setup mock MPV server
	tmpDir, err := os.MkdirTemp("", "cue-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	socketPath := filepath.Join(tmpDir, "mpv.sock")
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	// Mock server state
	serverState := struct {
		pos         float64
		playlistPos int
		mu          sync.Mutex
	}{
		pos:         10.0,
		playlistPos: 0,
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				dec := json.NewDecoder(c)
				for {
					var req struct {
						Command   []interface{} `json:"command"`
						RequestID int64         `json:"request_id"`
					}
					if err := dec.Decode(&req); err != nil {
						return
					}
					if len(req.Command) == 0 {
						continue
					}

					var resp interface{}
					cmd := req.Command[0].(string)
					switch cmd {
					case "get_property":
						prop := req.Command[1].(string)
						serverState.mu.Lock()
						switch prop {
						case "time-pos":
							resp = serverState.pos
						case "playlist-pos":
							resp = float64(serverState.playlistPos)
						}
						serverState.mu.Unlock()
					}

					res, _ := json.Marshal(map[string]interface{}{
						"data":       resp,
						"error":      "success",
						"request_id": req.RequestID,
					})
					_, _ = c.Write(append(res, '\n'))
				}
			}(conn)
		}
	}()

	client := &mockPlaybackClient{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := NewScrobbler(client, logger)
	s.interval = 50 * time.Millisecond // fast polling for tests

	items := []domain.MediaItem{
		{ID: "1", Title: "Ep 1", Duration: 100 * time.Second},
		{ID: "2", Title: "Ep 2", Duration: 100 * time.Second},
	}

	// Mock command that "runs" for a bit
	cmd := exec.Command("sleep", "1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	handle := s.Monitor(ctx, cmd, socketPath, 0, items...)

	// 1. Check progress reporting
	time.Sleep(200 * time.Millisecond)
	if p := client.GetProgress("1"); p == 0 {
		t.Error("expected progress update for item 1")
	}

	// 2. Change playlist position
	serverState.mu.Lock()
	serverState.playlistPos = 1
	serverState.pos = 5.0
	serverState.mu.Unlock()

	time.Sleep(200 * time.Millisecond)
	marks := client.GetMarks()
	found := false
	for _, m := range marks {
		if m == "1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected item 1 to be marked watched after playlist change")
	}

	// 3. Test auto-scrobble on exit (>90%)
	serverState.mu.Lock()
	serverState.pos = 95.0 // 95/100 = 95%
	serverState.mu.Unlock()

	time.Sleep(100 * time.Millisecond)
	cancel() // Stop monitoring
	_ = cmd.Process.Kill()

	select {
	case res := <-handle.ResultCh:
		if !res.AutoMarked {
			t.Error("expected auto-marked to be true at 95% progress")
		}
		marks = client.GetMarks()
		found = false
		for _, m := range marks {
			if m == "2" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected item 2 to be marked watched on exit")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for scrobble result")
	}
}
