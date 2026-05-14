package player

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/SuperCoolPencil/cue/internal/domain"
)

// PlaybackHandle provides channels for monitoring progress and final result.
type PlaybackHandle struct {
	ResultCh <-chan ScrobbleResult
	StatusCh <-chan string
}

// ScrobbleResult contains the final outcome of a monitored playback session.
type ScrobbleResult struct {
	Item       domain.MediaItem
	ItemID     string
	Title      string
	FinalPosMs int64
	Duration   time.Duration
	AutoMarked bool
	Err        error
}

// Scrobbler monitors a running player process and reports progress to the server.
type Scrobbler struct {
	client   domain.PlaybackClient
	logger   *slog.Logger
	interval time.Duration
}

// NewScrobbler creates a new scrobbler.
func NewScrobbler(client domain.PlaybackClient, logger *slog.Logger) *Scrobbler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Scrobbler{
		client:   client,
		logger:   logger,
		interval: 10 * time.Second,
	}
}

// Monitor starts a background goroutine to track playback progress for one or more items.
// If multiple items are provided, it uses mpv IPC to detect which one is active.
func (s *Scrobbler) Monitor(ctx context.Context, cmd *exec.Cmd, ipcSocket string, playlistStart int, items ...domain.MediaItem) PlaybackHandle {

	resCh := make(chan ScrobbleResult, 1)
	statusCh := make(chan string, 10)

	go func() {
		defer close(resCh)
		defer close(statusCh)
		defer removeMPVSocket(ipcSocket)

		var mpv *mpvConn
		var err error
		var activeItem domain.MediaItem
		var lastPosMs int64
		markedIDs := make(map[string]bool)

		if len(items) > 0 {
			startIdx := playlistStart
			if startIdx < 0 || startIdx >= len(items) {
				startIdx = 0
			}
			activeItem = items[startIdx]
		}

		// Try to connect to MPV IPC if available
		if ipcSocket != "" {
			mpv, err = dialMPV(ipcSocket)
			if err != nil {
				s.logger.Warn("mpv IPC connection failed, falling back to exit-only reporting", "error", err)
			} else {
				defer func() { _ = mpv.Close() }()
			}
		}

		// Polling loop
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		processDone := make(chan error, 1)
		go func() {
			processDone <- cmd.Wait()
		}()

	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case err := <-processDone:
				if err != nil {
					s.logger.Warn("player process exited with error", "error", err)
				}
				break loop
			case <-ticker.C:
				if mpv != nil {
					// Detect if item changed (for playlists)
					if len(items) > 1 {
						if _, err := mpv.GetPath(); err == nil {

							// Find which item matches this path
							// We might need a map of URL -> Item, but URLs are resolved lazily.
							// For now, we'll match by Title if path is complex, or we can improve this later.
							// Actually, if we resolved URLs upfront, we can match exactly.
							// Let's assume for now the order in 'items' matches the playlist order.
							if pos, err := mpv.GetProperty("playlist-pos"); err == nil {
								if idx, ok := pos.(float64); ok && int(idx) < len(items) {
									newIdx := int(idx)
									newItem := items[newIdx]
									if newItem.ID != activeItem.ID {
										s.logger.Info("playlist item changed", "from", activeItem.Title, "to", newItem.Title)
										// Mark all previous items in the playlist as watched
										s.markPreviousWatched(items, newIdx, markedIDs)
										activeItem = newItem
									}
								}
							}

						}
					}

					posSecs, err := mpv.GetTimePos()
					if err == nil {
						lastPosMs = int64(posSecs * 1000)
						s.logger.Debug("reporting progress", "item", activeItem.Title, "pos", lastPosMs)

						// Fire and forget progress update
						go func(item domain.MediaItem, pos int64) {
							updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
							defer cancel()
							if err := s.client.UpdateProgress(updateCtx, item.ID, pos); err == nil {
								// Format position as MM:SS for user display
								d := time.Duration(pos) * time.Millisecond
								statusCh <- fmt.Sprintf("Saved %s %02d:%02d to server", item.Title, int(d.Minutes()), int(d.Seconds())%60)
							} else {
								s.logger.Warn("failed to update progress", "error", err)
							}
						}(activeItem, lastPosMs)
					}
				}

			}
		}

		// Final position update on exit
		if mpv != nil {
			posSecs, err := mpv.GetTimePos()
			if err == nil {
				lastPosMs = int64(posSecs * 1000)
				s.logger.Debug("final progress update", "item", activeItem.Title, "pos", lastPosMs)
				updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := s.client.UpdateProgress(updateCtx, activeItem.ID, lastPosMs); err != nil {
					s.logger.Warn("failed to report final progress", "error", err)
				}
				cancel()
			}
		}

		// Handle auto-scrobble on exit (90% threshold)
		autoMarked := false
		if activeItem.Duration > 0 && lastPosMs > 0 {
			progress := float64(lastPosMs) / float64(activeItem.Duration.Milliseconds())
			if progress >= 0.90 {
				s.logger.Info("auto-marking watched", "item", activeItem.Title, "progress", progress)
				markCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := s.client.MarkPlayed(markCtx, activeItem.ID); err == nil {
					autoMarked = true
					markedIDs[activeItem.ID] = true
					// Find current index and mark all previous
					for i, it := range items {
						if it.ID == activeItem.ID {
							s.markPreviousWatched(items, i, markedIDs)
							break
						}
					}
				}

			}
		}

		resCh <- ScrobbleResult{
			Item:       activeItem,
			ItemID:     activeItem.ID,
			Title:      activeItem.Title,
			FinalPosMs: lastPosMs,
			Duration:   activeItem.Duration,
			AutoMarked: autoMarked,
		}
	}()

	return PlaybackHandle{
		ResultCh: resCh,
		StatusCh: statusCh,
	}
}

func (s *Scrobbler) markPreviousWatched(items []domain.MediaItem, currentIdx int, markedIDs map[string]bool) {
	for i := 0; i < currentIdx; i++ {
		item := items[i]
		if item.IsPlayed || markedIDs[item.ID] {
			continue
		}
		markedIDs[item.ID] = true
		s.logger.Info("bulk-marking previous item watched", "item", item.Title)
		go func(it domain.MediaItem) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.client.MarkPlayed(ctx, it.ID); err != nil {
				s.logger.Warn("failed to mark previous item watched", "item", it.Title, "error", err)
			}
		}(item)
	}
}
