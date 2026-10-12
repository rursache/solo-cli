package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

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
		maybePromptSkillInstall()
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
		runRevenuesCmd(cmdArgs)
	case "invoice", "inv":
		runInvoice(cmdArgs)
	case "expenses", "expense", "exp":
		runExpensesCmd(cmdArgs)
	case "queue", "q":
		runQueueCmd(cmdArgs)
	case "efactura", "einvoice", "ei":
		runEFacturaCmd(cmdArgs)
	case "company":
		if len(cmdArgs) > 0 {
			fmt.Fprintf(os.Stderr, "Error: unexpected argument %q\nUsage: solo-cli company\n", cmdArgs[0])
			os.Exit(1)
		}
		withClient(runCompany)
	case "taxes", "tax":
		withClientArgs(runTaxes, cmdArgs)
	case "alerts", "alert":
		runAlertsCmd(cmdArgs)
	case "declaration", "du":
		withClientArgs(runDeclaration, cmdArgs)
	case "upload", "up":
		withClientArgs(runUpload, cmdArgs)
	case "documents", "docs":
		runDocuments(cmdArgs)
	case "report", "reports":
		runReport(cmdArgs)
	case "subscription":
		runSubscription(cmdArgs)
	case "referral":
		runReferral(cmdArgs)
	case "clients", "customers":
		runClients(cmdArgs)
	case "rates":
		runRates(cmdArgs)
	case "setup-skills":
		runSetupSkills()
	case "tui":
		maybePromptSkillInstall()
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

Overview:
  summary [year]        Account summary, active alerts follow on stderr
  taxes [year]          Local tax breakdown with thresholds plus SOLO's own estimate (alias: tax)
  declaration [year]    Declaratia Unica status, default last year (alias: du)
  alerts                Active account alerts (VAT threshold, CAEN update, procura) (alias: alert)

Revenues and invoices:
  revenues              List invoices, last column is the invoice code (aliases: revenue, rev)
  revenues queue        List revenue documents still being processed (aliases: queued, q)
  revenues rejected     List rejected revenue documents
  revenues delete <id> [-y]
                        Delete a queued or rejected revenue document (aliases: del, rm)
  invoice <code>        Show an invoice with its lines (alias: inv)
  invoice paid <code> [YYYY-MM-DD] [-y]
                        Mark an invoice paid, default today
  invoice unpaid <code> [-y]
                        Mark an invoice unpaid
  invoice download <code> [--xml | --original] [-o path] [-f]
                        Save the invoice PDF, its e-Factura XML or the uploaded original (alias: dl)
  invoice delete <code> [-y]
                        Delete the last issued invoice (aliases: del, rm)

e-Factura:
  efactura              List received e-Factura documents (aliases: einvoice, ei)
  efactura status <invoice>
                        e-Factura status of an issued invoice (code or serial)
  efactura send <invoice> [-y]
                        Send an invoice left on manual sending to ANAF
  efactura expedite <invoice> [-y]
                        Send a scheduled e-Factura now
  efactura settings     SPV connection and sending preferences
  efactura settings set [--delay 0|2] [--contact-info yes|no] [-y]
                        Change the sending preferences, connecting the SPV happens on the web

Expenses:
  expenses              List expenses, last column is the document code if any (aliases: expense, exp)
  expenses rejected     List rejected expense documents with the ID queue delete takes
  expenses download <code> [-o path] [-f]
                        Save an expense, queued or rejected document (alias: dl)
  queue                 List the expense queue (alias: q)
  queue delete <id> [-y]
                        Delete a queued or rejected expense document (aliases: del, rm)
  upload [-r] <file>    Upload an expense document, -r (--revenue) for a revenue one (alias: up)

Documents:
  documents             List filed tax declarations (alias: docs)
  documents pfa         List the Documente PFA files
  documents download <code> [-o path] [-f]
                        Save a declaration or PFA document (alias: dl)
  report [year] [-o path] [-f]
                        Save the cash flow register PDF (registrul de incasari si plati, alias: reports)

Account:
  company               Company profile, VIES status and bank accounts
  subscription          SOLO.ro subscription, changes stay on the web
  referral              Referral link, prizes and rewards
  clients [text]        List clients, optionally filtered by a search text (alias: customers)
  rates [YYYY-MM-DD]    BNR exchange rates for a day, default today

Other:
  setup-skills          Install AI skills for Claude Code and other agents
  tui                   Start interactive TUI (default when no command)
  demo                  Start TUI with demo data (for screenshots)

Options:
  --config, -c          Path to custom config file
  help, -h              Show this help message
  version, -v           Show version

Command flags:
  --yes, -y             Confirm a destructive or outward-facing action, required without a terminal
  --output, -o          Path or directory for a download, default the current directory
  --force, -f           Overwrite an existing file when downloading

Config:
  Default: ~/.config/solo-cli/config.json

Examples:
  solo-cli                            # Start TUI
  solo-cli summary 2025               # Show 2025 summary
  solo-cli upload invoice.pdf         # Upload expense document
  solo-cli upload -r invoice.pdf      # Upload revenue document
  solo-cli queue delete 123 -y        # Delete a queued expense document
  solo-cli invoice paid <code> -y     # Mark an invoice paid today
  solo-cli invoice download <code>    # Save the invoice PDF
  solo-cli efactura status INV-001    # e-Factura status of an issued invoice
  solo-cli report 2025 -o ~/Desktop   # Save the 2025 cash flow register
  solo-cli -c ~/my-config.json rev    # Use custom config
  solo-cli expenses | grep -i "food"

`)
}

func runTUI() {
	apiClient, cfg := setupClient()

	model := tui.NewModel(apiClient, cfg.PageSize)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func runDemoTUI() {
	model := tui.NewDemoModel()
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
