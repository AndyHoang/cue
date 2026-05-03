package player

import (
	"log/slog"
	"testing"
)

func TestLookupSeekFlag(t *testing.T) {
	launcher := NewLauncher("", nil, "", slog.Default())
	if got := launcher.lookupSeekFlag("mpv"); got != "--start=%d" {
		t.Fatalf("mpv seek flag = %q", got)
	}
	if got := launcher.lookupSeekFlag("unknown-player"); got != "" {
		t.Fatalf("unknown seek flag = %q", got)
	}
}
