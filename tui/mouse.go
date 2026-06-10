package tui

import "github.com/charmbracelet/lipgloss"

const (
	// tabsRowY is the screen row of the tab bar: title line, its margin and
	// one blank line above it
	tabsRowY = 3
	// listRowsStartY is where table rows begin on list tabs: tab bar chrome
	// (5) plus the showing line (2) and the table header with border (2)
	listRowsStartY = 9
)

func (m *Model) handleClick(x, y int) {
	if y == tabsRowY {
		m.clickTab(x)
		return
	}
	m.clickRow(y)
}

// clickTab hit-tests x against the rendered tab cells
func (m *Model) clickTab(x int) {
	pos := 0
	for _, tab := range tabOrder {
		style := InactiveTabStyle
		if tab == m.activeTab {
			style = ActiveTabStyle
		}
		w := lipgloss.Width(style.Render(tab.String()))
		if x >= pos && x < pos+w {
			if tab != m.activeTab {
				m.setTab(tab)
			}
			return
		}
		pos += w
	}
}

// clickRow moves the cursor to the clicked table row on list tabs
func (m *Model) clickRow(y int) {
	switch m.activeTab {
	case TabRevenues, TabExpenses, TabEFactura, TabQueue:
	default:
		return
	}

	start := listRowsStartY
	// The rejected warning block shifts the expenses table down
	if m.activeTab == TabExpenses && m.rejected != nil && len(m.rejected.Items) > 0 {
		start += len(m.rejected.Items) + 2
	}

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
