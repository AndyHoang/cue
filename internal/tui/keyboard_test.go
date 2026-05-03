package tui

import (
	"testing"
	"time"

	"github.com/SuperCoolPencil/cue/internal/domain"
	"github.com/SuperCoolPencil/cue/internal/tui/components"
	tea "github.com/charmbracelet/bubbletea"
)

func TestEnterOnResumableItemShowsResumeConfirmation(t *testing.T) {
	item := &domain.MediaItem{ID: "m1", Title: "Movie", ViewOffset: 10 * time.Minute}
	col := components.NewListColumn(components.ColumnTypeMovies, "Movies")
	col.SetItems([]*domain.MediaItem{item})
	col.SetFocused(true)

	model := Model{
		State:       StateBrowsing,
		ColumnStack: NewColumnStack(),
	}
	model.ColumnStack.Push(col, 0)

	updated, _ := model.handleEnter()
	got := updated.(Model)
	if got.State != StateConfirmResume {
		t.Fatalf("state = %v, want StateConfirmResume", got.State)
	}
	if got.pendingPlayback == nil || got.pendingPlayback.ID != "m1" {
		t.Fatalf("pending playback = %#v", got.pendingPlayback)
	}
}

func TestResumeConfirmationCancel(t *testing.T) {
	model := Model{
		State:           StateConfirmResume,
		pendingPlayback: &domain.MediaItem{ID: "m1", Title: "Movie"},
	}

	updated, _ := model.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)
	if got.State != StateBrowsing {
		t.Fatalf("state = %v, want StateBrowsing", got.State)
	}
	if got.pendingPlayback != nil {
		t.Fatalf("pending playback should be cleared")
	}
}
