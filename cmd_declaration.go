package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"solo-cli/client"
	"solo-cli/taxes"
)

func runDeclaration(c *client.Client, args []string) {
	year := parseYearArg(args)
	if year == 0 {
		// The declaration filed this year covers the previous one
		year = time.Now().Year() - 1
	}
	d, err := c.GetTaxDeclarationDashboard(year)
	if err != nil {
		exitErr(err)
	}
	if d.FiscalYear == 0 {
		d.FiscalYear = year
	}
	writeDeclaration(os.Stdout, d, time.Now())
}

func writeDeclaration(w io.Writer, d *client.TaxDeclarationDashboard, now time.Time) {
	fmt.Fprintf(w, "Declaratia Unica %d (filed in %d)\n", d.FiscalYear, d.FilingYear())
	fmt.Fprintf(w, "══════════════════════════════════════════\n")
	if d.DeclarationStatus == "" {
		fmt.Fprintln(w, "Status:               Unknown status")
	} else {
		fmt.Fprintf(w, "Status:               %s (%s)\n", d.DeclarationStatus.Label(), d.DeclarationStatus)
	}
	if d.LastCompletedOn != "" {
		fmt.Fprintf(w, "Last Completed:       %s\n", d.LastCompletedOn)
	}
	fmt.Fprintln(w)

	if s := d.TaxesSummary; d.HasTaxes() {
		fmt.Fprintf(w, "CAS:                  %s\n", taxes.FormatRON(valueOrZero(s.RetirementTax)))
		fmt.Fprintf(w, "CASS:                 %s\n", taxes.FormatRON(s.HealthTaxes()))
		if other := valueOrZero(s.HealthOtherRevenuesTax); other > 0 {
			fmt.Fprintf(w, "  of which other income: %s\n", taxes.FormatRON(other))
		}
		fmt.Fprintf(w, "Income Tax:           %s\n", taxes.FormatRON(valueOrZero(s.IncomeTax)))
		if s.Discount() > 0 {
			fmt.Fprintf(w, "Income Tax Discount:  -%s\n", taxes.FormatRON(s.Discount()))
		}
		fmt.Fprintf(w, "Total Taxes:          %s\n", taxes.FormatRON(s.Total()))
		if s.Discount() > 0 {
			fmt.Fprintf(w, "Total After Discount: %s\n", taxes.FormatRON(s.DiscountedTotal()))
		}
	} else {
		fmt.Fprintln(w, "Taxes:                not computed yet")
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Deadline:             %s\n", client.DeadlineText(d.Deadline(), now))
	fmt.Fprintf(w, "File or review at %s\n", client.TaxDeclarationWebURL)
}

func valueOrZero(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
