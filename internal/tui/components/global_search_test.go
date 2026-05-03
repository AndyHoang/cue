package components

import (
	"testing"

	"github.com/SuperCoolPencil/cue/internal/domain"
	"github.com/SuperCoolPencil/cue/internal/search"
	tea "github.com/charmbracelet/bubbletea"
)

func TestGlobalSearchSelectionAndQueryChange(t *testing.T) {
	searchBox := NewGlobalSearch()
	searchBox.Show()
	searchBox, _, _ = searchBox.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("matrix")})
	if searchBox.Query() != "matrix" {
		t.Fatalf("query = %q", searchBox.Query())
	}
	if !searchBox.QueryChanged() || searchBox.QueryChanged() {
		t.Fatalf("query changed tracking failed")
	}
	searchBox.SetResults([]search.FilterResult{{
		FilterItem: search.FilterItem{Item: &domain.MediaItem{ID: "m1"}, Title: "The Matrix", Type: domain.MediaTypeMovie},
	}})
	_, _, selected := searchBox.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !selected {
		t.Fatal("enter should select result")
	}
	if got := searchBox.Selected(); got == nil || got.Title != "The Matrix" {
		t.Fatalf("selected = %#v", got)
	}
}
