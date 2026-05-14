package components

import (
	"testing"

	"github.com/SuperCoolPencil/cue/internal/domain"
	tea "github.com/charmbracelet/bubbletea"
)

func TestListColumnSortSelectionAndFilter(t *testing.T) {
	col := NewListColumn(ColumnTypeMovies, "Movies")
	col.SetFocused(true)
	col.SetItems([]*domain.MediaItem{
		{ID: "old", Title: "Old", AddedAt: 10},
		{ID: "new", Title: "New", AddedAt: 30},
		{ID: "middle", Title: "Middle", AddedAt: 20},
	})

	col.ApplySort(SortDateAdded, SortDesc)
	if item := col.SelectedMediaItem(); item == nil || item.ID != "new" {
		t.Fatalf("selected after sort = %#v", item)
	}

	col.ToggleFilter()
	col, _ = col.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("mid")})
	if col.ItemCount() != 1 {
		t.Fatalf("filtered count = %d", col.ItemCount())
	}
	if item := col.SelectedMediaItem(); item == nil || item.ID != "middle" {
		t.Fatalf("selected after filter = %#v", item)
	}
	col.ClearFilter()
	if col.ItemCount() != 3 {
		t.Fatalf("count after clear = %d", col.ItemCount())
	}
}

func TestListColumnSetSelectedByIDHonorsSort(t *testing.T) {
	col := NewListColumn(ColumnTypeMovies, "Movies")
	col.SetItems([]*domain.MediaItem{
		{ID: "b", Title: "B"},
		{ID: "a", Title: "A"},
	})
	col.ApplySort(SortTitle, SortAsc)
	if !col.SetSelectedByID("b") {
		t.Fatal("expected to select existing item")
	}
	if item := col.SelectedMediaItem(); item == nil || item.ID != "b" {
		t.Fatalf("selected = %#v", item)
	}
	if col.SetSelectedByID("missing") {
		t.Fatal("missing id should not select")
	}
}

func TestListColumnHideWatched(t *testing.T) {
	col := NewListColumn(ColumnTypeMovies, "Movies")
	col.SetItems([]*domain.MediaItem{
		{ID: "watched", Title: "Watched", IsPlayed: true},
		{ID: "unwatched", Title: "Unwatched", IsPlayed: false},
	})

	if col.ItemCount() != 2 {
		t.Fatalf("initial count = %d", col.ItemCount())
	}

	col.SetHideWatched(true)
	if col.ItemCount() != 1 {
		t.Fatalf("count after hide watched = %d", col.ItemCount())
	}
	if item := col.SelectedMediaItem(); item == nil || item.ID != "unwatched" {
		t.Fatalf("selected after hide watched = %#v", item)
	}

	col.SetHideWatched(false)
	if col.ItemCount() != 2 {
		t.Fatalf("count after show watched = %d", col.ItemCount())
	}
}
