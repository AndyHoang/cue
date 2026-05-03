package store

import (
	"testing"

	"github.com/SuperCoolPencil/cue/internal/domain"
)

func TestLibraryStorePersistsAndInvalidatesContent(t *testing.T) {
	base := t.TempDir()
	st, err := NewLibraryStore(base, "http://server")
	if err != nil {
		t.Fatal(err)
	}

	libs := []domain.Library{{ID: "movies", Name: "Movies", Type: "movie", UpdatedAt: 10}}
	movies := []*domain.MediaItem{{ID: "m1", Title: "Movie"}}
	if err := st.SaveLibraries(libs); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveMovies("movies", movies, 10); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewLibraryStore(base, "http://server")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })

	gotLibs, ok := reopened.GetLibraries()
	if !ok || len(gotLibs) != 1 || gotLibs[0].ID != "movies" {
		t.Fatalf("libraries = %#v, %v", gotLibs, ok)
	}
	gotMovies, ok := reopened.GetMovies("movies")
	if !ok || len(gotMovies) != 1 || gotMovies[0].ID != "m1" {
		t.Fatalf("movies = %#v, %v", gotMovies, ok)
	}
	if !reopened.IsValid("movies", 10) {
		t.Fatal("cache should be valid at matching timestamp")
	}
	reopened.InvalidateLibrary("movies")
	if _, ok := reopened.GetMovies("movies"); ok {
		t.Fatal("movies should be invalidated")
	}
	if reopened.IsValid("movies", 10) {
		t.Fatal("timestamp should be invalidated")
	}
}

func TestLibraryStoreMixedContentAndQueue(t *testing.T) {
	st, err := NewLibraryStore("", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	movie := &domain.MediaItem{ID: "m1", Title: "Movie"}
	show := &domain.Show{ID: "s1", Title: "Show"}
	if err := st.SaveMixedContent("mixed", []domain.ListItem{movie, show}, 1); err != nil {
		t.Fatal(err)
	}
	content, ok := st.GetMixedContent("mixed")
	if !ok || len(content) != 2 {
		t.Fatalf("mixed content = %#v, %v", content, ok)
	}
	if _, ok := content[0].(*domain.MediaItem); !ok {
		t.Fatalf("first item type = %T", content[0])
	}
	if _, ok := content[1].(*domain.Show); !ok {
		t.Fatalf("second item type = %T", content[1])
	}

	if err := st.SaveQueueItems([]*domain.MediaItem{movie}); err != nil {
		t.Fatal(err)
	}
	queue, ok := st.GetQueueItems()
	if !ok || len(queue) != 1 || queue[0].ID != "m1" {
		t.Fatalf("queue = %#v, %v", queue, ok)
	}
	st.InvalidateQueue()
	if _, ok := st.GetQueueItems(); ok {
		t.Fatal("queue should be invalidated")
	}
}
