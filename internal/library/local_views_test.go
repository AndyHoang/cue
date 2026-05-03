package library

import (
	"log/slog"
	"testing"
	"time"

	"github.com/SuperCoolPencil/cue/internal/domain"
	"github.com/SuperCoolPencil/cue/internal/store"
)

func TestLocalViews(t *testing.T) {
	st, err := store.NewLibraryStore("", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	libs := []domain.Library{{ID: "movies", Name: "Movies", Type: "movie"}}
	if err := st.SaveLibraries(libs); err != nil {
		t.Fatal(err)
	}
	items := []*domain.MediaItem{
		{ID: "old", Title: "Old", AddedAt: 10, UpdatedAt: 10, IsPlayed: true},
		{ID: "recent", Title: "Recent", AddedAt: 30, UpdatedAt: 30, Rating: 8.5, Height: 2160},
		{ID: "progress", Title: "Progress", AddedAt: 20, UpdatedAt: 40, ViewOffset: 5 * time.Minute},
	}
	if err := st.SaveMovies("movies", items, 1); err != nil {
		t.Fatal(err)
	}

	svc := NewService(nil, st, slog.Default())

	continueItems := svc.ContinueWatching(0)
	if len(continueItems) != 1 || continueItems[0].ID != "progress" {
		t.Fatalf("continue watching = %#v", continueItems)
	}

	recent := svc.RecentlyAdded(2)
	if len(recent) != 2 || recent[0].ID != "recent" || recent[1].ID != "progress" {
		t.Fatalf("recent = %#v", recent)
	}

	filtered := svc.SmartFiltered("4k", 0)
	if len(filtered) != 1 || filtered[0].ID != "recent" {
		t.Fatalf("4k filter = %#v", filtered)
	}
}
