package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInputModalSubmitAndEscape(t *testing.T) {
	modal := NewInputModal()
	modal.Show("Name")
	modal, _, _ = modal.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Queue")})
	if modal.Value() != "Queue" {
		t.Fatalf("value = %q", modal.Value())
	}
	_, _, submitted := modal.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !submitted {
		t.Fatal("enter should submit")
	}

	modal.Show("Name")
	modal, _, submitted = modal.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if submitted || modal.IsVisible() {
		t.Fatalf("escape should hide without submit")
	}
}
