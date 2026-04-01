package tui

import (
	"fmt"
	"strings"

	"solo-cli/client"
	"solo-cli/config"
	"solo-cli/taxes"

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
	TabTaxes
)

const tabCount = 6

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
	case TabTaxes:
		return "Taxes"
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
	summary      *client.Summary
	company      *client.CompanyInfo
	revenues     *client.RevenueListResponse
	expenses     *client.ExpenseListResponse
	rejected     *client.RejectedExpenseResponse
	queue        *client.QueuedExpenseResponse
	efactura     *client.EFacturaListResponse
	companyID    string
	taxBreakdown *taxes.TaxBreakdown
	taxConfig    *config.TaxConfig

	// UI state
	loading        bool
	err            error
	spinner        spinner.Model
	cursor         int
	viewportOffset int // First visible item index
	viewportSize   int // Number of visible items
	taxesScroll    int // Scroll offset for taxes tab
	taxesLines     int // Total line count of taxes content
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
type rejectedMsg *client.RejectedExpenseResponse
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

	// Load tax config (non-fatal if it fails)
	taxCfg, _ := config.LoadTaxes()

	return Model{
		client:       c,
		activeTab:    TabDashboard,
		spinner:      s,
		loading:      true,
		pageSize:     pageSize,
		viewportSize: 10, // Show 10 at a time (keeps header visible)
		companyID:    companyID,
		taxConfig:    taxCfg,
	}
}

// NewDemoModel creates a TUI model with demo data for screenshots
func NewDemoModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	demoSummary := client.GetDemoSummary()
	taxCfg := config.DefaultTaxConfig()
	taxBreakdown := taxes.Calculate(demoSummary.TotalRevenues, demoSummary.TotalDeductibleExpenses, taxCfg)

	return Model{
		activeTab:    TabDashboard,
		spinner:      s,
		loading:      false, // Data already loaded
		pageSize:     100,
		viewportSize: 10,
		demoMode:     true,
		taxConfig:    taxCfg,
		taxBreakdown: taxBreakdown,
		// Pre-populate with demo data
		summary:  demoSummary,
		company:  client.GetDemoCompany(),
		revenues: client.GetDemoRevenues(),
		expenses: client.GetDemoExpenses(),
		rejected: client.GetDemoRejectedExpenses(),
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
					if m.cursor >= m.viewportOffset+m.viewportSize {
						m.viewportOffset = m.cursor - m.viewportSize + 1
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
			content := m.renderTaxes()
			lines := strings.Split(content, "\n")
			// Available height: total height minus header (title+tabs = ~4 lines) and footer (help = ~3 lines)
			availHeight := m.height - 7
			if availHeight < 5 {
				availHeight = 5
			}
			// Clamp scroll
			maxScroll := len(lines) - availHeight
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.taxesScroll > maxScroll {
				m.taxesScroll = maxScroll
			}
			// Slice visible lines
			end := m.taxesScroll + availHeight
			if end > len(lines) {
				end = len(lines)
			}
			b.WriteString(strings.Join(lines[m.taxesScroll:end], "\n"))
			if m.taxesScroll > 0 || end < len(lines) {
				b.WriteString("\n")
				b.WriteString(SummaryLabelStyle.Render("↑↓ scroll to see more"))
			}
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
	tabs := []Tab{TabDashboard, TabRevenues, TabExpenses, TabEFactura, TabQueue, TabTaxes}
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
	var b strings.Builder

	// Show rejected expenses warning if any
	if m.rejected != nil && len(m.rejected.Items) > 0 {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("⚠️  %d rejected expense(s):", len(m.rejected.Items))))
		b.WriteString("\n")
		for _, r := range m.rejected.Items {
			docName := truncate(r.DocumentName, 30)
			reason := truncate(r.Reason, 50)
			b.WriteString(WarningStyle().Render(fmt.Sprintf("   • %s - %s", docName, reason)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if m.expenses == nil || len(m.expenses.Items) == 0 {
		b.WriteString("No expenses found")
		return b.String()
	}

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

func (m Model) fetchRejected() tea.Msg {
	rejected, err := m.client.ListRejectedExpenses(0, m.pageSize)
	if err != nil {
		// Rejected expenses are optional, don't fail if unavailable
		return rejectedMsg(&client.RejectedExpenseResponse{Items: []client.RejectedExpense{}})
	}
	return rejectedMsg(rejected)
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

func (m Model) renderTaxes() string {
	if m.taxBreakdown == nil {
		if m.taxConfig == nil {
			return ErrorStyle.Render("Could not load taxes.json config")
		}
		return LoadingStyle.Render("Loading tax data...")
	}

	t := m.taxBreakdown
	var b strings.Builder

	// Header
	yearStr := ""
	if m.summary != nil {
		yearStr = fmt.Sprintf(" (%d)", m.summary.Year)
	}
	b.WriteString(TitleStyle.Render(fmt.Sprintf("Tax Breakdown%s", yearStr)))
	b.WriteString("\n")

	// Net income summary
	incomeContent := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s (%.1f @ %.0f RON salariu minim brut)",
		SummaryLabelStyle.Render("Total Revenues:"),
		SummaryValueStyle.Render(taxes.FormatRON(m.summary.TotalRevenues)),
		SummaryLabelStyle.Render("Deductible Expenses:"),
		SummaryValueStyle.Render(taxes.FormatRON(m.summary.TotalDeductibleExpenses)),
		SummaryLabelStyle.Render("Net Income:"),
		SummaryValueStyle.Render(taxes.FormatRON(t.NetIncome)),
		t.SalariesCount,
		t.SalariuMinimBrut,
	)
	b.WriteString(CompactBoxStyle.Render(incomeContent))
	b.WriteString("\n")

	// CAS
	casContent := fmt.Sprintf(
		"%s %s\n%s %s → %s %s",
		SummaryLabelStyle.Render("Bracket:"),
		t.CAS.Label,
		SummaryLabelStyle.Render("Base:"),
		SummaryValueStyle.Render(taxes.FormatRON(t.CAS.Base)),
		SummaryLabelStyle.Render("Amount:"),
		SummaryValueStyle.Render(taxes.FormatRON(t.CAS.Amount)),
	)
	if t.CAS.NextLabel != "" {
		bufferStyle := secondaryStyle
		if t.CAS.BufferToNext < 5000 {
			bufferStyle = dangerStyle
		} else if t.CAS.BufferToNext < 15000 {
			bufferStyle = warningStyle
		}
		casContent += fmt.Sprintf("\n%s %s → %s",
			SummaryLabelStyle.Render("Buffer:"),
			bufferStyle.Render(taxes.FormatRON(t.CAS.BufferToNext)),
			SummaryLabelStyle.Render(t.CAS.NextLabel),
		)
	}
	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("CAS (%.0f%%)", t.CAS.Percentage)))
	b.WriteString("\n")
	b.WriteString(CompactBoxStyle.Render(casContent))
	b.WriteString("\n")

	// CASS
	cassContent := fmt.Sprintf(
		"%s %s\n%s %s → %s %s",
		SummaryLabelStyle.Render("Bracket:"),
		t.CASS.Label,
		SummaryLabelStyle.Render("Base:"),
		SummaryValueStyle.Render(taxes.FormatRON(t.CASS.Base)),
		SummaryLabelStyle.Render("Amount:"),
		SummaryValueStyle.Render(taxes.FormatRON(t.CASS.Amount)),
	)
	if t.CASS.NextLabel != "" {
		bufferStyle := secondaryStyle
		if t.CASS.BufferToNext < 5000 {
			bufferStyle = dangerStyle
		} else if t.CASS.BufferToNext < 15000 {
			bufferStyle = warningStyle
		}
		cassContent += fmt.Sprintf("\n%s %s → %s",
			SummaryLabelStyle.Render("Buffer:"),
			bufferStyle.Render(taxes.FormatRON(t.CASS.BufferToNext)),
			SummaryLabelStyle.Render(t.CASS.NextLabel),
		)
	}
	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("CASS (%.0f%%)", t.CASS.Percentage)))
	b.WriteString("\n")
	b.WriteString(CompactBoxStyle.Render(cassContent))
	b.WriteString("\n")

	// Income Tax
	taxableIncome := t.NetIncome - t.CAS.Amount - t.CASS.Amount
	if taxableIncome < 0 {
		taxableIncome = 0
	}
	itContent := fmt.Sprintf(
		"%s %s (Net Income - CAS - CASS)\n%s %s",
		SummaryLabelStyle.Render("Taxable Income:"),
		SummaryValueStyle.Render(taxes.FormatRON(taxableIncome)),
		SummaryLabelStyle.Render("Amount:"),
		SummaryValueStyle.Render(taxes.FormatRON(t.IncomeTax)),
	)
	b.WriteString(SummaryLabelStyle.Render(fmt.Sprintf("Income Tax (%.0f%%)", m.taxConfig.IncomeTaxPercent)))
	b.WriteString("\n")
	b.WriteString(CompactBoxStyle.Render(itContent))
	b.WriteString("\n")

	// Totals
	totalsContent := fmt.Sprintf(
		"%s %s\n%s %s\n%s %.1f%%",
		SummaryLabelStyle.Render("Total Taxes:"),
		ErrorStyle.Render(taxes.FormatRON(t.TotalTaxes)),
		SummaryLabelStyle.Render("Net After Tax:"),
		PaidStyle.Render(taxes.FormatRON(t.NetAfterTax)),
		SummaryLabelStyle.Render("Effective Tax Rate:"),
		t.EffectiveRate,
	)
	b.WriteString(CompactBoxStyle.Render(totalsContent))

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
