package tui

import (
	"fmt"
	"strings"

	"solo-cli/client"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tab represents a navigation tab
type Tab int

const (
	TabDashboard Tab = iota
	TabRevenues
	TabExpenses
	TabEFactura
	TabQueue
)

func (t Tab) String() string {
	switch t {
	case TabDashboard:
		return "Dashboard"
	case TabRevenues:
		return "Revenues"
	case TabExpenses:
		return "Expenses"
	case TabEFactura:
		return "e-Factura"
	case TabQueue:
		return "Queue"
	default:
		return "Unknown"
	}
}

// Model is the main TUI model
type Model struct {
	client    *client.Client
	activeTab Tab
	width     int
	height    int

	// Data
	summary   *client.Summary
	company   *client.CompanyInfo
	revenues  *client.RevenueListResponse
	expenses  *client.ExpenseListResponse
	queue     *client.QueuedExpenseResponse
	efactura  *client.EFacturaListResponse
	companyID string

	// UI state
	loading        bool
	err            error
	spinner        spinner.Model
	cursor         int
	viewportOffset int // First visible item index
	viewportSize   int // Number of visible items
	demoMode       bool

	// Pagination
	revenueOffset int
	expenseOffset int
	queueOffset   int
	pageSize      int
}

// Messages
type summaryMsg *client.Summary
type companyMsg *client.CompanyInfo
type revenuesMsg *client.RevenueListResponse
type expensesMsg *client.ExpenseListResponse
type queueMsg *client.QueuedExpenseResponse
type efacturaMsg *client.EFacturaListResponse
type errMsg error
type deleteSuccessMsg struct{}

// NewModel creates a new TUI model
func NewModel(c *client.Client, companyID string, pageSize int) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	if pageSize <= 0 {
		pageSize = 100 // Default
	}

	return Model{
		client:       c,
		activeTab:    TabDashboard,
		spinner:      s,
		loading:      true,
		pageSize:     pageSize,
		viewportSize: 10, // Show 10 at a time (keeps header visible)
		companyID:    companyID,
	}
}

// NewDemoModel creates a TUI model with demo data for screenshots
func NewDemoModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	return Model{
		activeTab:    TabDashboard,
		spinner:      s,
		loading:      false, // Data already loaded
		pageSize:     100,
		viewportSize: 10,
		demoMode:     true,
		// Pre-populate with demo data
		summary:  client.GetDemoSummary(),
		company:  client.GetDemoCompany(),
		revenues: client.GetDemoRevenues(),
		expenses: client.GetDemoExpenses(),
		queue:    client.GetDemoQueue(),
		efactura: client.GetDemoEFactura(),
	}
}

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
			m.activeTab = (m.activeTab + 1) % 5
			m.cursor = 0
			m.viewportOffset = 0
		case "shift+tab", "left", "h":
			m.activeTab = (m.activeTab - 1 + 5) % 5
			m.cursor = 0
			m.viewportOffset = 0
		case "d", "delete", "backspace":
			if m.activeTab == TabQueue {
				m.loading = true
				return m, m.deleteSelectedExpense()
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				// Scroll viewport up if cursor moves above visible area
				if m.cursor < m.viewportOffset {
					m.viewportOffset = m.cursor
				}
			}
		case "down", "j":
			maxCursor := m.getMaxCursor()
			if m.cursor < maxCursor-1 {
				m.cursor++
				// Scroll viewport down if cursor moves below visible area
				if m.cursor >= m.viewportOffset+m.viewportSize {
					m.viewportOffset = m.cursor - m.viewportSize + 1
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
				m.fetchQueue,
				m.fetchEFactura,
			)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case summaryMsg:
		m.summary = msg
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
	if m.summary != nil && m.revenues != nil && m.expenses != nil && m.queue != nil && m.efactura != nil {
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
		}
	}

	// Help
	b.WriteString("\n")
	b.WriteString("\n")
	helpText := "←/→ tabs • ↑/↓ navigate • r refresh • q quit"
	if m.activeTab == TabQueue {
		helpText = "←/→ tabs • ↑/↓ navigate • d delete • r refresh • q quit"
	}
	b.WriteString(HelpStyle.Render(helpText))

	return b.String()
}

func (m Model) renderTabs() string {
	tabs := []Tab{TabDashboard, TabRevenues, TabExpenses, TabEFactura, TabQueue}
	var parts []string

	for _, tab := range tabs {
		if tab == m.activeTab {
			parts = append(parts, ActiveTabStyle.Render(tab.String()))
		} else {
			parts = append(parts, InactiveTabStyle.Render(tab.String()))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m Model) renderDashboard() string {
	if m.summary == nil {
		return LoadingStyle.Render("Loading summary...")
	}

	var b strings.Builder

	// Company info header
	if m.company != nil {
		b.WriteString(TitleStyle.Render(m.company.Name))
		b.WriteString("\n")
		b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("CUI: %s • Reg: %s", m.company.Code1, m.company.Code2)))
		b.WriteString("\n\n")
	} else if m.companyID == "" {
		b.WriteString(ErrorStyle.Render("‼️  company_id not configured in ~/.config/solo-cli/config.json"))
		b.WriteString("\n")
		b.WriteString(SummaryLabelStyle.Render("Visit https://falcon.solo.ro/settings#!/company and check Network tab for company_ID"))
		b.WriteString("\n\n")
	} else {
		b.WriteString(ErrorStyle.Render("‼️  Could not load company info (invalid company_id?)"))
		b.WriteString("\n\n")
	}

	// Summary box
	summaryContent := fmt.Sprintf(
		"%s %d\n%s %.2f %s\n%s %.2f %s",
		SummaryLabelStyle.Render("Year:"),
		m.summary.Year,
		SummaryLabelStyle.Render("Total Revenues:"),
		m.summary.TotalRevenues,
		m.summary.DisplayCurrency,
		SummaryLabelStyle.Render("Total Expenses:"),
		m.summary.TotalDeductibleExpenses,
		m.summary.DisplayCurrency,
	)

	if m.summary.HasTaxes {
		summaryContent += fmt.Sprintf("\n%s %.2f %s",
			SummaryLabelStyle.Render("Taxes:"),
			m.summary.Taxes,
			m.summary.DisplayCurrency,
		)
	}

	b.WriteString(SummaryBoxStyle.Render(summaryContent))

	// Show pending review info if any
	if m.queue != nil && len(m.queue.Items) > 0 {
		b.WriteString("\n\n")
		b.WriteString(InfoStyle().Render(fmt.Sprintf("ℹ️  %d documents pending review", len(m.queue.Items))))
	}

	return b.String()
}

func (m Model) renderRevenues() string {
	if m.revenues == nil || len(m.revenues.Items) == 0 {
		return "No revenues found"
	}

	var b strings.Builder
	total := len(m.revenues.Items)

	// Show scroll position indicator
	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("Showing %d-%d of %d", m.viewportOffset+1, min(m.viewportOffset+m.viewportSize, total), total)))
	b.WriteString("\n\n")

	b.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%-4s %-18s %12s %-5s %s", "Paid", "Invoice", "Amount", "Curr", "Client")))
	b.WriteString("\n")

	// Calculate visible range
	endIdx := min(m.viewportOffset+m.viewportSize, total)

	for i := m.viewportOffset; i < endIdx; i++ {
		r := m.revenues.Items[i]
		paid := "✅"
		if !r.IsPaid {
			paid = "❌"
		}

		// Truncate client name
		clientName := r.ClientName
		if len(clientName) > 30 {
			clientName = clientName[:27] + "..."
		}

		row := fmt.Sprintf("%-4s %-18s %12.2f %-5s %s",
			paid,
			truncate(r.SerialCode, 18),
			r.Total,
			r.Currency.ShortName,
			clientName,
		)

		if i == m.cursor {
			b.WriteString(TableSelectedStyle.Render(row))
		} else {
			b.WriteString(TableRowStyle.Render(row))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderExpenses() string {
	if m.expenses == nil || len(m.expenses.Items) == 0 {
		return "No expenses found"
	}

	var b strings.Builder
	total := len(m.expenses.Items)

	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("Showing %d-%d of %d", m.viewportOffset+1, min(m.viewportOffset+m.viewportSize, total), total)))
	b.WriteString("\n\n")

	b.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%-3s %12s %-5s %-25s %s", "Ded", "Amount", "Curr", "Category", "Supplier")))
	b.WriteString("\n")

	endIdx := min(m.viewportOffset+m.viewportSize, total)

	for i := m.viewportOffset; i < endIdx; i++ {
		e := m.expenses.Items[i]
		deductIcon := "✅"
		if strings.Contains(e.Category, "Nedeductibilă") {
			deductIcon = "❌"
		}

		category := truncate(e.Category, 25)
		supplier := truncate(e.SupplierName, 25)

		row := fmt.Sprintf("%-3s %12.2f %-5s %-25s %s",
			deductIcon,
			e.Total,
			e.Currency.ShortName,
			category,
			supplier,
		)

		if i == m.cursor {
			b.WriteString(TableSelectedStyle.Render(row))
		} else {
			b.WriteString(TableRowStyle.Render(row))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderQueue() string {
	if m.queue == nil || len(m.queue.Items) == 0 {
		return "No documents in queue"
	}

	var b strings.Builder
	total := len(m.queue.Items)

	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("Showing %d-%d of %d", m.viewportOffset+1, min(m.viewportOffset+m.viewportSize, total), total)))
	b.WriteString("\n\n")

	b.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%-30s %8s %s", "Document", "Days", "Status")))
	b.WriteString("\n")

	endIdx := min(m.viewportOffset+m.viewportSize, total)

	for i := m.viewportOffset; i < endIdx; i++ {
		q := m.queue.Items[i]
		status := "Pending"
		if q.IsOverdue {
			status = "OVERDUE"
		}

		row := fmt.Sprintf("%-30s %8d %s",
			truncate(q.DocumentName, 30),
			q.DaysPassed,
			status,
		)

		if i == m.cursor {
			b.WriteString(TableSelectedStyle.Render(row))
		} else {
			b.WriteString(TableRowStyle.Render(row))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// Fetch commands
func (m Model) fetchSummary() tea.Msg {
	summary, err := m.client.GetSummary()
	if err != nil {
		return errMsg(err)
	}
	return summaryMsg(summary)
}

func (m Model) fetchCompany() tea.Msg {
	if m.companyID == "" {
		return companyMsg(nil)
	}
	company, err := m.client.GetCompanyInfo(m.companyID)
	if err != nil {
		// Company info is optional, don't fail
		return companyMsg(nil)
	}
	return companyMsg(company)
}

func (m Model) fetchRevenues() tea.Msg {
	revenues, err := m.client.ListRevenues(m.revenueOffset, m.pageSize)
	if err != nil {
		return errMsg(err)
	}
	return revenuesMsg(revenues)
}

func (m Model) fetchExpenses() tea.Msg {
	expenses, err := m.client.ListExpenses(m.expenseOffset, m.pageSize)
	if err != nil {
		return errMsg(err)
	}
	return expensesMsg(expenses)
}

func (m Model) fetchQueue() tea.Msg {
	queue, err := m.client.ListQueuedExpenses(m.queueOffset, m.pageSize)
	if err != nil {
		return errMsg(err)
	}
	return queueMsg(queue)
}

// Helper
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// WarningStyle returns styled warning text
func WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
}

// InfoStyle returns styled info text (blue)
func InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))
}

func (m Model) fetchEFactura() tea.Msg {
	efactura, err := m.client.ListEFactura(0, m.pageSize)
	if err != nil {
		return errMsg(err)
	}
	return efacturaMsg(efactura)
}

func (m Model) renderEFactura() string {
	if m.efactura == nil || len(m.efactura.Items) == 0 {
		return "No e-Factura documents found"
	}

	var b strings.Builder
	total := len(m.efactura.Items)

	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("Showing %d-%d of %d", m.viewportOffset+1, min(m.viewportOffset+m.viewportSize, total), total)))
	b.WriteString("\n\n")

	b.WriteString(TableHeaderStyle.Render(fmt.Sprintf("%-20s %12s %-5s %-12s %s", "Serial", "Amount", "Curr", "Date", "Party")))
	b.WriteString("\n")

	endIdx := min(m.viewportOffset+m.viewportSize, total)

	for i := m.viewportOffset; i < endIdx; i++ {
		e := m.efactura.Items[i]
		row := fmt.Sprintf("%-20s %12.2f %-5s %-12s %s",
			truncate(e.SerialCode, 20),
			e.TotalAmount,
			e.CurrencyCode,
			e.InvoiceDate,
			truncate(e.PartyName, 25),
		)

		if i == m.cursor {
			b.WriteString(TableSelectedStyle.Render(row))
		} else {
			b.WriteString(TableRowStyle.Render(row))
		}
		b.WriteString("\n")
	}

	return b.String()
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
