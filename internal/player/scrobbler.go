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

// Monitor starts a background goroutine to track playback progress.
func (s *Scrobbler) Monitor(ctx context.Context, cmd *exec.Cmd, ipcSocket string, item domain.MediaItem) PlaybackHandle {
	resCh := make(chan ScrobbleResult, 1)
	statusCh := make(chan string, 10)

	go func() {
		defer close(resCh)
		defer close(statusCh)
		defer removeMPVSocket(ipcSocket)

		var lastPosMs int64
		var mpv *mpvConn
		var err error

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
					posSecs, err := mpv.GetTimePos()
					if err == nil {
						lastPosMs = int64(posSecs * 1000)
						s.logger.Debug("reporting progress", "item", item.Title, "pos", lastPosMs)

						// Fire and forget progress update
						go func(pos int64) {
							updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
							defer cancel()
							if err := s.client.UpdateProgress(updateCtx, item.ID, pos); err == nil {
								// Format position as MM:SS for user display
								d := time.Duration(pos) * time.Millisecond
								statusCh <- fmt.Sprintf("Saved %02d:%02d to server", int(d.Minutes()), int(d.Seconds())%60)
							} else {
								s.logger.Warn("failed to update progress", "error", err)
							}
						}(lastPosMs)
					}
				}
			}
		}

		// Final position update on exit
		if mpv != nil {
			posSecs, err := mpv.GetTimePos()
			if err == nil {
				lastPosMs = int64(posSecs * 1000)
				s.logger.Debug("final progress update", "item", item.Title, "pos", lastPosMs)
				updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := s.client.UpdateProgress(updateCtx, item.ID, lastPosMs); err != nil {
					s.logger.Warn("failed to report final progress", "error", err)
				}
				cancel()
			}
		}

		// Handle auto-scrobble on exit (90% threshold)
		autoMarked := false
		if item.Duration > 0 && lastPosMs > 0 {
			progress := float64(lastPosMs) / float64(item.Duration.Milliseconds())
			if progress >= 0.90 {
				s.logger.Info("auto-marking watched", "item", item.Title, "progress", progress)
				markCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := s.client.MarkPlayed(markCtx, item.ID); err == nil {
					autoMarked = true
				}
			}
		}

		resCh <- ScrobbleResult{
			ItemID:     item.ID,
			Title:      item.Title,
			FinalPosMs: lastPosMs,
			Duration:   item.Duration,
			AutoMarked: autoMarked,
		}
	}()

	return PlaybackHandle{
		ResultCh: resCh,
		StatusCh: statusCh,
	}
}
