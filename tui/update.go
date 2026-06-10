package tui

import (
	"strings"

	"solo-cli/taxes"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	// Demo mode: data already loaded, just tick the spinner for consistency
	if m.demoMode {
		return m.spinner.Tick
	}
	return tea.Batch(
		m.spinner.Tick,
		m.fetchSummary,
		m.fetchCompany,
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
			m.activeTab = (m.activeTab + 1) % tabCount
			m.cursor = 0
			m.viewportOffset = 0
			m.taxesScroll = 0
		case "shift+tab", "left", "h":
			m.activeTab = (m.activeTab - 1 + tabCount) % tabCount
			m.cursor = 0
			m.viewportOffset = 0
			m.taxesScroll = 0
		case "d", "delete", "backspace":
			if m.activeTab == TabQueue {
				m.loading = true
				return m, m.deleteSelectedExpense()
			}
		case "up", "k":
			if m.activeTab == TabTaxes {
				if m.taxesScroll > 0 {
					m.taxesScroll--
				}
			} else if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.viewportOffset {
					m.viewportOffset = m.cursor
				}
			}
		case "down", "j":
			if m.activeTab == TabTaxes {
				availHeight := m.height - 7
				if availHeight < 5 {
					availHeight = 5
				}
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
					size := m.tabViewportSize()
					if m.cursor >= m.viewportOffset+size {
						m.viewportOffset = m.cursor - size + 1
					}
				}
			}
		case "r":
			// Refresh
			m.loading = true
			return m, tea.Batch(
				m.fetchSummary,
				m.fetchCompany,
				m.fetchRevenues,
				m.fetchExpenses,
				m.fetchRejected,
				m.fetchQueue,
				m.fetchEFactura,
			)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Fit the list viewport to the terminal height. Chrome around the
		// list: title block (3), tabs (2), showing line (2), table header
		// with border (2), help footer (4)
		m.viewportSize = m.height - 13
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
		if m.summary != nil && m.taxConfig != nil {
			m.taxBreakdown = taxes.Calculate(m.summary.TotalRevenues, m.summary.TotalDeductibleExpenses, m.taxConfig)
			m.taxesLines = len(strings.Split(m.renderTaxes(), "\n"))
		}
		m.checkLoadingDone()

	case companyMsg:
		m.company = msg
		// Company is optional, don't block loading

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

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
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
