package client

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

const filedDeclarationJSON = `{
	"FiscalYear": 2025,
	"ReportingFiscalYear": 2026,
	"DeclarationStatus": "DeclarationAutoFiledOk",
	"LastCompletedWorkflow": "wfc_0000",
	"LastCompletedOn": "2026-02-01",
	"HasDueInFiscalYear": false,
	"TaxesSummary": {
		"IncomeTax": 5000,
		"HealthTax": 4000,
		"HealthOtherRevenuesTax": "500",
		"TotalHealthTaxes": 4500,
		"RetirementTax": 10000,
		"TotalTaxes": 99999,
		"IncomeTaxDiscount": 150,
		"TotalTaxesDiscounted": 0,
		"RevenueTaxes": [{"RevenueTypeId": 1, "Name": "Activitate\u001b[31m", "Sequence": 1, "Tax": 5000}],
		"Revenues": [],
		"TotalRevenues": 80000,
		"TotalOtherRevenues": 0,
		"HasCompanyRevenues": true,
		"CompanyRevenue": {"TotalRevenues": 80000, "NetRevenues": 60000, "DeductibleExpenses": 20000}
	},
	"ProfileSummary": {"IsStudent": false, "EmploymentMonths": []}
}`

const unfiledDeclarationJSON = `{
	"FiscalYear": 2026,
	"ReportingFiscalYear": 2027,
	"DeclarationStatus": "None",
	"LastCompletedWorkflow": null,
	"LastCompletedOn": null,
	"HasDueInFiscalYear": false,
	"TaxesSummary": {
		"IncomeTax": null, "HealthTax": null, "HealthOtherRevenuesTax": null, "RetirementTax": null,
		"TotalHealthTaxes": 0, "TotalTaxes": 0, "IncomeTaxDiscount": 0, "TotalTaxesDiscounted": 0,
		"RevenueTaxes": [], "Revenues": [], "TotalRevenues": 0, "TotalOtherRevenues": 0,
		"HasCompanyRevenues": false, "CompanyRevenue": null
	},
	"ProfileSummary": {"IsStudent": null, "EmploymentMonths": []}
}`

func TestGetTaxDeclarationDashboardFiled(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/proxy/accounting/tax-declaration/tax-dashboard/2025" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if !strings.HasSuffix(r.Header.Get("Referer"), "/tax-declaration/dashboard") {
			t.Errorf("referer = %q", r.Header.Get("Referer"))
		}
		w.Write([]byte(filedDeclarationJSON))
	}))

	d, err := c.GetTaxDeclarationDashboard(2025)
	if err != nil {
		t.Fatal(err)
	}
	if d.FiscalYear != 2025 || d.FilingYear() != 2026 || d.DeclarationStatus != DeclarationAutoFiledOk || d.LastCompletedOn != "2026-02-01" {
		t.Errorf("dashboard = %+v", d)
	}
	if !d.DeclarationStatus.Filed() || d.DeclarationStatus.NeedsAttention() || d.DeclarationStatus.Label() != "Filed at ANAF by SOLO" {
		t.Errorf("status helpers for %s", d.DeclarationStatus)
	}
	if !d.HasTaxes() {
		t.Fatal("HasTaxes = false")
	}
	s := d.TaxesSummary
	// The total sums the four taxes and ignores the server TotalTaxes, like the web
	if s.HealthTaxes() != 4500 || s.Total() != 19500 || s.Discount() != 150 || s.DiscountedTotal() != 19350 {
		t.Errorf("health %v total %v discount %v discounted %v", s.HealthTaxes(), s.Total(), s.Discount(), s.DiscountedTotal())
	}
	if s.CompanyRevenue == nil || *s.CompanyRevenue.NetRevenues != 60000 {
		t.Errorf("company revenue = %+v", s.CompanyRevenue)
	}
	if len(s.RevenueTaxes) != 1 || s.RevenueTaxes[0].Name != "Activitate[31m" || *s.RevenueTaxes[0].Tax != 5000 {
		t.Errorf("revenue taxes = %+v (names must be sanitized)", s.RevenueTaxes)
	}

	deadline := d.Deadline()
	if deadline.Year() != 2026 || deadline.Month() != time.May || deadline.Day() != 25 {
		t.Errorf("deadline = %v", deadline)
	}
}

func TestTaxDeclarationDashboardUnfiled(t *testing.T) {
	var d TaxDeclarationDashboard
	if err := json.Unmarshal([]byte(unfiledDeclarationJSON), &d); err != nil {
		t.Fatal(err)
	}
	if d.DeclarationStatus != DeclarationNone || d.DeclarationStatus.Label() != "Not started" || d.HasTaxes() || d.LastCompletedOn != "" {
		t.Errorf("unfiled = %+v", d)
	}
	if d.TaxesSummary == nil || d.TaxesSummary.CompanyRevenue != nil || d.TaxesSummary.Total() != 0 {
		t.Errorf("unfiled summary = %+v", d.TaxesSummary)
	}
}

func TestTaxDeclarationDashboardOddValues(t *testing.T) {
	var d TaxDeclarationDashboard
	err := json.Unmarshal([]byte(`{"FiscalYear":"2024","DeclarationStatus":"declarationautofilingerror","TaxesSummary":"broken"}`), &d)
	if err != nil {
		t.Fatal(err)
	}
	if d.FiscalYear != 2024 || d.FilingYear() != 2025 || d.DeclarationStatus != DeclarationAutoFilingError || !d.DeclarationStatus.NeedsAttention() {
		t.Errorf("odd values = %+v", d)
	}
	if d.TaxesSummary != nil || d.HasTaxes() {
		t.Errorf("a malformed summary must be dropped: %+v", d.TaxesSummary)
	}

	var unknown TaxDeclarationDashboard
	json.Unmarshal([]byte(`{"FiscalYear":2024,"DeclarationStatus":"SomethingNew"}`), &unknown)
	if unknown.DeclarationStatus != "SomethingNew" || unknown.DeclarationStatus.Known() || unknown.DeclarationStatus.Label() != "Unknown status" {
		t.Errorf("unknown status = %q", unknown.DeclarationStatus)
	}
}

func TestDeclarationStatusLabels(t *testing.T) {
	for _, s := range []DeclarationStatus{
		DeclarationNone, DeclarationFormAutoFiling, DeclarationFormManualFiling, DeclarationFormNoFiling,
		DeclarationFormPnlExpired, DeclarationAutoFilingSoon, DeclarationAutoFilingError, DeclarationAutoFiledOk,
		DeclarationManualFilingStarted, DeclarationManualFilingCompleted, DeclarationPnlExpired, DeclarationCustom,
	} {
		if !s.Known() || s.Label() == "Unknown status" {
			t.Errorf("%s has no label", s)
		}
		if normalizeDeclarationStatus(strings.ToUpper(string(s))) != s {
			t.Errorf("%s is not matched ignoring case", s)
		}
	}
}
