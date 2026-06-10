package tui

import (
	"solo-cli/client"
	"solo-cli/config"
	"solo-cli/taxes"

	"github.com/charmbracelet/bubbles/spinner"
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
	year      int // Displayed year (0 = current, set from the first summary)
	maxYear   int // Current fiscal year, the upper bound for year switching

	// Data
	summary      *client.Summary
	company      *client.CompanyInfo
	caenCodes    []client.CAENCode
	revenues     *client.RevenueListResponse
	expenses     *client.ExpenseListResponse
	rejected     *client.RejectedExpenseResponse
	queue        *client.QueuedExpenseResponse
	efactura     *client.EFacturaListResponse
	taxBreakdown *taxes.TaxBreakdown
	taxConfig    *config.TaxConfig

	// UI state
	loading        bool
	err            error
	spinner        spinner.Model
	cursor         int
	searching      bool   // Typing in the search input
	searchInput    string // Text being typed
	searchQuery    string // Applied server-side filter for the active list tab
	marqueeOffset  int    // Scroll position of the focused row's marquee
	viewportOffset int // First visible item index
	viewportSize   int // Number of visible items
	taxesScroll    int  // Scroll offset for taxes tab
	taxesLines     int  // Total line count of taxes content
	fetchingMore   bool // A next-page fetch is in flight
	demoMode       bool

	pageSize int
}

// Messages
type summaryMsg *client.Summary
type companyMsg *client.CompanyInfo
type caenMsg []client.CAENCode
type revenuesMsg *client.RevenueListResponse
type expensesMsg *client.ExpenseListResponse
type rejectedMsg *client.RejectedExpenseResponse
type queueMsg *client.QueuedExpenseResponse
type efacturaMsg *client.EFacturaListResponse
type errMsg error
type deleteSuccessMsg struct{}

// Page messages append to the already loaded list instead of replacing it
type revenuesPageMsg *client.RevenueListResponse
type expensesPageMsg *client.ExpenseListResponse
type queuePageMsg *client.QueuedExpenseResponse
type efacturaPageMsg *client.EFacturaListResponse

func newSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))
	return s
}

// NewModel creates a new TUI model
func NewModel(c *client.Client, pageSize int) Model {
	if pageSize <= 0 {
		pageSize = 100 // Default
	}

	// Load tax config (non-fatal if it fails)
	taxCfg, _ := config.LoadTaxes()

	return Model{
		client:       c,
		activeTab:    TabDashboard,
		spinner:      newSpinner(),
		loading:      true,
		pageSize:     pageSize,
		viewportSize: 10, // Fallback until the first WindowSizeMsg arrives
		taxConfig:    taxCfg,
	}
}

// NewDemoModel creates a TUI model with demo data for screenshots
func NewDemoModel() Model {
	demoSummary := client.GetDemoSummary()
	taxCfg := config.DefaultTaxConfig()
	taxBreakdown := taxes.Calculate(demoSummary.TotalRevenues, demoSummary.TotalDeductibleExpenses, taxCfg)

	return Model{
		activeTab:    TabDashboard,
		spinner:      newSpinner(),
		loading:      false, // Data already loaded
		pageSize:     100,
		viewportSize: 10,
		demoMode:     true,
		taxConfig:    taxCfg,
		taxBreakdown: taxBreakdown,
		// Pre-populate with demo data
		summary:   demoSummary,
		company:   client.GetDemoCompany(),
		caenCodes: client.GetDemoCAENCodes(),
		revenues: client.GetDemoRevenues(),
		expenses: client.GetDemoExpenses(),
		rejected: client.GetDemoRejectedExpenses(),
		queue:    client.GetDemoQueue(),
		efactura: client.GetDemoEFactura(),
	}
}
