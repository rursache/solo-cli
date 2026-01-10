package main

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"solo-cli/client"
	"solo-cli/config"
	"solo-cli/tui"
)

var version = "dev"

func main() {
	// Parse global flags first
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--config" || args[i] == "-c" {
			if i+1 < len(args) {
				config.SetConfigPath(args[i+1])
				// Remove the flag and value from args
				args = append(args[:i], args[i+2:]...)
				break
			} else {
				fmt.Fprintln(os.Stderr, "Error: --config requires a path argument")
				os.Exit(1)
			}
		}
	}

	// Handle no args or help
	if len(args) < 1 {
		runTUI()
		return
	}

	cmd := args[0]
	cmdArgs := args[1:] // Additional arguments for commands

	switch cmd {
	case "help", "--help", "-h":
		printHelp()
	case "version", "--version", "-v":
		fmt.Printf("solo-cli %s\n", version)
	case "summary":
		withClientArgs(runSummary, cmdArgs)
	case "revenues", "revenue", "rev":
		withClient(runRevenues)
	case "expenses", "expense", "exp":
		withClient(runExpenses)
	case "queue", "q":
		withClientArgs(runQueue, cmdArgs)
	case "efactura", "einvoice", "ei":
		withClient(runEFactura)
	case "company":
		withClient(runCompany)
	case "upload", "up":
		withClientArgs(runUpload, cmdArgs)
	case "tui":
		runTUI()
	case "demo":
		runDemoTUI()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`solo-cli - SOLO.ro accounting platform CLI

Usage:
  solo-cli [options] [command] [args]

Commands:
  summary [year]  Show account summary (year, revenues, expenses, taxes)
  revenues        List revenue invoices (aliases: revenue, rev)
  expenses        List expenses (aliases: expense, exp)
  queue           List expense queue (alias: q). Subcommands: delete <id>
  efactura        List e-Factura documents (aliases: einvoice, ei)
  company         Show company profile
  upload <file>   Upload expense document (alias: up)
  tui             Start interactive TUI (default when no command)
  demo            Start TUI with demo data (for screenshots)

Options:
  --config, -c    Path to custom config file
  help, -h        Show this help message
  version, -v     Show version

Config:
  Default: ~/.config/solo-cli/config.json

Examples:
  solo-cli                          # Start TUI
  solo-cli summary                  # Show current year summary
  solo-cli summary 2025             # Show 2025 summary
  solo-cli upload invoice.pdf       # Upload expense document
  solo-cli queue delete 123         # Delete queued item
  solo-cli -c ~/my-config.json rev  # Use custom config
  solo-cli expenses | grep -i "food"

`)
}

// withClient handles auth and runs a command with a client
func withClient(fn func(*client.Client)) {
	// Ensure config file exists
	if err := config.EnsureExists(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrCredentialsMissing) {
			configPath, _ := config.GetConfigPath()
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Please edit: %s\n", configPath)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create API client with user agent from config
	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = config.DefaultUserAgent
	}
	apiClient, err := client.New(userAgent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	// Try to load saved cookies first
	needsLogin := true
	if loaded, _ := apiClient.LoadCookies(); loaded {
		if _, err := apiClient.GetSummary(); err == nil {
			needsLogin = false
		}
	}

	// Login if cookies are missing, expired, or invalid
	if needsLogin {
		fmt.Fprintln(os.Stderr, "Logging in to SOLO.ro...")
		if err := apiClient.Login(cfg.Username, cfg.Password); err != nil {
			if errors.Is(err, client.ErrAuthenticationFailed) {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				fmt.Fprintln(os.Stderr, "Please check your credentials in the config file.")
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Login error: %v\n", err)
			os.Exit(1)
		}
		if err := apiClient.SaveCookies(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save session: %v\n", err)
		}
	}

	fn(apiClient)
}

// withClientArgs handles auth and runs a command with client and args
func withClientArgs(fn func(*client.Client, []string), args []string) {
	if err := config.EnsureExists(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrCredentialsMissing) {
			configPath, _ := config.GetConfigPath()
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Please edit: %s\n", configPath)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = config.DefaultUserAgent
	}
	apiClient, err := client.New(userAgent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	needsLogin := true
	if loaded, _ := apiClient.LoadCookies(); loaded {
		if _, err := apiClient.GetSummary(); err == nil {
			needsLogin = false
		}
	}

	if needsLogin {
		fmt.Fprintln(os.Stderr, "Logging in to SOLO.ro...")
		if err := apiClient.Login(cfg.Username, cfg.Password); err != nil {
			if errors.Is(err, client.ErrAuthenticationFailed) {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				fmt.Fprintln(os.Stderr, "Please check your credentials in the config file.")
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Login error: %v\n", err)
			os.Exit(1)
		}
		if err := apiClient.SaveCookies(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save session: %v\n", err)
		}
	}

	fn(apiClient, args)
}

func runTUI() {
	// Ensure config file exists
	if err := config.EnsureExists(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrCredentialsMissing) {
			configPath, _ := config.GetConfigPath()
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Please edit: %s\n", configPath)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = config.DefaultUserAgent
	}
	apiClient, err := client.New(userAgent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	needsLogin := true
	if loaded, _ := apiClient.LoadCookies(); loaded {
		if _, err := apiClient.GetSummary(); err == nil {
			needsLogin = false
		}
	}

	if needsLogin {
		fmt.Fprintln(os.Stderr, "Logging in to SOLO.ro...")
		if err := apiClient.Login(cfg.Username, cfg.Password); err != nil {
			if errors.Is(err, client.ErrAuthenticationFailed) {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				fmt.Fprintln(os.Stderr, "Please check your credentials in the config file.")
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Login error: %v\n", err)
			os.Exit(1)
		}
		if err := apiClient.SaveCookies(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save session: %v\n", err)
		}
	}

	model := tui.NewModel(apiClient, cfg.CompanyID, cfg.PageSize)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func runDemoTUI() {
	model := tui.NewDemoModel()
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func runSummary(c *client.Client, args []string) {
	// Parse optional year argument
	year := 0
	if len(args) > 0 {
		if _, err := fmt.Sscanf(args[0], "%d", &year); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid year: %s\n", args[0])
			os.Exit(1)
		}
	}

	summary, err := c.GetSummaryForYear(year)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Year: %d\n", summary.Year)
	fmt.Printf("Revenues: %.2f %s\n", summary.TotalRevenues, summary.DisplayCurrency)
	fmt.Printf("Expenses: %.2f %s\n", summary.TotalDeductibleExpenses, summary.DisplayCurrency)
	if summary.HasTaxes {
		fmt.Printf("Taxes: %.2f %s\n", summary.Taxes, summary.DisplayCurrency)
	}
}

func runRevenues(c *client.Client) {
	revenues, err := c.ListRevenues(0, 100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	for _, r := range revenues.Items {
		paid := "UNPAID"
		if r.IsPaid {
			paid = "PAID"
		}
		fmt.Printf("%s\t%.2f %s\t%s\t%s\n", r.SerialCode, r.Total, r.Currency.ShortName, paid, r.ClientName)
	}
}

func runExpenses(c *client.Client) {
	expenses, err := c.ListExpenses(0, 100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	for _, e := range expenses.Items {
		fmt.Printf("%.2f %s\t%s\t%s\n", e.Total, e.Currency.ShortName, e.Category, e.SupplierName)
	}
}

func runQueue(c *client.Client, args []string) {
	// Handle subcommands
	if len(args) > 0 {
		cmd := args[0]
		if cmd == "delete" || cmd == "del" || cmd == "rm" {
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "Error: missing ID")
				fmt.Fprintln(os.Stderr, "Usage: solo-cli queue delete <id>")
				os.Exit(1)
			}
			idStr := args[1]
			var id int
			if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid ID '%s' (must be a number)\n", idStr)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Deleting queued item %d...\n", id)
			if err := c.DeleteExpense(id); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Item deleted successfully.")
			return
		}
	}

	queue, err := c.ListQueuedExpenses(0, 100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	for _, q := range queue.Items {
		overdue := ""
		if q.IsOverdue {
			overdue = "OVERDUE"
		}
		fmt.Printf("%s\t%d days\t%s\t(ID: %d)\n", q.DocumentName, q.DaysPassed, overdue, q.Id)
	}
}

func runEFactura(c *client.Client) {
	efactura, err := c.ListEFactura(0, 100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	for _, e := range efactura.Items {
		fmt.Printf("%s\t%.2f %s\t%s\t%s\n", e.SerialCode, e.TotalAmount, e.CurrencyCode, e.InvoiceDate, e.PartyName)
	}
}

func runCompany(c *client.Client) {
	cfg, _ := config.Load()
	if cfg.CompanyID == "" {
		fmt.Fprintln(os.Stderr, "Error: company_id not configured")
		fmt.Fprintln(os.Stderr, "Add 'company_id' to ~/.config/solo-cli/config.json")
		os.Exit(1)
	}
	company, err := c.GetCompanyInfo(cfg.CompanyID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Name: %s\n", company.Name)
	fmt.Printf("CUI: %s\n", company.Code1)
	fmt.Printf("Reg: %s\n", company.Code2)
	fmt.Printf("Address: %s\n", company.Address)
}

func runUpload(c *client.Client, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: no file specified")
		fmt.Fprintln(os.Stderr, "Usage: solo-cli upload <file>")
		os.Exit(1)
	}

	filePath := args[0]

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", filePath)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Uploading %s...\n", filePath)

	filename, err := c.UploadDocument(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Uploaded: %s\n", filename)
	fmt.Println("Document added to expense queue for processing.")
}
