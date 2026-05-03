package tui

import (
	"log/slog"
	"testing"
	"time"

	"github.com/SuperCoolPencil/cue/internal/config"
	"github.com/SuperCoolPencil/cue/internal/domain"
	"github.com/SuperCoolPencil/cue/internal/library"
	"github.com/SuperCoolPencil/cue/internal/playlist"
	"github.com/SuperCoolPencil/cue/internal/store"
	"github.com/SuperCoolPencil/cue/internal/tui/components"
)

func TestVirtualLibraryEntries(t *testing.T) {
	entries := virtualLibraryEntries()
	if len(entries) != 7 {
		t.Fatalf("virtual entries = %d", len(entries))
	}
	if entries[0].ID != continueLibraryID || entries[0].Name != "Continue Watching" {
		t.Fatalf("first virtual entry = %#v", entries[0])
	}
}

func TestDrillVirtualContinueWatching(t *testing.T) {
	st, _ := store.NewLibraryStore("", "")
	libs := []domain.Library{{ID: "movies", Name: "Movies", Type: "movie"}}
	_ = st.SaveLibraries(libs)
	_ = st.SaveMovies("movies", []*domain.MediaItem{{ID: "m1", Title: "Movie", ViewOffset: time.Minute}}, 1)

	model := newVirtualTestModel(st)
	root := components.NewLibraryColumn(model.allLibraryEntries())
	model.ColumnStack.Reset(root)

	result := model.drillVirtualLibrary(domain.Library{ID: continueLibraryID, Name: "Continue Watching", Type: "cue"}, 0)
	if result == nil || model.ColumnStack.Len() != 2 {
		t.Fatalf("virtual drill failed: %#v len=%d", result, model.ColumnStack.Len())
	}
	top := model.ColumnStack.Top()
	if top.ContentID() != continueLibraryID {
		t.Fatalf("content id = %q", top.ContentID())
	}
	if item := top.SelectedMediaItem(); item == nil || item.ID != "m1" {
		t.Fatalf("selected item = %#v", item)
	}
}

func newVirtualTestModel(st domain.Store) Model {
	libSvc := library.NewService(nil, st, slog.Default())
	playlistSvc := playlist.NewService(nil, st, slog.Default())
	return Model{
		State:           StateBrowsing,
		Store:           st,
		LibraryService:  libSvc,
		PlaylistService: playlistSvc,
		ColumnStack:     NewColumnStack(),
		LibraryStates:   make(map[string]components.LibrarySyncState),
		AppConfig:       config.DefaultConfig(),
	}
}
