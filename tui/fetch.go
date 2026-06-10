package tui

import (
	"solo-cli/client"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) fetchSummary() tea.Msg {
	summary, err := m.client.GetSummaryForYear(m.year)
	if err != nil {
		return errMsg(err)
	}
	return summaryMsg(summary)
}

func (m Model) fetchCompany() tea.Msg {
	if m.client.CompanyID == "" {
		return companyMsg(nil)
	}
	company, err := m.client.GetCompanyInfo(m.client.CompanyID)
	if err != nil {
		// Company info is optional, don't fail
		return companyMsg(nil)
	}
	return companyMsg(company)
}

func (m Model) fetchCAEN() tea.Msg {
	if m.client.CompanyID == "" {
		return caenMsg(nil)
	}
	codes, err := m.client.GetCAENCodes(m.client.CompanyID)
	if err != nil {
		// CAEN codes are optional, don't fail
		return caenMsg(nil)
	}
	return caenMsg(codes)
}

// searchFor returns the active search query when tab is the searched tab.
// The query only ever applies to the tab it was typed on
func (m Model) searchFor(tab Tab) string {
	if m.activeTab == tab {
		return m.searchQuery
	}
	return ""
}

func (m Model) fetchRevenues() tea.Msg {
	revenues, err := m.client.ListRevenues(0, m.pageSize, m.searchFor(TabRevenues))
	if err != nil {
		return errMsg(err)
	}
	return revenuesMsg(revenues)
}

func (m Model) fetchExpenses() tea.Msg {
	expenses, err := m.client.ListExpenses(0, m.pageSize, m.searchFor(TabExpenses))
	if err != nil {
		return errMsg(err)
	}
	return expensesMsg(expenses)
}

func (m Model) fetchRejected() tea.Msg {
	rejected, err := m.client.ListRejectedExpenses(0, m.pageSize)
	if err != nil {
		// Rejected expenses are optional, don't fail if unavailable
		return rejectedMsg(&client.RejectedExpenseResponse{Items: []client.RejectedExpense{}})
	}
	return rejectedMsg(rejected)
}

func (m Model) fetchQueue() tea.Msg {
	queue, err := m.client.ListQueuedExpenses(0, m.pageSize, m.searchFor(TabQueue))
	if err != nil {
		return errMsg(err)
	}
	return queueMsg(queue)
}

func (m Model) fetchEFactura() tea.Msg {
	efactura, err := m.client.ListEFactura(0, m.pageSize, m.searchFor(TabEFactura))
	if err != nil {
		return errMsg(err)
	}
	return efacturaMsg(efactura)
}

// fetchActiveList returns the fetch command for the active list tab
func (m Model) fetchActiveList() tea.Cmd {
	switch m.activeTab {
	case TabRevenues:
		return m.fetchRevenues
	case TabExpenses:
		return m.fetchExpenses
	case TabEFactura:
		return m.fetchEFactura
	case TabQueue:
		return m.fetchQueue
	}
	return nil
}

// loadedAndTotal returns how many items the active tab has loaded and the
// server-reported total (equal to loaded when the API omits TotalResults)
func (m Model) loadedAndTotal() (int, int) {
	var loaded int
	var total *int
	switch m.activeTab {
	case TabRevenues:
		if m.revenues != nil {
			loaded, total = len(m.revenues.Items), m.revenues.TotalResults
		}
	case TabExpenses:
		if m.expenses != nil {
			loaded, total = len(m.expenses.Items), m.expenses.TotalResults
		}
	case TabEFactura:
		if m.efactura != nil {
			loaded, total = len(m.efactura.Items), m.efactura.TotalResults
		}
	case TabQueue:
		if m.queue != nil {
			loaded, total = len(m.queue.Items), m.queue.TotalResults
		}
	}
	if total != nil && *total > loaded {
		return loaded, *total
	}
	return loaded, loaded
}

// fetchRestOfRevenues loads the next revenue page unconditionally. The
// Chart tab needs the complete invoice list to aggregate by month, so it
// chains this until everything is loaded
func (m *Model) fetchRestOfRevenues() tea.Cmd {
	if m.demoMode || m.fetchingMore || m.revenues == nil || m.revenues.TotalResults == nil {
		return nil
	}
	loaded := len(m.revenues.Items)
	if loaded >= *m.revenues.TotalResults {
		return nil
	}
	m.fetchingMore = true

	offset, pageSize, c := loaded, m.pageSize, m.client
	return func() tea.Msg {
		resp, err := c.ListRevenues(offset, pageSize, "")
		if err != nil {
			return errMsg(err)
		}
		return revenuesPageMsg(resp)
	}
}

// maybeFetchMore starts a next-page fetch when the cursor gets within one
// viewport of the end of the loaded items and the server has more
func (m *Model) maybeFetchMore() tea.Cmd {
	if m.fetchingMore || m.demoMode || !m.isListTab() {
		return nil
	}
	loaded, total := m.loadedAndTotal()
	if loaded >= total || m.cursor < loaded-m.tabViewportSize() {
		return nil
	}
	m.fetchingMore = true

	offset, pageSize := loaded, m.pageSize
	c, tab, search := m.client, m.activeTab, m.searchQuery
	switch tab {
	case TabRevenues:
		return func() tea.Msg {
			resp, err := c.ListRevenues(offset, pageSize, search)
			if err != nil {
				return errMsg(err)
			}
			return revenuesPageMsg(resp)
		}
	case TabExpenses:
		return func() tea.Msg {
			resp, err := c.ListExpenses(offset, pageSize, search)
			if err != nil {
				return errMsg(err)
			}
			return expensesPageMsg(resp)
		}
	case TabEFactura:
		return func() tea.Msg {
			resp, err := c.ListEFactura(offset, pageSize, search)
			if err != nil {
				return errMsg(err)
			}
			return efacturaPageMsg(resp)
		}
	case TabQueue:
		return func() tea.Msg {
			resp, err := c.ListQueuedExpenses(offset, pageSize, search)
			if err != nil {
				return errMsg(err)
			}
			return queuePageMsg(resp)
		}
	}
	return nil
}

func (m Model) deleteSelectedExpense() tea.Cmd {
	if m.activeTab != TabQueue || m.queue == nil || len(m.queue.Items) == 0 {
		return nil
	}

	// Safety check for index
	idx := m.cursor
	if idx < 0 || idx >= len(m.queue.Items) {
		return nil
	}

	id := m.queue.Items[idx].Id

	return func() tea.Msg {
		if m.demoMode {
			// In demo mode, just return success
			return deleteSuccessMsg{}
		}
		err := m.client.DeleteExpense(id)
		if err != nil {
			return errMsg(err)
		}
		return deleteSuccessMsg{}
	}
}
