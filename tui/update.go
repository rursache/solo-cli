package tui

import (
	"strings"
	"time"

	"solo-cli/taxes"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type marqueeTickMsg struct{}

// marqueeTick drives the focused-row text scrolling
func marqueeTick() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg { return marqueeTickMsg{} })
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	// Demo mode: data already loaded, just tick the spinner for consistency
	if m.demoMode {
		return tea.Batch(m.spinner.Tick, marqueeTick())
	}
	return tea.Batch(m.spinner.Tick, marqueeTick(), m.fetchAll())
}

// fetchAll loads every tab's data concurrently
func (m Model) fetchAll() tea.Cmd {
	return tea.Batch(
		m.fetchSummary,
		m.fetchCompany,
		m.fetchCAEN,
		m.fetchRevenues,
		m.fetchExpenses,
		m.fetchRejected,
		m.fetchQueue,
		m.fetchEFactura,
	)
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right", "l":
			m.setTab((m.activeTab + 1) % tabCount)
		case "shift+tab", "left", "h":
			m.setTab((m.activeTab - 1 + tabCount) % tabCount)
		case "d", "delete", "backspace":
			if m.activeTab == TabQueue {
				m.loading = true
				return m, m.deleteSelectedExpense()
			}
		case "up", "k":
			m.scrollUp()
		case "down", "j":
			m.scrollDown()
		case "[":
			if m.canSwitchYear() && m.year > 2015 {
				m.year--
				m.taxesScroll = 0
				return m, m.fetchSummary
			}
		case "]":
			if m.canSwitchYear() && m.year < m.maxYear {
				m.year++
				m.taxesScroll = 0
				return m, m.fetchSummary
			}
		case "r":
			// Refresh
			m.loading = true
			return m, m.fetchAll()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Fit the list viewport to the body: showing line (2) and table
		// header with border (2) are the list's own chrome
		m.viewportSize = m.bodyHeight() - 4
		if m.viewportSize < 3 {
			m.viewportSize = 3
		}
		// Keep the cursor visible after a resize
		if m.cursor >= m.viewportOffset+m.viewportSize {
			m.viewportOffset = m.cursor - m.viewportSize + 1
		} else if m.viewportOffset > 0 {
			// Pull the list up if the larger viewport leaves dead space
			maxOffset := m.getMaxCursor() - m.viewportSize
			if maxOffset < 0 {
				maxOffset = 0
			}
			if m.viewportOffset > maxOffset {
				m.viewportOffset = maxOffset
			}
		}
		if m.taxBreakdown != nil && m.taxesLines == 0 {
			m.taxesLines = len(strings.Split(m.renderTaxes(), "\n"))
		}

	case summaryMsg:
		m.summary = msg
		if m.summary != nil {
			// The first summary establishes the current fiscal year, the
			// upper bound for [ and ] year switching
			if m.maxYear == 0 {
				m.maxYear = m.summary.Year
			}
			m.year = m.summary.Year
			if m.taxConfig != nil {
				m.taxBreakdown = taxes.Calculate(m.summary.TotalRevenues, m.summary.TotalDeductibleExpenses, m.taxConfig)
				m.taxesLines = len(strings.Split(m.renderTaxes(), "\n"))
			}
		}
		m.checkLoadingDone()

	case companyMsg:
		m.company = msg
		// Company is optional, don't block loading

	case caenMsg:
		m.caenCodes = msg
		// CAEN codes are optional, don't block loading

	case revenuesMsg:
		m.revenues = msg
		m.checkLoadingDone()

	case expensesMsg:
		m.expenses = msg
		m.checkLoadingDone()

	case rejectedMsg:
		m.rejected = msg
		m.checkLoadingDone()

	case queueMsg:
		m.queue = msg
		m.checkLoadingDone()

	case efacturaMsg:
		m.efactura = msg
		m.checkLoadingDone()

	case errMsg:
		m.err = msg
		m.loading = false

	case deleteSuccessMsg:
		m.loading = true
		// Refresh queue after deletion
		return m, m.fetchQueue

	case tea.MouseMsg:
		switch {
		case msg.Button == tea.MouseButtonWheelUp:
			m.scrollUp()
		case msg.Button == tea.MouseButtonWheelDown:
			m.scrollDown()
		case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
			m.handleClick(msg.X, msg.Y)
		}

	case marqueeTickMsg:
		m.marqueeOffset++
		return m, marqueeTick()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// canSwitchYear limits [ and ] to the year-scoped tabs. Demo mode has no
// API to refetch from and the bound is unknown until the first summary
func (m Model) canSwitchYear() bool {
	return (m.activeTab == TabDashboard || m.activeTab == TabTaxes) && !m.demoMode && m.year > 0
}

// setTab switches the active tab and resets per-tab navigation state
func (m *Model) setTab(t Tab) {
	m.activeTab = t
	m.cursor = 0
	m.marqueeOffset = 0
	m.viewportOffset = 0
	m.taxesScroll = 0
}

func (m *Model) scrollUp() {
	if m.activeTab == TabTaxes {
		if m.taxesScroll > 0 {
			m.taxesScroll--
		}
	} else if m.cursor > 0 {
		m.cursor--
		m.marqueeOffset = 0
		if m.cursor < m.viewportOffset {
			m.viewportOffset = m.cursor
		}
	}
}

func (m *Model) scrollDown() {
	if m.activeTab == TabTaxes {
		// Must match the viewport math in renderTaxesViewport
		availHeight := m.bodyHeight() - 1
		maxScroll := m.taxesLines - availHeight
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.taxesScroll < maxScroll {
			m.taxesScroll++
		}
	} else {
		maxCursor := m.getMaxCursor()
		if m.cursor < maxCursor-1 {
			m.cursor++
			m.marqueeOffset = 0
			size := m.tabViewportSize()
			if m.cursor >= m.viewportOffset+size {
				m.viewportOffset = m.cursor - size + 1
			}
		}
	}
}

func (m *Model) checkLoadingDone() {
	if m.summary != nil && m.revenues != nil && m.expenses != nil && m.rejected != nil && m.queue != nil && m.efactura != nil {
		m.loading = false
	}
}

// getMaxCursor returns the number of items in the current tab's list
func (m Model) getMaxCursor() int {
	switch m.activeTab {
	case TabRevenues:
		if m.revenues != nil {
			return len(m.revenues.Items)
		}
	case TabExpenses:
		if m.expenses != nil {
			return len(m.expenses.Items)
		}
	case TabEFactura:
		if m.efactura != nil {
			return len(m.efactura.Items)
		}
	case TabQueue:
		if m.queue != nil {
			return len(m.queue.Items)
		}
	}
	return 0
}
