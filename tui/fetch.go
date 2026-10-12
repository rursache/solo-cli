package tui

import (
	"time"

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

// fetchTaxEstimate loads SOLO's tax estimate for year. A failure hides the
// box, the local breakdown stays valid
func (m Model) fetchTaxEstimate(year int) tea.Cmd {
	c := m.client
	return func() tea.Msg {
		est, err := c.GetTaxEstimate(year)
		if err != nil {
			return taxEstimateMsg{year: year}
		}
		return taxEstimateMsg{year, est}
	}
}

// fetchYear reloads the year scoped data after a year switch
func (m *Model) fetchYear() tea.Cmd {
	m.taxEstimate = nil
	m.estimateYear = m.year
	return tea.Batch(m.fetchSummary, m.fetchTaxEstimate(m.year))
}

// fetchAlerts loads the dashboard alerts. Sources that fail are left out
func (m Model) fetchAlerts() tea.Msg {
	return alertsMsg(m.client.GetAlerts().Active())
}

// fetchDeclaration loads the Declaratia Unica status of the last fiscal
// year, the one filed this year. A failure hides it
func (m Model) fetchDeclaration() tea.Msg {
	year := time.Now().Year() - 1
	d, err := m.client.GetTaxDeclarationDashboard(year)
	if err != nil {
		return declarationMsg(nil)
	}
	if d.FiscalYear == 0 {
		d.FiscalYear = year
	}
	return declarationMsg(d)
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

func (m Model) fetchRejectedRevenues() tea.Msg {
	rejected, err := m.client.ListRejectedRevenues(0, m.pageSize)
	if err != nil {
		// Optional like rejected expenses, an empty list hides the warning
		return rejectedRevenuesMsg(&client.RejectedRevenueResponse{Items: []client.RejectedRevenue{}})
	}
	return rejectedRevenuesMsg(rejected)
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

	offset, pageSize, c, gen := loaded, m.pageSize, m.client, m.listGen
	return func() tea.Msg {
		resp, err := c.ListRevenues(offset, pageSize, "")
		if err != nil {
			return errMsg(err)
		}
		return revenuesPageMsg{resp, gen}
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

	offset, pageSize, gen := loaded, m.pageSize, m.listGen
	c, tab, search := m.client, m.activeTab, m.searchQuery
	switch tab {
	case TabRevenues:
		return func() tea.Msg {
			resp, err := c.ListRevenues(offset, pageSize, search)
			if err != nil {
				return errMsg(err)
			}
			return revenuesPageMsg{resp, gen}
		}
	case TabExpenses:
		return func() tea.Msg {
			resp, err := c.ListExpenses(offset, pageSize, search)
			if err != nil {
				return errMsg(err)
			}
			return expensesPageMsg{resp, gen}
		}
	case TabEFactura:
		return func() tea.Msg {
			resp, err := c.ListEFactura(offset, pageSize, search)
			if err != nil {
				return errMsg(err)
			}
			return efacturaPageMsg{resp, gen}
		}
	case TabQueue:
		return func() tea.Msg {
			resp, err := c.ListQueuedExpenses(offset, pageSize, search)
			if err != nil {
				return errMsg(err)
			}
			return queuePageMsg{resp, gen}
		}
	}
	return nil
}

// deleteQueuedDocument deletes a queued document by id. Demo mode has no
// client, the success handler removes it locally
func (m Model) deleteQueuedDocument(id int) tea.Cmd {
	if m.demoMode {
		return func() tea.Msg { return deleteSuccessMsg{id} }
	}
	c := m.client
	return func() tea.Msg {
		if err := c.DeleteExpense(id); err != nil {
			return errMsg(err)
		}
		return deleteSuccessMsg{id}
	}
}
