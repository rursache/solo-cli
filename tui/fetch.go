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
	revenues, err := m.client.ListRevenues(m.revenueOffset, m.pageSize, m.searchFor(TabRevenues))
	if err != nil {
		return errMsg(err)
	}
	return revenuesMsg(revenues)
}

func (m Model) fetchExpenses() tea.Msg {
	expenses, err := m.client.ListExpenses(m.expenseOffset, m.pageSize, m.searchFor(TabExpenses))
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
	queue, err := m.client.ListQueuedExpenses(m.queueOffset, m.pageSize, m.searchFor(TabQueue))
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
