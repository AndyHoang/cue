package log

import (
	"log/slog"
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	tests := map[string]slog.Level{
		"DEBUG":   slog.LevelDebug,
		"info":    slog.LevelInfo,
		"WARNING": slog.LevelWarn,
		"ERROR":   slog.LevelError,
		"bad":     slog.LevelInfo,
	}
	for input, want := range tests {
		if got := parseLogLevel(input); got != want {
			t.Fatalf("parseLogLevel(%q)=%v want %v", input, got, want)
		}
	}
}
