package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"solo-cli/client"
	"solo-cli/taxes"
)

// estimateTolerance ignores rounding noise when comparing SOLO with the local calculator
const estimateTolerance = 1.0

// printSOLOEstimate appends SOLO's own estimate to the taxes output. A
// failing call only warns, the local breakdown above is still valid
func printSOLOEstimate(c *client.Client, year int, local *taxes.TaxBreakdown) {
	if year == 0 {
		year = time.Now().Year()
	}
	est, err := c.GetTaxEstimate(year)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: SOLO estimate unavailable: %v\n", err)
		return
	}
	writeSOLOEstimate(os.Stdout, year, est, local)
}

// writeSOLOEstimate prints SOLO's figures and marks the lines that differ
// from the local calculator
func writeSOLOEstimate(w io.Writer, year int, est *client.TaxEstimate, local *taxes.TaxBreakdown) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "SOLO Estimate (%d)\n", year)
	fmt.Fprintf(w, "══════════════════════════════════════════\n")
	if !est.HasFigures() {
		fmt.Fprintln(w, "SOLO has no estimate for this year")
		return
	}

	differs := false
	row := func(label string, solo *float64, local float64) {
		if solo == nil {
			fmt.Fprintf(w, "%-22sn/a\n", label)
			return
		}
		fmt.Fprintf(w, "%-22s%s", label, taxes.FormatRON(*solo))
		if diff := *solo - local; math.Abs(diff) >= estimateTolerance {
			differs = true
			fmt.Fprintf(w, "  (local %s, diff %+.2f)", taxes.FormatRON(local), diff)
		}
		fmt.Fprintln(w)
	}
	row("Taxable Income:", est.TaxableRevenue, local.NetIncome)
	row("CAS:", est.RetirementTax, local.CAS.Amount)
	row("CASS:", est.HealthTax, local.CASS.Amount)
	row("Income Tax:", est.IncomeTax, local.IncomeTax)
	total := est.Total()
	row("Total Taxes:", &total, local.TotalTaxes)

	if differs {
		fmt.Fprintln(w, "Differs from the local breakdown, check taxes.json and the revenues and expenses SOLO counts")
	} else {
		fmt.Fprintln(w, "Matches the local breakdown")
	}
}
