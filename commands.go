package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"solo-cli/client"
	"solo-cli/config"
	"solo-cli/taxes"
)

// parseYearArg reads an optional year argument, 0 meaning current year
func parseYearArg(args []string) int {
	if len(args) == 0 {
		return 0
	}
	year := 0
	if _, err := fmt.Sscanf(args[0], "%d", &year); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid year: %s\n", args[0])
		os.Exit(1)
	}
	return year
}

func runSummary(c *client.Client, args []string) {
	summary, err := c.GetSummaryForYear(parseYearArg(args))
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
	printSummaryAlerts(c)
}

func runRevenues(c *client.Client, pageSize int) {
	warnRejectedRevenues(c, pageSize)

	for _, r := range listed(c.ListAllRevenues(pageSize, "")) {
		paid := "UNPAID"
		if r.IsPaid {
			paid = "PAID"
		}
		fmt.Printf("%s\t%.2f %s\t%s\t%s\t%s\n", r.SerialCode, r.Total, r.Currency.ShortName, paid, r.ClientName, r.UniqueCode)
	}
}

func runExpenses(c *client.Client, pageSize int) {
	// Check for rejected expenses first
	rejected, err := c.ListAllRejectedExpenses(pageSize)
	if lookupErr(err) == nil && len(rejected) > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  %d rejected expense(s):\n", len(rejected))
		for _, r := range rejected {
			fmt.Fprintf(os.Stderr, "   • %s - %s%s\n", r.DocumentName, r.Reason, rejectedRef(r.Id, r.DocumentCode))
		}
		fmt.Fprintln(os.Stderr)
	}

	for _, e := range listed(c.ListAllExpenses(pageSize, "")) {
		fmt.Printf("%.2f %s\t%s\t%s%s\n", e.Total, e.Currency.ShortName, e.Category, e.SupplierName, lastColumn(strOrEmpty(e.DocumentCode)))
	}
}

func listRejectedExpenses(c *client.Client, pageSize int) {
	for _, r := range listed(c.ListAllRejectedExpenses(pageSize)) {
		fmt.Printf("%s\t%s\t(ID: %d)%s\n", r.DocumentName, r.Reason, r.Id, lastColumn(r.DocumentCode))
	}
}

const queueUsage = "Usage: solo-cli queue [delete <id> [-y]]"

// runQueueCmd validates the queue subcommand before signing in
func runQueueCmd(args []string) {
	if len(args) == 0 {
		withPagedClient(runQueue)
		return
	}
	switch args[0] {
	case "delete", "del", "rm":
		flags, err := parseCmdFlags(args[1:])
		if err != nil {
			exitErr(err)
		}
		id, err := parseDocumentID(flags.Args, "solo-cli queue delete <id> [-y]")
		if err != nil {
			exitErr(err)
		}
		c, cfg := setupClient()
		if err := deleteExpenseDocument(c, cfg.PageSize, id, flags.Yes); err != nil {
			exitErr(err)
		}
	default:
		exitErr(fmt.Errorf("unknown queue subcommand %q\n%s", args[0], queueUsage))
	}
}

func runQueue(c *client.Client, pageSize int) {
	for _, q := range listed(c.ListAllQueuedExpenses(pageSize, "")) {
		overdue := ""
		if q.IsOverdue {
			overdue = "OVERDUE"
		}
		fmt.Printf("%s\t%d days\t%s\t(ID: %d)%s\n", q.DocumentName, q.DaysPassed, overdue, q.Id, lastColumn(q.DocumentCode))
	}
}

func runEFactura(c *client.Client, pageSize int) {
	for _, e := range listed(c.ListAllEFactura(pageSize, "")) {
		fmt.Printf("%s\t%.2f %s\t%s\t%s\n", e.SerialCode, e.TotalAmount, e.CurrencyCode, e.InvoiceDate, e.PartyName)
	}
}

func runCompany(c *client.Client) {
	if c.CompanyID == "" {
		fmt.Fprintln(os.Stderr, "Error: could not determine company ID")
		os.Exit(1)
	}
	company, err := c.GetCompanyInfo(c.CompanyID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Name: %s\n", company.Name)
	fmt.Printf("CUI: %s\n", company.Code1)
	fmt.Printf("Reg: %s\n", company.Code2)
	fmt.Printf("Address: %s\n", company.Address)
	if m := strings.TrimSpace(company.InvoiceMentions); m != "" {
		fmt.Printf("Invoice mentions: %s\n", m)
	}

	if codes, err := c.GetCAENCodes(c.CompanyID); err == nil {
		for _, code := range codes {
			marker := ""
			if code.IsPrimary {
				marker = " (principal)"
			}
			fmt.Printf("CAEN: %s - %s%s\n", code.Code, code.Name, marker)
		}
	}
	printCompanyExtras(c)
}

func runTaxes(c *client.Client, args []string) {
	summary, err := c.GetSummaryForYear(parseYearArg(args))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	taxCfg, err := config.LoadTaxes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading taxes config: %v\n", err)
		os.Exit(1)
	}

	result := taxes.Calculate(summary.TotalRevenues, summary.TotalDeductibleExpenses, taxCfg)

	fmt.Printf("Tax Breakdown (%d)\n", summary.Year)
	fmt.Printf("══════════════════════════════════════════\n")
	fmt.Printf("Total Revenues:       %s\n", taxes.FormatRON(summary.TotalRevenues))
	fmt.Printf("Deductible Expenses:  %s\n", taxes.FormatRON(summary.TotalDeductibleExpenses))
	fmt.Printf("Net Income:           %s\n", taxes.FormatRON(result.NetIncome))
	fmt.Printf("  (%.1f salarii minime brute)\n", result.SalariesCount)
	fmt.Println()

	fmt.Printf("CAS (%.0f%%): %s\n", result.CAS.Percentage, result.CAS.Label)
	fmt.Printf("  Base: %s → Amount: %s\n", taxes.FormatRON(result.CAS.Base), taxes.FormatRON(result.CAS.Amount))
	printThresholdHint(result.CAS)
	fmt.Println()

	fmt.Printf("CASS (%.0f%%): %s\n", result.CASS.Percentage, result.CASS.Label)
	fmt.Printf("  Base: %s → Amount: %s\n", taxes.FormatRON(result.CASS.Base), taxes.FormatRON(result.CASS.Amount))
	printThresholdHint(result.CASS)
	fmt.Println()

	fmt.Printf("Income Tax (%.0f%%): %s\n", taxCfg.IncomeTaxPercent, taxes.FormatRON(result.IncomeTax))
	fmt.Printf("  Base: Net Income - CAS - CASS = %s\n", taxes.FormatRON(result.NetIncome-result.CAS.Amount-result.CASS.Amount))
	fmt.Println()

	fmt.Printf("══════════════════════════════════════════\n")
	fmt.Printf("Total Taxes:          %s\n", taxes.FormatRON(result.TotalTaxes))
	fmt.Printf("Net After Tax:        %s\n", taxes.FormatRON(result.NetAfterTax))
	fmt.Printf("Effective Tax Rate:   %.1f%%\n", result.EffectiveRate)

	printSOLOEstimate(c, summary.Year, result)
}

func printThresholdHint(t taxes.ThresholdResult) {
	if t.PrevLabel != "" {
		fmt.Printf("  %s (→ %s)\n", taxes.FormatExpensesHint(t.ExpensesToPrev), t.PrevLabel)
		return
	}
	if t.NextLabel != "" {
		fmt.Printf("  Buffer: %s (%s)\n", taxes.FormatBuffer(t.BufferToNext), t.NextLabel)
	}
}

func runUpload(c *client.Client, args []string) {
	category, kind := client.UploadExpenses, "expense"
	var files []string
	for _, a := range args {
		if a == "--revenue" || a == "-r" {
			category, kind = client.UploadRevenues, "revenue"
			continue
		}
		files = append(files, a)
	}
	if len(files) < 1 {
		fmt.Fprintln(os.Stderr, "Error: no file specified")
		fmt.Fprintln(os.Stderr, "Usage: solo-cli upload [--revenue] <file>")
		os.Exit(1)
	}

	filePath := files[0]

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", filePath)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Uploading %s...\n", filePath)

	filename, err := c.UploadDocument(filePath, category)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Uploaded: %s\n", filename)
	fmt.Printf("Document added to %s queue for processing.\n", kind)
}

// parseDocumentID reads the single numeric id of a delete subcommand
func parseDocumentID(args []string, usage string) (int, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("missing ID\nUsage: %s", usage)
	}
	id, err := strconv.Atoi(args[0])
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid ID '%s' (must be a number)", args[0])
	}
	return id, nil
}

// deleteExpenseDocument deletes a queued or rejected expense document. Like
// the revenue delete, it only accepts ids those two lists show
func deleteExpenseDocument(c *client.Client, pageSize, id int, yes bool) error {
	name, err := findExpenseDocument(c, pageSize, id)
	if err != nil {
		return err
	}
	if err := confirm(fmt.Sprintf("Delete expense document %s (ID %d)?", name, id), yes); err != nil {
		return err
	}
	if err := c.DeleteExpense(id); err != nil {
		return err
	}
	fmt.Println("Document deleted")
	return nil
}

func findExpenseDocument(c *client.Client, pageSize, id int) (string, error) {
	queue, err := c.ListAllQueuedExpenses(pageSize, "")
	if err := lookupErr(err); err != nil {
		return "", err
	}
	for _, q := range queue {
		if q.Id == id {
			return q.DocumentName, nil
		}
	}
	rejected, err := c.ListAllRejectedExpenses(pageSize)
	if err := lookupErr(err); err != nil {
		return "", err
	}
	for _, r := range rejected {
		if r.Id == id {
			return r.DocumentName, nil
		}
	}
	return "", fmt.Errorf("no queued or rejected expense document with ID %d, see solo-cli queue and solo-cli expenses rejected", id)
}
