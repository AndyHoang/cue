package library

import (
	"context"
	"log/slog"
	"testing"

	"github.com/SuperCoolPencil/cue/internal/domain"
	"github.com/SuperCoolPencil/cue/internal/store"
)

type fakeLibraryClient struct {
	libs         []domain.Library
	moviePages   [][]*domain.MediaItem
	showPages    [][]*domain.Show
	mixedPages   [][]domain.ListItem
	seasons      []*domain.Season
	episodes     []*domain.MediaItem
	libraryCalls int
	movieCalls   int
}

func (f *fakeLibraryClient) GetLibraries(context.Context) ([]domain.Library, error) {
	f.libraryCalls++
	return f.libs, nil
}

func (f *fakeLibraryClient) GetMovies(_ context.Context, _ string, offset, limit int) ([]*domain.MediaItem, int, error) {
	f.movieCalls++
	idx := offset / limit
	if idx >= len(f.moviePages) {
		return nil, len(flattenMovies(f.moviePages)), nil
	}
	return f.moviePages[idx], len(flattenMovies(f.moviePages)), nil
}

func (f *fakeLibraryClient) GetShows(_ context.Context, _ string, _, _ int) ([]*domain.Show, int, error) {
	return flattenShows(f.showPages), len(flattenShows(f.showPages)), nil
}

func (f *fakeLibraryClient) GetMixedContent(_ context.Context, _ string, _, _ int) ([]domain.ListItem, int, error) {
	return flattenMixed(f.mixedPages), len(flattenMixed(f.mixedPages)), nil
}

func (f *fakeLibraryClient) GetSeasons(context.Context, string) ([]*domain.Season, error) {
	return f.seasons, nil
}

func (f *fakeLibraryClient) GetEpisodes(context.Context, string) ([]*domain.MediaItem, error) {
	return f.episodes, nil
}

func TestFetchLibrariesSavesToStore(t *testing.T) {
	st, _ := store.NewLibraryStore("", "")
	client := &fakeLibraryClient{libs: []domain.Library{{ID: "lib", Name: "Movies", Type: "movie"}}}
	svc := NewService(client, st, slog.Default())

	libs, err := svc.FetchLibraries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(libs) != 1 || client.libraryCalls != 1 {
		t.Fatalf("libs=%#v calls=%d", libs, client.libraryCalls)
	}
	cached, ok := st.GetLibraries()
	if !ok || len(cached) != 1 || cached[0].ID != "lib" {
		t.Fatalf("cached libraries = %#v, %v", cached, ok)
	}
}

func TestSyncLibraryUsesFreshCache(t *testing.T) {
	st, _ := store.NewLibraryStore("", "")
	if err := st.SaveMovies("lib", []*domain.MediaItem{{ID: "cached"}}, 100); err != nil {
		t.Fatal(err)
	}
	client := &fakeLibraryClient{}
	svc := NewService(client, st, slog.Default())

	result, err := svc.SyncLibrary(context.Background(), domain.Library{ID: "lib", Type: "movie", UpdatedAt: 50}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.FromCache || result.Count != 1 || client.movieCalls != 0 {
		t.Fatalf("result=%#v calls=%d", result, client.movieCalls)
	}
}

func TestFetchMoviesPaginatesAndReportsProgress(t *testing.T) {
	st, _ := store.NewLibraryStore("", "")
	client := &fakeLibraryClient{moviePages: [][]*domain.MediaItem{
		{{ID: "1"}},
		{{ID: "2"}},
		{{ID: "3"}},
	}}
	svc := NewService(client, st, slog.Default())
	var progress []int

	movies, err := svc.FetchMovies(context.Background(), "lib", func(loaded, total int) {
		progress = append(progress, loaded)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 3 || client.movieCalls != 3 {
		t.Fatalf("movies=%#v calls=%d", movies, client.movieCalls)
	}
	if len(progress) != 3 || progress[0] != 1 || progress[2] != 3 {
		t.Fatalf("progress=%v", progress)
	}
}

func flattenMovies(pages [][]*domain.MediaItem) []*domain.MediaItem {
	var out []*domain.MediaItem
	for _, page := range pages {
		out = append(out, page...)
	}
	return out
}

func flattenShows(pages [][]*domain.Show) []*domain.Show {
	var out []*domain.Show
	for _, page := range pages {
		out = append(out, page...)
	}
	return out
}

func flattenMixed(pages [][]domain.ListItem) []domain.ListItem {
	var out []domain.ListItem
	for _, page := range pages {
		out = append(out, page...)
	}
	return out
}
