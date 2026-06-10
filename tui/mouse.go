package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	// tabsRowY is the screen row of the tab bar: title line and one blank
	tabsRowY = 2
	// searchBarRowY is the combined search/showing line, the first body row
	searchBarRowY = 4
	// listRowsStartY is where table rows begin on list tabs: tab bar chrome
	// (4), search/showing line with blank (2), header with border (2)
	listRowsStartY = 8
)

// listChromeShift is how far the list layout is pushed down by content
// above it (the rejected warning block on the Expenses tab)
func (m Model) listChromeShift() int {
	if m.activeTab == TabExpenses && m.rejected != nil && len(m.rejected.Items) > 0 {
		return len(m.rejected.Items) + 2
	}
	return 0
}

func (m *Model) handleClick(x, y int) tea.Cmd {
	if cmd := m.clickQuit(x, y); cmd != nil {
		return cmd
	}
	if y == tabsRowY {
		return m.clickTab(x)
	}
	if m.activeTab == TabDashboard {
		return m.clickYear(x, y)
	}
	if m.isListTab() && !m.demoMode && y == searchBarRowY+m.listChromeShift() {
		m.searching = true
		m.searchInput = m.searchQuery
		return nil
	}
	m.clickRow(y)
	return nil
}

// clickYear hit-tests a click against the year row in the dashboard
// summary box. The row is located in the rendered view so the hit zones
// cannot drift from the layout
func (m *Model) clickYear(x, y int) tea.Cmd {
	if m.demoMode || m.maxYear == 0 || m.summary == nil {
		return nil
	}

	lines := strings.Split(m.View(), "\n")
	if y < 0 || y >= len(lines) {
		return nil
	}
	plain := ansi.Strip(lines[y])
	if !strings.Contains(plain, "Year:") {
		return nil
	}

	for yr := m.maxYear; yr > m.maxYear-yearOptions; yr-- {
		token := fmt.Sprintf("%d", yr)
		idx := strings.Index(plain, token)
		if idx < 0 {
			continue
		}
		col := utf8.RuneCountInString(plain[:idx])
		if x >= col && x < col+len(token) {
			if yr != m.year {
				m.year = yr
				m.taxesScroll = 0
				return m.fetchSummary
			}
			return nil
		}
	}
	return nil
}

// clickQuit hit-tests the quit button on the help row (the last terminal
// line) by locating it in the rendered view, so the zone cannot drift
// from the layout math
func (m *Model) clickQuit(x, y int) tea.Cmd {
	if m.height == 0 || y != m.height-1 {
		return nil
	}
	lines := strings.Split(m.View(), "\n")
	plain := ansi.Strip(lines[len(lines)-1])
	idx := strings.Index(plain, quitLabel)
	if idx < 0 {
		return nil
	}
	col := utf8.RuneCountInString(plain[:idx])
	// One cell of slack on each side for easier clicking
	if x >= col-1 && x <= col+utf8.RuneCountInString(quitLabel) {
		return tea.Quit
	}
	return nil
}

// clickTab hit-tests x against the rendered tab cells
func (m *Model) clickTab(x int) tea.Cmd {
	pos := 0
	for _, tab := range tabOrder {
		style := InactiveTabStyle
		if tab == m.activeTab {
			style = ActiveTabStyle
		}
		w := lipgloss.Width(style.Render(tab.String()))
		if x >= pos && x < pos+w {
			if tab != m.activeTab {
				return m.setTab(tab)
			}
			return nil
		}
		pos += w
	}
	return nil
}

// clickRow moves the cursor to the clicked table row on list tabs
func (m *Model) clickRow(y int) {
	switch m.activeTab {
	case TabRevenues, TabExpenses, TabEFactura, TabQueue:
	default:
		return
	}

	start := listRowsStartY + m.listChromeShift()

	if y < start {
		return
	}
	idx := m.viewportOffset + (y - start)
	if idx >= m.getMaxCursor() || idx >= m.viewportOffset+m.tabViewportSize() {
		return
	}
	if idx != m.cursor {
		m.cursor = idx
		m.marqueeOffset = 0
	}
}
