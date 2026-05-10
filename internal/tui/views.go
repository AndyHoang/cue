package tui

import (
	"fmt"
	"strings"

	"github.com/SuperCoolPencil/cue/internal/tui/components"
	"github.com/SuperCoolPencil/cue/internal/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderSpinner renders a loading spinner
func RenderSpinner(frame int) string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return styles.SpinnerStyle.Render(frames[frame%len(frames)])
}

// View renders the application
func (m Model) View() string {
	if !m.Ready {
		return "Loading..."
	}

	// Handle modal states
	if m.State == StateHelp {
		return m.renderHelp()
	}

	if m.State == StateConfirmLogout {
		return m.renderLogoutConfirmation()
	}
	if m.State == StateConfirmResume {
		return m.renderResumeConfirmation()
	}

	contentHeight := m.Height - ChromeHeight
	stackLen := m.ColumnStack.Len()
	layout := m.calculateColumnLayout(m.Width)

	var content string

	if stackLen == 0 {
		content = ""
	} else {
		topIdx := stackLen - 1

		// List-height for content columns: 33% of available height
		listHeight := contentHeight / 3
		if listHeight < 4 {
			listHeight = 4
		}
		infoHeight := contentHeight - listHeight

		var columnViews []string

		switch stackLen {
		case 1:
			// Root: library list, full height — no info pane
			libCol := m.ColumnStack.Get(0)
			libCol.SetSize(layout.activeWidth, contentHeight)
			columnViews = append(columnViews, libCol.View())

		case 2:
			// Tab 1: library list (full height)
			// Tab 2: content column split 33/66
			libCol := m.ColumnStack.Get(0)
			libCol.SetSize(layout.parentWidth, contentHeight)
			columnViews = append(columnViews, libCol.View())

			contentCol := m.ColumnStack.Get(1)
			columnViews = append(columnViews, m.renderSplitColumn(contentCol, layout.activeWidth, listHeight, infoHeight))

		default:
			// Tab 1: library list (full height)
			// Tab 2: shows/movies column split 33/66
			// Tab 3: episodes/season-episodes column split 33/66
			libCol := m.ColumnStack.Get(topIdx - 2)
			if layout.grandparentWidth > 0 {
				libCol.SetSize(layout.grandparentWidth, contentHeight)
			} else {
				libCol.SetSize(layout.parentWidth, contentHeight)
			}
			columnViews = append(columnViews, libCol.View())

			if layout.parentWidth > 0 && layout.grandparentWidth == 0 {
				// Only show parent (shows list) when grandparent is not shown
				parentCol := m.ColumnStack.Get(topIdx - 1)
				columnViews = append(columnViews, m.renderSplitColumn(parentCol, layout.parentWidth, listHeight, infoHeight))
			} else if layout.grandparentWidth > 0 {
				parentCol := m.ColumnStack.Get(topIdx - 1)
				parentCol.SetSize(layout.parentWidth, contentHeight)
				columnViews = append(columnViews, parentCol.View())
			}

			activeCol := m.ColumnStack.Get(topIdx)
			columnViews = append(columnViews, m.renderSplitColumn(activeCol, layout.activeWidth, listHeight, infoHeight))
		}

		content = lipgloss.JoinHorizontal(lipgloss.Top, columnViews...)
	}

	// Footer
	footer := m.renderFooter()

	// Combine all
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		footer,
	)

	// Overlay omnibar if visible
	if m.GlobalSearch.IsVisible() {
		view = lipgloss.Place(m.Width, m.Height,
			lipgloss.Center, lipgloss.Center,
			m.GlobalSearch.View())
	}

	// Overlay sort modal if visible
	if m.SortModal.IsVisible() {
		view = lipgloss.Place(m.Width, m.Height,
			lipgloss.Center, lipgloss.Center,
			m.SortModal.View())
	}

	// Overlay playlist modal if visible
	if m.PlaylistModal.IsVisible() {
		view = lipgloss.Place(m.Width, m.Height,
			lipgloss.Center, lipgloss.Center,
			m.PlaylistModal.View())
	}

	// Overlay input modal if visible
	if m.InputModal.IsVisible() {
		view = lipgloss.Place(m.Width, m.Height,
			lipgloss.Center, lipgloss.Center,
			m.InputModal.View())
	}

	return view
}

// renderSplitColumn renders a content column as a 33/66 vertical split:
// top = list (listHeight), bottom = info pane for selected item (infoHeight).
func (m Model) renderSplitColumn(col *components.ListColumn, colWidth, listHeight, infoHeight int) string {
	col.SetSize(colWidth, listHeight)
	listView := col.View()

	// Info pane: fresh inspector for this column's selected item
	insp := components.NewInspector()
	insp.SetSize(colWidth, infoHeight)
	insp.SetItem(col.SelectedItem())
	infoView := insp.View()

	return lipgloss.JoinVertical(lipgloss.Left, listView, infoView)
}

// renderFooter renders a single-line minimal footer
func (m Model) renderFooter() string {
	// Left side: now-playing takes priority, then loading/status
	var left string
	if m.isPlayingTitle != "" {
		// Pulsing indicator via spinner frame
		frames := []string{"▶", "▷"}
		icon := styles.AccentStyle.Render(frames[m.SpinnerFrame/5%len(frames)])
		title := styles.Truncate(m.isPlayingTitle, 40)
		left = icon + " " + styles.DimStyle.Render("Now Playing: "+title)
	} else if m.Loading {
		statusText := "Loading..."

		if m.MultiLibSync {
			// Multi-library: stable library completion fraction
			syncingCount := 0
			for _, state := range m.LibraryStates {
				if state.Status == components.StatusSyncing {
					syncingCount++
				}
			}
			done := len(m.LibraryStates) - syncingCount
			statusText = fmt.Sprintf("Syncing %d/%d libraries...", done, len(m.LibraryStates))
		} else {
			// Single library: show name + item progress
			for id, state := range m.LibraryStates {
				if state.Status == components.StatusSyncing {
					libName := ""
					for _, lib := range m.Libraries {
						if lib.ID == id {
							libName = lib.Name
							break
						}
					}
					if state.Total > 0 {
						statusText = fmt.Sprintf("Syncing %s · %d/%d", libName, state.Loaded, state.Total)
					} else if libName != "" {
						statusText = fmt.Sprintf("Syncing %s...", libName)
					}
					break
				}
			}
		}

		left = RenderSpinner(m.SpinnerFrame) + " " + styles.DimStyle.Render(statusText)
	} else if m.StatusMsg != "" {
		if m.StatusIsErr {
			left = styles.ErrorStyle.Render(m.StatusMsg)
		} else {
			left = styles.DimStyle.Render(m.StatusMsg)
		}
	}

	// Center section: context-specific hints based on column type
	var center string
	if top := m.ColumnStack.Top(); top != nil {
		switch top.ColumnType() {
		case components.ColumnTypePlaylists:
			center = styles.AccentStyle.Render("x") + styles.DimStyle.Render(" Delete")
		case components.ColumnTypePlaylistItems:
			center = styles.AccentStyle.Render("x") + styles.DimStyle.Render(" Remove")
		}
	}

	// Right side: "? help" hint
	right := styles.AccentStyle.Render("?") + styles.DimStyle.Render(" help")

	// Layout: left + centered hints + right
	leftWidth := lipgloss.Width(left)
	centerWidth := lipgloss.Width(center)
	rightWidth := lipgloss.Width(right)

	totalContent := leftWidth + centerWidth + rightWidth
	if totalContent >= m.Width {
		// Not enough space - just left + right
		gap := max(0, m.Width-leftWidth-rightWidth)
		return left + strings.Repeat(" ", gap) + right
	}

	// Center the hints in available space
	available := m.Width - leftWidth - rightWidth
	leftPad := (available - centerWidth) / 2
	rightPad := available - centerWidth - leftPad

	return left + strings.Repeat(" ", leftPad) + center + strings.Repeat(" ", rightPad) + right
}

// helpEntry is a key-description pair for the help screen
type helpEntry struct {
	key  string
	desc string
}

// renderHelp renders the help screen
func (m Model) renderHelp() string {
	nav := []helpEntry{
		{"j / k", "Up / Down"},
		{"h / l", "Back / Drill in"},
		{"g / Home", "First item"},
		{"G / End", "Last item"},
		{"PgUp/Dn", "Scroll page"},
		{"^u / ^d", "Scroll half page"},
	}

	playback := []helpEntry{
		{"Enter", "Play / Resume"},
		{"p", "Play from start"},
		{"w", "Mark watched"},
		{"u", "Mark unwatched"},
		{"N", "Next episode"},
	}

	searchView := []helpEntry{
		{"/", "Filter"},
		{"f", "Global search"},
		{"s", "Sort"},
		{"i", "Toggle inspector"},
		{"Space", "Manage playlists"},
		{"a", "Add/remove queue"},
	}

	other := []helpEntry{
		{"r", "Refresh library"},
		{"R", "Refresh all"},
		{"q", "Quit"},
		{"L", "Logout"},
		{"?", "This help"},
		{"Esc", "Close / Cancel"},
	}

	keyW := 12
	descW := 18
	gap := 4
	colW := keyW + descW
	totalW := colW*2 + gap

	bg := lipgloss.NewStyle().Background(styles.SlateDark)
	keyStyle := bg.Foreground(styles.PlexOrange).Width(keyW)
	descStyle := bg.Foreground(styles.LightGray).Width(descW)
	headerStyle := bg.Foreground(styles.PlexOrange).Bold(true).Width(colW)
	gapStyle := bg.Width(gap)
	fullRowStyle := bg.Width(totalW)

	// Build rows: pair left and right sections side by side
	sections := []struct {
		leftTitle  string
		left       []helpEntry
		rightTitle string
		right      []helpEntry
	}{
		{"NAVIGATION", nav, "PLAYBACK", playback},
		{"SEARCH & VIEW", searchView, "OTHER", other},
	}

	var rows []string
	for i, sec := range sections {
		if i > 0 {
			// Blank separator row between section pairs
			rows = append(rows, fullRowStyle.Render(""))
		}

		// Headers
		rows = append(rows, headerStyle.Render(sec.leftTitle)+gapStyle.Render("")+headerStyle.Render(sec.rightTitle))

		// Entry rows — pad whichever side is shorter
		maxLen := len(sec.left)
		if len(sec.right) > maxLen {
			maxLen = len(sec.right)
		}
		for j := 0; j < maxLen; j++ {
			var leftPart, rightPart string
			if j < len(sec.left) {
				leftPart = keyStyle.Render(sec.left[j].key) + descStyle.Render(sec.left[j].desc)
			} else {
				leftPart = bg.Width(colW).Render("")
			}
			if j < len(sec.right) {
				rightPart = keyStyle.Render(sec.right[j].key) + descStyle.Render(sec.right[j].desc)
			} else {
				rightPart = bg.Width(colW).Render("")
			}
			rows = append(rows, leftPart+gapStyle.Render("")+rightPart)
		}
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(styles.DimGray).
		Italic(true).
		Width(totalW).
		Align(lipgloss.Center).
		Background(styles.SlateDark).
		MarginTop(1)

	rows = append(rows, footerStyle.Render("Press any key to return"))

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.PlexOrange).
		Background(styles.SlateDark).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(m.Width, m.Height,
		lipgloss.Center, lipgloss.Center,
		modal)
}

// renderConfirmDialog renders a centered confirmation modal with styled buttons
func renderConfirmDialog(width, height int, title, body, yesLabel, noLabel, cancelLabel string) string {
	modalWidth := 54

	bg := lipgloss.NewStyle().Background(styles.SlateDark)

	titleStyle := bg.
		Foreground(styles.White).
		Bold(true).
		Width(modalWidth).
		Align(lipgloss.Center)

	bodyStyle := bg.
		Foreground(styles.LightGray).
		Width(modalWidth).
		Align(lipgloss.Center).
		MarginTop(1)

	yesBtn := lipgloss.NewStyle().
		Foreground(styles.White).
		Background(styles.PlexOrange).
		Padding(0, 2).
		Bold(true).
		Render(yesLabel)

	btnGap := bg.Render("  ")

	noBtn := lipgloss.NewStyle().
		Foreground(styles.LightGray).
		Background(styles.SlateLight).
		Padding(0, 2).
		Render(noLabel)

	buttonList := []string{yesBtn, btnGap, noBtn}

	if cancelLabel != "" {
		cancelBtn := lipgloss.NewStyle().
			Foreground(styles.DimGray).
			Background(styles.SlateLight).
			Padding(0, 2).
			Render(cancelLabel)
		buttonList = append(buttonList, btnGap, cancelBtn)
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, buttonList...)

	buttonRow := bg.
		Width(modalWidth).
		Align(lipgloss.Center).
		MarginTop(1).
		Render(buttons)

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		bodyStyle.Render(body),
		buttonRow,
	)

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.PlexOrange).
		Background(styles.SlateDark).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		modal)
}

func (m Model) renderResumeConfirmation() string {
	title := "Resume Playback?"
	body := "Selected item"
	if m.pendingPlayback != nil {
		body = styles.Truncate(m.pendingPlayback.Title, 38)
	}

	return renderConfirmDialog(m.Width, m.Height,
		title, body,
		"Y  Resume", "N  Start Over", "Esc  Cancel")
}

// renderLogoutConfirmation renders the logout confirmation modal
func (m Model) renderLogoutConfirmation() string {
	return renderConfirmDialog(m.Width, m.Height,
		"Log Out?",
		"This will clear your credentials,\nserver URL, and all cached data.",
		"Y  Yes", "N  No", "")
}
