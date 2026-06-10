package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model
func (m Model) View() string {
	if m.err != nil {
		return ErrorStyle.Render(fmt.Sprintf("Error: %v\n\nPress q to quit", m.err))
	}

	var b strings.Builder

	// Title row with the quit button right aligned
	title := AppTitleStyle.Render("SOLO.ro CLI")
	quit := SummaryLabelStyle.Render(quitLabel)
	if gap := m.width - lipgloss.Width(title) - lipgloss.Width(quit) - 1; gap > 0 {
		title += strings.Repeat(" ", gap) + quit
	}
	b.WriteString(title)
	b.WriteString("\n\n")

	// Tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Loading state
	if m.loading {
		b.WriteString(m.spinner.View())
		b.WriteString(LoadingStyle.Render(" Loading..."))
		b.WriteString("\n")
	} else {
		// Content based on active tab
		switch m.activeTab {
		case TabDashboard:
			b.WriteString(m.renderDashboard())
		case TabRevenues:
			b.WriteString(m.renderRevenues())
		case TabExpenses:
			b.WriteString(m.renderExpenses())
		case TabQueue:
			b.WriteString(m.renderQueue())
		case TabEFactura:
			b.WriteString(m.renderEFactura())
		case TabTaxes:
			b.WriteString(m.renderTaxesViewport())
		}
	}

	// Help, pinned to the bottom row of the terminal
	helpText := "←/→ tabs • ↑/↓ navigate • / search • r refresh • q quit"
	switch {
	case m.searching:
		helpText = "type to filter live • enter done • esc clear"
	case m.activeTab == TabQueue:
		helpText = "←/→ tabs • ↑/↓ navigate • / search • d delete • r refresh • q quit"
	case m.activeTab == TabDashboard:
		helpText = "←/→ tabs • [ and ] switch year • r refresh • q quit"
	case m.activeTab == TabTaxes:
		helpText = "←/→ tabs • ↑/↓ scroll • [ and ] switch year • r refresh • q quit"
	}
	help := HelpStyle.Render(helpText)

	content := b.String()
	if m.height > 0 {
		padding := m.height - lipgloss.Height(content) - lipgloss.Height(help) + 1
		if padding < 1 {
			padding = 1
		}
		content += strings.Repeat("\n", padding)
	} else {
		content += "\n\n"
	}

	return content + help
}

// tabOrder is the display order of the tab bar, shared with click handling
var tabOrder = []Tab{TabDashboard, TabRevenues, TabExpenses, TabEFactura, TabQueue, TabTaxes}

func (m Model) renderTabs() string {
	var parts []string

	for _, tab := range tabOrder {
		if tab == m.activeTab {
			parts = append(parts, ActiveTabStyle.Render(tab.String()))
		} else {
			parts = append(parts, InactiveTabStyle.Render(tab.String()))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// Helper, rune-safe so diacritics are not split mid-character
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-3]) + "..."
}

// padTruncate fits s to exactly width runes, truncating or padding with
// spaces. Used for fill columns so the row (and selection highlight)
// spans the full terminal width
func padTruncate(s string, width int) string {
	r := []rune(s)
	if len(r) > width {
		return truncate(s, width)
	}
	return s + strings.Repeat(" ", width-len(r))
}

// marqueeHoldTicks keeps the marquee still briefly before it starts sliding
const marqueeHoldTicks = 4

// marquee returns a width-sized window into s that slides with offset,
// wrapping around with a gap. Strings that fit are just padded
func marquee(s string, width, offset int) string {
	r := []rune(s)
	if len(r) <= width {
		return padTruncate(s, width)
	}
	offset -= marqueeHoldTicks
	if offset < 0 {
		offset = 0
	}
	r = append(r, []rune("   ")...) // gap between wrap-arounds
	start := offset % len(r)
	out := make([]rune, 0, width)
	for i := 0; i < width; i++ {
		out = append(out, r[(start+i)%len(r)])
	}
	return string(out)
}

// cell renders a table cell: the focused row scrolls overflowing text in
// place, other rows show the static truncated version
func (m Model) cell(s string, width int, focused bool) string {
	if focused {
		return marquee(s, width, m.marqueeOffset)
	}
	return padTruncate(s, width)
}

// tabViewportSize returns the visible row count for the active tab. The
// Expenses tab loses rows to the rejected warning block when present
func (m Model) tabViewportSize() int {
	size := m.viewportSize
	if m.activeTab == TabExpenses && m.rejected != nil && len(m.rejected.Items) > 0 {
		size -= len(m.rejected.Items) + 2
	}
	if size < 1 {
		return 1
	}
	return size
}

// renderList renders the standard list tab layout: scroll position line,
// table header and the visible rows with the cursor row highlighted
func (m Model) renderList(total int, header string, row func(i int) string) string {
	var b strings.Builder
	size := m.tabViewportSize()

	// total is the loaded item count; the server may report more available
	_, available := m.loadedAndTotal()
	showing := fmt.Sprintf("Showing %d-%d of %d", m.viewportOffset+1, min(m.viewportOffset+size, total), available)
	b.WriteString(m.searchAndShowingLine(showing))
	b.WriteString("\n\n")

	b.WriteString(TableHeaderStyle.Render(header))
	b.WriteString("\n")

	endIdx := min(m.viewportOffset+size, total)
	for i := m.viewportOffset; i < endIdx; i++ {
		if i == m.cursor {
			b.WriteString(TableSelectedStyle.Render(row(i)))
		} else {
			b.WriteString(TableRowStyle.Render(row(i)))
		}
		b.WriteString("\n")
	}

	// No trailing newline: with a full viewport it would push the view one
	// line past the terminal height and clip the title off the top
	return strings.TrimSuffix(b.String(), "\n")
}

// renderSearchBar renders the always-visible search bar on list tabs:
// the live input while typing, the applied filter, or a hint
func (m Model) renderSearchBar() string {
	label := SummaryLabelStyle.Render("Search: ")
	switch {
	case m.searching:
		return label + TableRowStyle.Render(m.searchInput+"█")
	case m.searchQuery != "":
		return label + TableRowStyle.Render(m.searchQuery) + SummaryLabelStyle.Render("  (esc to clear)")
	default:
		return label + SummaryLabelStyle.Render("press / or click here to filter")
	}
}

// searchAndShowingLine combines the search bar (left) and the result
// counter (right aligned) on a single line. The counter yields when a
// long search input needs the space
func (m Model) searchAndShowingLine(showing string) string {
	search := m.renderSearchBar()
	gap := m.width - lipgloss.Width(search) - len(showing) - 1
	if gap < 2 {
		return search
	}
	return search + strings.Repeat(" ", gap) + SummaryLabelStyle.Render(showing)
}

// emptyList renders the empty state, keeping the search bar visible so a
// query without matches can still be edited or cleared
func (m Model) emptyList(message string) string {
	return m.renderSearchBar() + "\n\n" + message
}

// quitLabel is the clickable quit button on the title row. The × is
// U+00D7 which is single-width in every terminal font, unlike U+2715
// which renders wide in some fonts and wraps the title row
const quitLabel = "× quit"

// bodyHeight returns the rows available for tab content between the
// title/tab chrome (4 lines) and the pinned help footer (2 lines, after
// one padding row). Every tab derives its viewport from this so resize
// behavior stays identical across tabs
func (m Model) bodyHeight() int {
	h := m.height - 6
	if h < 5 {
		h = 5
	}
	return h
}

// fillWidth returns the space left for a table's fill column given the
// total width used by its fixed columns (including separators)
func (m Model) fillWidth(used, minWidth int) int {
	avail := m.width - used - 1
	if avail < minWidth {
		return minWidth
	}
	return avail
}

// WarningStyle returns styled warning text
func WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
}

// InfoStyle returns styled info text (blue)
func InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))
}
