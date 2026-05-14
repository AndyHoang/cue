package tui

import (
	"testing"

	"github.com/SuperCoolPencil/cue/internal/domain"
	"github.com/SuperCoolPencil/cue/internal/tui/components"
)

func TestModelPropagateWatchStatus(t *testing.T) {
	m := &Model{
		ColumnStack: NewColumnStack(),
	}

	// Setup stack: Shows -> Seasons -> Episodes
	showCol := components.NewListColumn(components.ColumnTypeShows, "Shows")
	show := &domain.Show{ID: "show1", UnwatchedCount: 10, EpisodeCount: 10}
	showCol.SetItems([]*domain.Show{show})
	m.ColumnStack.Push(showCol, 0)

	seasonCol := components.NewListColumn(components.ColumnTypeSeasons, "Seasons")
	season := &domain.Season{ID: "season1", UnwatchedCount: 5, EpisodeCount: 5}
	seasonCol.SetItems([]*domain.Season{season})
	m.ColumnStack.Push(seasonCol, 0)

	episodeCol := components.NewListColumn(components.ColumnTypeEpisodes, "Episodes")
	episode := &domain.MediaItem{
		ID:       "ep1",
		Type:     domain.MediaTypeEpisode,
		ShowID:   "show1",
		ParentID: "season1",
	}
	episodeCol.SetItems([]*domain.MediaItem{episode})
	m.ColumnStack.Push(episodeCol, 0)

	// Test: Mark episode as watched (watched=true, delta=-1)
	m.propagateWatchStatus(episode, true)

	if show.UnwatchedCount != 9 {
		t.Errorf("expected show unwatched 9, got %d", show.UnwatchedCount)
	}
	if season.UnwatchedCount != 4 {
		t.Errorf("expected season unwatched 4, got %d", season.UnwatchedCount)
	}

	// Test: Mark episode as unwatched (watched=false, delta=+1)
	m.propagateWatchStatus(episode, false)

	if show.UnwatchedCount != 10 {
		t.Errorf("expected show unwatched 10, got %d", show.UnwatchedCount)
	}
	if season.UnwatchedCount != 5 {
		t.Errorf("expected season unwatched 5, got %d", season.UnwatchedCount)
	}
}
