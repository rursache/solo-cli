package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"solo-cli/client"
)

func TestWriteDeclarationFiled(t *testing.T) {
	var d client.TaxDeclarationDashboard
	err := json.Unmarshal([]byte(`{"FiscalYear":2025,"ReportingFiscalYear":2026,"DeclarationStatus":"DeclarationAutoFiledOk",
		"LastCompletedOn":"2026-02-01","TaxesSummary":{"IncomeTax":5000,"HealthTax":4000,"HealthOtherRevenuesTax":500,
		"RetirementTax":10000,"TotalTaxes":1,"IncomeTaxDiscount":150}}`), &d)
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	writeDeclaration(&out, &d, time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC))
	want := `Declaratia Unica 2025 (filed in 2026)
══════════════════════════════════════════
Status:               Filed at ANAF by SOLO (DeclarationAutoFiledOk)
Last Completed:       2026-02-01

CAS:                  10000.00 RON
CASS:                 4500.00 RON
  of which other income: 500.00 RON
Income Tax:           5000.00 RON
Income Tax Discount:  -150.00 RON
Total Taxes:          19500.00 RON
Total After Discount: 19350.00 RON

Deadline:             25 May 2026 (in 85 days)
File or review at https://falcon.solo.ro/tax-declaration/dashboard
`
	if out.String() != want {
		t.Errorf("output:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestWriteDeclarationUnfiled(t *testing.T) {
	var d client.TaxDeclarationDashboard
	err := json.Unmarshal([]byte(`{"FiscalYear":2026,"ReportingFiscalYear":2027,"DeclarationStatus":"None",
		"LastCompletedOn":null,"TaxesSummary":{"IncomeTax":null,"HealthTax":null,"HealthOtherRevenuesTax":null,
		"RetirementTax":null,"TotalTaxes":0,"IncomeTaxDiscount":0,"CompanyRevenue":null}}`), &d)
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	writeDeclaration(&out, &d, time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC))
	got := out.String()
	for _, want := range []string{
		"Status:               Not started (None)\n\nTaxes:                not computed yet\n",
		"Deadline:             25 May 2027 (passed)\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Last Completed") || strings.Contains(got, "CAS:") {
		t.Errorf("unfiled year must not show completion or tax lines:\n%s", got)
	}
}

func TestDeadlineText(t *testing.T) {
	loc := time.FixedZone("EEST", 3*3600)
	deadline := time.Date(2026, time.May, 25, 0, 0, 0, 0, loc)
	for _, tc := range []struct {
		now  time.Time
		want string
	}{
		{time.Date(2026, 5, 25, 20, 0, 0, 0, loc), "25 May 2026 (today)"},
		// 22:30 UTC on the 24th is already the 25th in Bucharest
		{time.Date(2026, 5, 24, 22, 30, 0, 0, time.UTC), "25 May 2026 (today)"},
		{time.Date(2026, 5, 24, 9, 0, 0, 0, loc), "25 May 2026 (tomorrow)"},
		{time.Date(2026, 5, 26, 0, 0, 1, 0, loc), "25 May 2026 (passed)"},
		{time.Date(2026, 1, 1, 0, 0, 0, 0, loc), "25 May 2026 (in 144 days)"},
	} {
		if got := client.DeadlineText(deadline, tc.now); got != tc.want {
			t.Errorf("DeadlineText(%v) = %q, want %q", tc.now, got, tc.want)
		}
	}
}
