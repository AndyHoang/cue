package library

import (
	"sort"

	"github.com/SuperCoolPencil/cue/internal/domain"
)

const defaultLocalViewLimit = 200

// ContinueWatching returns cached playable items that have progress.
func (s *Service) ContinueWatching(limit int) []*domain.MediaItem {
	items := s.cachedPlayableItems()
	filtered := make([]*domain.MediaItem, 0, len(items))
	for _, item := range items {
		if item.ViewOffset > 0 && item.WatchStatus() != domain.WatchStatusWatched {
			filtered = append(filtered, item)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].UpdatedAt > filtered[j].UpdatedAt
	})
	return limitMediaItems(filtered, limit)
}

// RecentlyAdded returns cached items (movies and shows) ordered by AddedAt descending.
func (s *Service) RecentlyAdded(limit int) []domain.ListItem {
	libs, ok := s.store.GetLibraries()
	if !ok {
		return nil
	}

	var all []domain.ListItem
	for _, lib := range libs {
		switch lib.Type {
		case "movie":
			if movies, ok := s.store.GetMovies(lib.ID); ok {
				for _, m := range movies {
					all = append(all, m)
				}
			}
		case "show":
			if shows, ok := s.store.GetShows(lib.ID); ok {
				for _, sh := range shows {
					all = append(all, sh)
				}
			}
		case "mixed":
			if items, ok := s.store.GetMixedContent(lib.ID); ok {
				all = append(all, items...)
			}
		}
	}

	sort.SliceStable(all, func(i, j int) bool {
		return all[i].GetAddedAt() > all[j].GetAddedAt()
	})

	if limit <= 0 {
		limit = defaultLocalViewLimit
	}
	if len(all) <= limit {
		return all
	}
	return all[:limit]
}

// SmartFiltered returns cached playable items matching a named local filter.
func (s *Service) SmartFiltered(filter string, limit int) []*domain.MediaItem {
	items := s.cachedPlayableItems()
	filtered := make([]*domain.MediaItem, 0, len(items))
	for _, item := range items {
		switch filter {
		case "unwatched":
			if !item.IsPlayed && item.ViewOffset == 0 {
				filtered = append(filtered, item)
			}
		case "in_progress":
			if item.ViewOffset > 0 && item.WatchStatus() != domain.WatchStatusWatched {
				filtered = append(filtered, item)
			}
		case "4k":
			if item.Height >= 2160 {
				filtered = append(filtered, item)
			}
		case "highly_rated":
			if item.Rating >= 8 {
				filtered = append(filtered, item)
			}
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].GetSortTitle() < filtered[j].GetSortTitle()
	})
	return limitMediaItems(filtered, limit)
}

func (s *Service) cachedPlayableItems() []*domain.MediaItem {
	libs, ok := s.store.GetLibraries()
	if !ok {
		return nil
	}

	var items []*domain.MediaItem
	for _, lib := range libs {
		switch lib.Type {
		case "movie":
			if movies, ok := s.store.GetMovies(lib.ID); ok {
				items = append(items, movies...)
			}
		case "show":
			if eps, ok := s.store.GetAllEpisodes(lib.ID); ok {
				items = append(items, eps...)
			}
		case "mixed":
			if content, ok := s.store.GetMixedContent(lib.ID); ok {
				var libEps []*domain.MediaItem
				var fetched bool
				for _, item := range content {
					switch v := item.(type) {
					case *domain.MediaItem:
						items = append(items, v)
					case *domain.Show:
						if !fetched {
							libEps, _ = s.store.GetAllEpisodes(lib.ID)
							fetched = true
						}
						// Filter episodes for this specific show only
						for _, ep := range libEps {
							if ep.ShowID == v.ID {
								items = append(items, ep)
							}
						}
					}
				}
			}
		}
	}
	return items
}

func limitMediaItems(items []*domain.MediaItem, limit int) []*domain.MediaItem {
	if limit <= 0 {
		limit = defaultLocalViewLimit
	}
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}
