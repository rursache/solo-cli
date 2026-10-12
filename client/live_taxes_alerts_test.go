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

// Read-only live checks for the tax estimate endpoint. It sends its nullable keys
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
