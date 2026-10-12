//go:build live

package client

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Read-only live checks for the tax estimate, Declaratia Unica dashboard
// and dashboard alert endpoints. These endpoints send their nullable keys
// with null rather than omitting them, so every mapped key is required

// requireAllKeys checks that every json-tagged field of typ is a key of raw
// and returns the decoded object for nested checks
func requireAllKeys(t *testing.T, raw json.RawMessage, typ reflect.Type, path string) map[string]json.RawMessage {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("%s: payload is not a JSON object: %v", path, err)
	}
	for i := 0; i < typ.NumField(); i++ {
		tag := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		if _, ok := obj[tag]; !ok {
			t.Errorf("%s: field %q missing from live payload (schema drift)", path, tag)
		}
	}
	return obj
}

func fetchRaw(t *testing.T, c *Client, path, referer string) json.RawMessage {
	t.Helper()
	body, err := c.doRaw("GET", path, referer, nil)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return body
}

func TestLiveTaxEstimate(t *testing.T) {
	c := liveClient(t)
	year := time.Now().Year()

	path := fmt.Sprintf("/proxy/accounting/tax-declaration/tax-estimate/%d", year)
	requireAllKeys(t, fetchRaw(t, c, path, "/dashboard"), reflect.TypeOf(TaxEstimate{}), "TaxEstimate")

	e, err := c.GetTaxEstimate(year)
	if err != nil {
		t.Fatal(err)
	}
	if e.FiscalYear != year {
		t.Errorf("FiscalYear = %d, want %d", e.FiscalYear, year)
	}
	if !e.HasFigures() {
		t.Error("estimate has no tax figures")
	}
	if e.TotalTaxes != nil && e.Total() < 0 {
		t.Errorf("negative total")
	}
}

func TestLiveTaxDeclarationDashboard(t *testing.T) {
	c := liveClient(t)
	// Last year is usually filed, the current one never is
	for _, year := range []int{time.Now().Year() - 1, time.Now().Year()} {
		path := fmt.Sprintf("/proxy/accounting/tax-declaration/tax-dashboard/%d", year)
		obj := requireAllKeys(t, fetchRaw(t, c, path, "/tax-declaration/dashboard"), reflect.TypeOf(TaxDeclarationDashboard{}), "TaxDeclarationDashboard")
		if raw := obj["TaxesSummary"]; string(raw) != "null" {
			summary := requireAllKeys(t, raw, reflect.TypeOf(DeclarationTaxesSummary{}), "TaxesSummary")
			if cr := summary["CompanyRevenue"]; len(cr) > 0 && string(cr) != "null" {
				requireAllKeys(t, cr, reflect.TypeOf(DeclarationCompanyRevenue{}), "TaxesSummary.CompanyRevenue")
			}
			var items []json.RawMessage
			json.Unmarshal(summary["RevenueTaxes"], &items)
			for _, item := range items {
				requireAllKeys(t, item, reflect.TypeOf(DeclarationRevenueTax{}), "TaxesSummary.RevenueTaxes[]")
			}
		}

		d, err := c.GetTaxDeclarationDashboard(year)
		if err != nil {
			t.Fatal(err)
		}
		if d.FiscalYear != year || d.FilingYear() != year+1 {
			t.Errorf("%d: years %d/%d", year, d.FiscalYear, d.FilingYear())
		}
		if !d.DeclarationStatus.Known() {
			t.Errorf("%d: undocumented DeclarationStatus %q", year, d.DeclarationStatus)
		}
		if d.DeclarationStatus == DeclarationNone && d.HasTaxes() {
			t.Errorf("%d: unfiled year with taxes", year)
		}
		t.Logf("%d: %s", year, d.DeclarationStatus)
	}
}

func TestLiveAlerts(t *testing.T) {
	c := liveClient(t)

	requireAllKeys(t, fetchRaw(t, c, "/proxy/accounting/vat/alert-status", "/dashboard"), reflect.TypeOf(VATAlertStatus{}), "VATAlertStatus")
	requireAllKeys(t, fetchRaw(t, c, "/proxy/accounting/caen-update/alert-status", "/dashboard"), reflect.TypeOf(CAENAlertStatus{}), "CAENAlertStatus")
	requireAllKeys(t, fetchRaw(t, c, "/proxy/accounting/document-workflows/notarial-power-of-attorney/status", "/dashboard"), reflect.TypeOf(PowerOfAttorneyStatus{}), "PowerOfAttorneyStatus")

	a := c.GetAlerts()
	for _, e := range a.Errors {
		t.Errorf("alert source failed: %v", e)
	}
	if a.VAT == nil || a.VAT.Status == "" || a.CAEN == nil || a.CAEN.Status == "" || a.PowerOfAttorney == nil || a.PowerOfAttorney.Status == "" {
		t.Fatalf("alert statuses missing: %+v", a)
	}
	t.Logf("vat %s, caen %s, procura %s, %d active", a.VAT.Status, a.CAEN.Status, a.PowerOfAttorney.Status, len(a.Active()))
}
