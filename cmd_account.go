package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"solo-cli/client"
)

var isoDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// dateOnly keeps the YYYY-MM-DD prefix of a server timestamp
func dateOnly(s string) string {
	if isoDateRe.MatchString(s) {
		return s[:10]
	}
	return s
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// lastColumn appends a document code as a new last tab-separated column,
// left out when the server has none so the existing columns stay as they were
func lastColumn(code string) string {
	if code == "" {
		return ""
	}
	return "\t" + code
}

func docCodeSuffix(code string) string {
	if code == "" {
		return ""
	}
	return " (document " + code + ")"
}

const documentsUsage = "Usage: solo-cli documents [pfa | download <code> [-o file] [-f]]"

func runDocuments(args []string) {
	flags, err := parseCmdFlags(args)
	if err != nil {
		exitErr(err)
	}
	sub := ""
	if len(flags.Args) > 0 {
		sub = flags.Args[0]
	}
	switch sub {
	case "", "list", "declarations":
		if len(flags.Args) > 1 {
			exitErr(fmt.Errorf("unexpected argument %q\n%s", flags.Args[1], documentsUsage))
		}
		c, _ := setupClient()
		listTaxDeclarations(c)
	case "pfa":
		c, _ := setupClient()
		listRegistrationDocuments(c)
	case "download", "dl":
		if len(flags.Args) != 2 {
			exitErr(fmt.Errorf("missing document code\n%s", documentsUsage))
		}
		c, _ := setupClient()
		downloadDocument(c, flags.Args[1], flags)
	default:
		exitErr(fmt.Errorf("unknown documents subcommand %q\n%s", sub, documentsUsage))
	}
}

func listTaxDeclarations(c *client.Client) {
	docs, err := c.ListTaxDeclarations()
	if err != nil {
		exitErr(err)
	}
	if len(docs) == 0 {
		fmt.Fprintln(os.Stderr, "No filed declarations")
		return
	}
	for _, d := range docs {
		fmt.Printf("%s\t%s\t%s\t%s\n", d.TaxDeclarationType, d.ReportingPeriod, dateOnly(d.FiledOn), d.UniqueCode)
	}
}

func listRegistrationDocuments(c *client.Client) {
	sections, err := c.ListRegistrationDocuments()
	if err != nil {
		exitErr(err)
	}
	found := false
	for _, s := range sections {
		for _, d := range s.Documents {
			found = true
			fmt.Printf("%s\t%s\t%s\t%s\n", s.Title(), d.UniqueCode, d.OriginalFileName, dateOnly(d.ModifiedOn))
		}
	}
	if !found {
		fmt.Fprintln(os.Stderr, "No PFA documents")
	}
}

// documentBaseName finds a file name for a document code in the declaration
// and PFA lists, falling back to the code itself
func documentBaseName(c *client.Client, code string) string {
	name := ""
	if docs, err := c.ListTaxDeclarations(); err == nil {
		for _, d := range docs {
			if d.UniqueCode == code {
				name = d.OriginalFileName
				if name == "" {
					name = d.TaxDeclarationType + " " + d.ReportingPeriod
				}
				break
			}
		}
	}
	if name == "" {
		if sections, err := c.ListRegistrationDocuments(); err == nil {
		search:
			for _, s := range sections {
				for _, d := range s.Documents {
					if d.UniqueCode == code {
						name = d.OriginalFileName
						break search
					}
				}
			}
		}
	}
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if strings.TrimSpace(name) == "" {
		return code
	}
	return name
}

func downloadDocument(c *client.Client, code string, flags cmdFlags) {
	fmt.Fprintf(os.Stderr, "Downloading document %s...\n", code)
	f, err := c.DownloadDocument(code)
	if err != nil {
		exitErr(err)
	}
	if err := saveDownload(f, documentBaseName(c, code), flags); err != nil {
		exitErr(err)
	}
}

// runExpensesCmd keeps "expenses" as the list and adds "expenses download"
func runExpensesCmd(args []string) {
	if len(args) == 0 || (args[0] != "download" && args[0] != "dl") {
		withClient(runExpenses)
		return
	}
	flags, err := parseCmdFlags(args[1:])
	if err != nil {
		exitErr(err)
	}
	if len(flags.Args) != 1 {
		exitErr(fmt.Errorf("missing document code\nUsage: solo-cli expenses download <DocumentCode> [-o file] [-f]"))
	}
	code := flags.Args[0]
	c, _ := setupClient()
	fmt.Fprintf(os.Stderr, "Downloading expense document %s...\n", code)
	f, err := c.DownloadFinancialDocument(code)
	if err != nil {
		exitErr(err)
	}
	if err := saveDownload(f, code, flags); err != nil {
		exitErr(err)
	}
}

func runReport(args []string) {
	flags, err := parseCmdFlags(args)
	if err != nil {
		exitErr(err)
	}
	year := time.Now().Year()
	switch len(flags.Args) {
	case 0:
	case 1:
		year, err = strconv.Atoi(flags.Args[0])
		if err != nil {
			exitErr(fmt.Errorf("invalid year %q\nUsage: solo-cli report [year] [-o file] [-f]", flags.Args[0]))
		}
	default:
		exitErr(fmt.Errorf("too many arguments\nUsage: solo-cli report [year] [-o file] [-f]"))
	}

	c, _ := setupClient()
	if c.CompanyID == "" {
		exitErr(fmt.Errorf("could not determine company ID"))
	}
	if first, last := c.CashflowYears(); year < first || year > last {
		exitErr(fmt.Errorf("no cash flow register for %d, available years are %d to %d", year, first, last))
	}
	fmt.Fprintf(os.Stderr, "Downloading cash flow register for %d...\n", year)
	f, err := c.DownloadCashflowReport(c.CompanyID, year)
	if err != nil {
		exitErr(err)
	}
	if err := saveDownload(f, fmt.Sprintf("Registru-incasari-plati-%d", year), flags); err != nil {
		exitErr(err)
	}
}
