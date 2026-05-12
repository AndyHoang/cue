package components

import (
	"fmt"
	"time"

	"github.com/SuperCoolPencil/cue/internal/domain"
)

// SeasonHeader is a collapsible section header inside a ColumnTypeSeasonEpisodes column.
// It represents one season and can be expanded to reveal its episodes.
type SeasonHeader struct {
	Season   *domain.Season
	Expanded bool
	Loading  bool
}

// SeasonGroup pairs a SeasonHeader with its lazily-loaded episodes.
type SeasonGroup struct {
	Header   *SeasonHeader
	Episodes []*domain.MediaItem
	Loaded   bool
}

// ──────────────────────────────────────────────────────────────────────────────
// domain.ListItem implementation
// ──────────────────────────────────────────────────────────────────────────────

func (h *SeasonHeader) GetID() string {
	return "season_header_" + h.Season.ID
}

func (h *SeasonHeader) GetTitle() string {
	if h.Season.SeasonNum == 0 {
		return "Specials"
	}
	return fmt.Sprintf("S%02d", h.Season.SeasonNum)
}

func (h *SeasonHeader) GetSortTitle() string {
	return fmt.Sprintf("%03d", h.Season.SeasonNum)
}

func (h *SeasonHeader) GetDuration() time.Duration { return 0 }
func (h *SeasonHeader) GetRating() float64         { return 0 }
func (h *SeasonHeader) GetYear() int               { return 0 }
func (h *SeasonHeader) GetAddedAt() int64          { return 0 }
func (h *SeasonHeader) GetUpdatedAt() int64        { return 0 }
func (h *SeasonHeader) GetItemType() string        { return "season_header" }
func (h *SeasonHeader) CanDrillDown() bool         { return false }

func (h *SeasonHeader) GetWatchStatus() domain.WatchStatus {
	return h.Season.WatchStatus()
}

func (h *SeasonHeader) GetDescription() string {
	watched := h.Season.EpisodeCount - h.Season.UnwatchedCount
	return fmt.Sprintf("%d/%d watched", watched, h.Season.EpisodeCount)
}
