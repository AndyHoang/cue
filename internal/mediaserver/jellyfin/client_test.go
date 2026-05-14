package jellyfin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientMarkPlayedUnplayed(t *testing.T) {
	var lastMethod string
	var lastPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastMethod = r.Method
		lastPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", "user123", nil)

	// Test MarkPlayed
	err := client.MarkPlayed(context.Background(), "item456")
	if err != nil {
		t.Fatalf("MarkPlayed failed: %v", err)
	}
	if lastMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", lastMethod)
	}
	if lastPath != "/Users/user123/PlayedItems/item456" {
		t.Errorf("expected /Users/user123/PlayedItems/item456, got %s", lastPath)
	}

	// Test MarkUnplayed
	err = client.MarkUnplayed(context.Background(), "item456")
	if err != nil {
		t.Fatalf("MarkUnplayed failed: %v", err)
	}
	if lastMethod != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", lastMethod)
	}
	if lastPath != "/Users/user123/PlayedItems/item456" {
		t.Errorf("expected /Users/user123/PlayedItems/item456, got %s", lastPath)
	}
}
