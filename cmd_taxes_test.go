package main

import (
	"strings"
	"testing"

	"solo-cli/client"
	"solo-cli/config"
	"solo-cli/taxes"
)

func ptr(v float64) *float64 { return &v }

func TestWriteSOLOEstimateMatches(t *testing.T) {
	local := taxes.Calculate(100000, 20000, config.DefaultTaxConfig())
	est := &client.TaxEstimate{
		FiscalYear:     2026,
		TaxableRevenue: ptr(local.NetIncome),
		RetirementTax:  ptr(local.CAS.Amount + 0.4),
		HealthTax:      ptr(local.CASS.Amount),
		IncomeTax:      ptr(local.IncomeTax),
	}
	var out strings.Builder
	writeSOLOEstimate(&out, 2026, est, local)
	got := out.String()
	for _, want := range []string{
		"\nSOLO Estimate (2026)\n",
		"CAS:                  " + taxes.FormatRON(local.CAS.Amount+0.4) + "\n",
		"Total Taxes:          " + taxes.FormatRON(est.Total()) + "\n",
		"Matches the local breakdown\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "diff") {
		t.Errorf("rounding noise must not count as a difference:\n%s", got)
	}
}

func TestWriteSOLOEstimateDiffers(t *testing.T) {
	local := taxes.Calculate(100000, 20000, config.DefaultTaxConfig())
	est := &client.TaxEstimate{
		RetirementTax: ptr(local.CAS.Amount + 500),
		HealthTax:     ptr(local.CASS.Amount),
		IncomeTax:     ptr(local.IncomeTax),
		TotalTaxes:    ptr(local.TotalTaxes + 500),
	}
	var out strings.Builder
	writeSOLOEstimate(&out, 2026, est, local)
	got := out.String()
	for _, want := range []string{
		"Taxable Income:       n/a\n",
		"(local " + taxes.FormatRON(local.CAS.Amount) + ", diff +500.00)",
		"Differs from the local breakdown",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestWriteSOLOEstimateEmpty(t *testing.T) {
	var out strings.Builder
	writeSOLOEstimate(&out, 2027, &client.TaxEstimate{FiscalYear: 2027}, taxes.Calculate(0, 0, config.DefaultTaxConfig()))
	if !strings.HasSuffix(out.String(), "SOLO has no estimate for this year\n") {
		t.Errorf("empty estimate:\n%s", out.String())
	}
}
