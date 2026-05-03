package playlist

import "github.com/SuperCoolPencil/cue/internal/domain"

// QueueItems returns the local playback queue.
func (s *Service) QueueItems() []*domain.MediaItem {
	items, _ := s.store.GetQueueItems()
	return items
}

// AddToQueue appends an item to the local playback queue if it is not already present.
func (s *Service) AddToQueue(item *domain.MediaItem) error {
	if item == nil {
		return nil
	}
	items, _ := s.store.GetQueueItems()
	for _, existing := range items {
		if existing.ID == item.ID {
			return nil
		}
	}
	items = append(items, item)
	return s.store.SaveQueueItems(items)
}

// RemoveFromQueue removes an item from the local playback queue.
func (s *Service) RemoveFromQueue(itemID string) error {
	items, _ := s.store.GetQueueItems()
	filtered := items[:0]
	for _, item := range items {
		if item.ID != itemID {
			filtered = append(filtered, item)
		}
	}
	return s.store.SaveQueueItems(filtered)
}

// ClearQueue clears the local playback queue.
func (s *Service) ClearQueue() {
	s.store.InvalidateQueue()
}
