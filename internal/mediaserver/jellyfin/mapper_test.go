package jellyfin

import (
	"testing"
	"time"

	"github.com/SuperCoolPencil/cue/internal/domain"
)

func TestMapJellyfinLibraries(t *testing.T) {
	libs := MapLibraries([]Item{
		{ID: "m", Name: "Movies", CollectionType: "movies", DateCreated: "2026-01-02T03:04:05Z"},
		{ID: "s", Name: "Shows", CollectionType: "tvshows"},
		{ID: "x", Name: "Music", CollectionType: "music"},
	})
	if len(libs) != 2 {
		t.Fatalf("libs = %#v", libs)
	}
	if libs[0].Type != "movie" || libs[0].UpdatedAt == 0 || libs[1].Type != "show" {
		t.Fatalf("mapped libs = %#v", libs)
	}
}

func TestMapJellyfinMovieAndMixedContent(t *testing.T) {
	items := []Item{
		{
			ID:              "m1",
			Name:            "Movie",
			SortName:        "Movie Sort",
			Type:            "Movie",
			ParentID:        "lib",
			RunTimeTicks:    int64(90 * time.Second / 100),
			CommunityRating: 8.1,
			OfficialRating:  "Not Rated",
			DateCreated:     "2026-01-02T03:04:05Z",
			UserData:        &UserData{PlaybackPositionTicks: int64(30 * time.Second / 100)},
			ImageTags:       ImageTags{Primary: "tag"},
			Container:       "mkv,mp4",
			MediaSources: []MediaSource{{
				Size: 42,
				MediaStreams: []MediaStream{
					{Type: "Video", Codec: "hevc", Width: 3840, Height: 2160, BitRate: 8000000},
					{Type: "Audio", Codec: "eac3", Channels: 6},
				},
			}},
		},
		{ID: "s1", Name: "Show", Type: "Series", ChildCount: 2, RecursiveItemCount: 10, UserData: &UserData{UnplayedItemCount: 4}},
	}

	movies := MapMovies(items, "http://server")
	if len(movies) != 1 {
		t.Fatalf("movies = %#v", movies)
	}
	movie := movies[0]
	if movie.Duration != 90*time.Second || movie.ViewOffset != 30*time.Second {
		t.Fatalf("movie timing = %#v", movie)
	}
	if movie.ContentRating != "NR" || movie.VideoCodec != "HEVC" || movie.AudioCodec != "EAC3" || movie.Bitrate != 8000 {
		t.Fatalf("movie normalization = %#v", movie)
	}
	if movie.ThumbURL != "http://server/Items/m1/Images/Primary?tag=tag" || movie.FileSize != 42 {
		t.Fatalf("movie media fields = %#v", movie)
	}

	mixed := MapLibraryContent(items, "http://server")
	if len(mixed) != 2 {
		t.Fatalf("mixed = %#v", mixed)
	}
	if _, ok := mixed[0].(*domain.MediaItem); !ok {
		t.Fatalf("first mixed type = %T", mixed[0])
	}
	if show, ok := mixed[1].(*domain.Show); !ok || show.UnwatchedCount != 4 {
		t.Fatalf("show = %#v type=%T", mixed[1], mixed[1])
	}
}

func TestMapJellyfinEpisodeSearchAndPlaylist(t *testing.T) {
	episode := MapEpisodes([]Item{{
		ID:                "e1",
		Name:              "Episode",
		Type:              "Episode",
		SeriesID:          "show",
		SeriesName:        "Show",
		SeasonID:          "season",
		ParentIndexNumber: 1,
		IndexNumber:       2,
	}}, "")[0]
	if episode.Type != domain.MediaTypeEpisode || episode.EpisodeCode() != "S01E02" || episode.ParentID != "season" {
		t.Fatalf("episode = %#v", episode)
	}

	results := MapSearchResults([]SearchHint{
		{ID: "m1", Name: "Movie", Type: "Movie"},
		{ID: "bad", Name: "Song", Type: "Audio"},
	}, "")
	if len(results) != 1 || results[0].ID != "m1" {
		t.Fatalf("results = %#v", results)
	}

	playlists := MapPlaylists([]Item{{ID: "p1", Name: "Playlist", Type: "Playlist", ChildCount: 3, RunTimeTicks: int64(time.Minute / 100)}}, "")
	if len(playlists) != 1 || playlists[0].ItemCount != 3 || playlists[0].Duration != time.Minute {
		t.Fatalf("playlists = %#v", playlists)
	}
}
