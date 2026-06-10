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
		withClient(runRevenues)
	case "expenses", "expense", "exp":
		withClient(runExpenses)
	case "queue", "q":
		withClientArgs(runQueue, cmdArgs)
	case "efactura", "einvoice", "ei":
		withClient(runEFactura)
	case "company":
		withClient(runCompany)
	case "taxes", "tax":
		withClientArgs(runTaxes, cmdArgs)
	case "upload", "up":
		withClientArgs(runUpload, cmdArgs)
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

Commands:
  summary [year]  Show account summary (year, revenues, expenses, taxes)
  taxes [year]    Show tax breakdown with thresholds (alias: tax)
  revenues        List revenue invoices (aliases: revenue, rev)
  expenses        List expenses (aliases: expense, exp)
  queue           List expense queue (alias: q). Subcommands: delete <id>
  efactura        List e-Factura documents (aliases: einvoice, ei)
  company         Show company profile
  upload <file>   Upload expense document (alias: up)
  setup-skills    Install AI skills for Claude Code and other agents
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
