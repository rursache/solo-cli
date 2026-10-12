package client

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestGetTaxEstimate(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/proxy/accounting/tax-declaration/tax-estimate/2026" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if !strings.HasSuffix(r.Header.Get("Referer"), "/dashboard") {
			t.Errorf("referer = %q", r.Header.Get("Referer"))
		}
		w.Write([]byte(`{"FiscalYear":2026,"TotalRevenues":100000,"HasDeductibleExpenses":true,
			"TotalDeductibleExpenses":"20000.50","HasTaxableRevenue":true,"TaxableRevenue":79999.5,
			"IncomeTax":"5000,25","RetirementTax":12000,"HealthTax":8000,"TotalTaxes":25000.25}`))
	}))

	e, err := c.GetTaxEstimate(2026)
	if err != nil {
		t.Fatal(err)
	}
	if e.FiscalYear != 2026 || !e.HasDeductibleExpenses || !e.HasTaxableRevenue {
		t.Errorf("estimate = %+v", e)
	}
	for name, got := range map[string]*float64{
		"TotalDeductibleExpenses": e.TotalDeductibleExpenses,
		"IncomeTax":               e.IncomeTax,
		"RetirementTax":           e.RetirementTax,
		"HealthTax":               e.HealthTax,
	} {
		if got == nil {
			t.Errorf("%s is nil", name)
		}
	}
	if *e.TotalDeductibleExpenses != 20000.50 || *e.IncomeTax != 5000.25 {
		t.Errorf("numeric strings not decoded: %v %v", *e.TotalDeductibleExpenses, *e.IncomeTax)
	}
	if !e.HasFigures() || e.Total() != 25000.25 {
		t.Errorf("HasFigures %v, Total %v", e.HasFigures(), e.Total())
	}
}

func TestTaxEstimateLenientDecode(t *testing.T) {
	var e TaxEstimate
	err := json.Unmarshal([]byte(`{"FiscalYear":"2025","IncomeTax":null,"RetirementTax":"abc",
		"HealthTax":1000,"TotalTaxes":null,"HasTaxableRevenue":"true","TaxableRevenue":{}}`), &e)
	if err != nil {
		t.Fatal(err)
	}
	if e.FiscalYear != 2025 || !e.HasTaxableRevenue || e.IncomeTax != nil || e.RetirementTax != nil || e.TaxableRevenue != nil {
		t.Errorf("lenient decode = %+v", e)
	}
	// Without TotalTaxes the total is the sum of the three taxes
	if !e.HasFigures() || e.Total() != 1000 {
		t.Errorf("HasFigures %v, Total %v", e.HasFigures(), e.Total())
	}

	var empty TaxEstimate
	if err := json.Unmarshal([]byte(`{"FiscalYear":2025,"TotalRevenues":0}`), &empty); err != nil {
		t.Fatal(err)
	}
	if empty.HasFigures() || empty.Total() != 0 {
		t.Errorf("empty estimate: %+v", empty)
	}
}

func TestGetTaxEstimateError(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	if _, err := c.GetTaxEstimate(2026); err == nil || !strings.Contains(err.Error(), "tax estimate") {
		t.Errorf("err = %v", err)
	}
}
