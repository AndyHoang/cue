package tui

import (
	"testing"

	"github.com/SuperCoolPencil/cue/internal/tui/components"
)

func TestColumnStackPushPopAndReset(t *testing.T) {
	stack := NewColumnStack()
	root := components.NewListColumn(components.ColumnTypeLibraries, "Libraries")
	stack.Reset(root)
	if stack.Len() != 1 || stack.Top() != root || !root.IsFocused() {
		t.Fatalf("root stack not initialized")
	}

	child := components.NewListColumn(components.ColumnTypeMovies, "Movies")
	stack.Push(child, 3)
	if stack.Len() != 2 || stack.Top() != child || child.IsFocused() != true || root.IsFocused() {
		t.Fatalf("push focus/length failed")
	}
	if !stack.CanGoBack() || stack.Depth() != 1 {
		t.Fatalf("back/depth state failed")
	}

	popped, cursor := stack.Pop()
	if popped != child || cursor != 3 {
		t.Fatalf("pop = %v cursor=%d", popped, cursor)
	}
	if stack.Len() != 1 || stack.Top() != root || !root.IsFocused() {
		t.Fatalf("root focus not restored")
	}
}
