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

	// Title
	b.WriteString(TitleStyle.Render("SOLO.ro CLI"))
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
	helpText := "←/→ tabs • ↑/↓ navigate • r refresh • q quit"
	if m.activeTab == TabQueue {
		helpText = "←/→ tabs • ↑/↓ navigate • d delete • r refresh • q quit"
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

	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("Showing %d-%d of %d", m.viewportOffset+1, min(m.viewportOffset+size, total), total)))
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

	return b.String()
}

// bodyHeight returns the rows available for tab content between the
// title/tab chrome (5 lines) and the pinned help footer (2 lines, after
// one padding row). Every tab derives its viewport from this so resize
// behavior stays identical across tabs
func (m Model) bodyHeight() int {
	h := m.height - 7
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
