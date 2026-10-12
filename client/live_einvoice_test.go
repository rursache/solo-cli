//go:build live

package client

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

// Read-only checks of the issued invoice e-Factura endpoints. Never calls
// manual-send, expedited-send or settings/update

func rawKeys(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("payload is not a JSON object: %v", err)
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestLiveEInvoiceSettings(t *testing.T) {
	c := liveClient(t)

	var raw json.RawMessage
	if err := c.doJSON("GET", "/proxy/accounting/e-invoice/settings", "/settings", nil, &raw); err != nil {
		t.Fatal(err)
	}
	t.Logf("settings keys: %v", rawKeys(t, raw))
	assertSchema(t, raw, reflect.TypeOf(EInvoiceSettings{}), "einvoice settings")

	s, err := c.GetEInvoiceSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.SendingDelayDays == nil || s.SendContactInfo == nil {
		t.Errorf("sending preferences missing: %+v", s)
	}
}

func TestLiveEInvoiceStatusLookups(t *testing.T) {
	c := liveClient(t)

	resp, err := c.ListRevenues(0, 200, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) == 0 {
		t.Skip("no revenues on account")
	}

	first := resp.Items[0]
	for _, code := range []string{first.UniqueCode, first.SerialCode} {
		r, err := c.FindRevenueByCode(code)
		if err != nil || r.UniqueCode != first.UniqueCode {
			t.Errorf("FindRevenueByCode did not find the first invoice: %v", err)
		}
	}

	counts := map[string]int{}
	var tracked *Revenue
	for i, r := range resp.Items {
		kind := r.EInvoiceStatus.Kind()
		if r.EInvoiceStatus != nil && kind == EInvoiceUnknown {
			t.Errorf("unknown EInvoiceStatus code %q", r.EInvoiceStatus.Code)
		}
		counts[kind]++
		if tracked == nil && r.TracksEInvoice() {
			tracked = &resp.Items[i]
		}
	}
	t.Logf("e-Factura status counts in the first %d invoices: %v", len(resp.Items), counts)
	if tracked == nil {
		t.Skip("no invoice with a tracked e-Factura")
	}

	// The lookups answer for any invoice, so all three shapes are checked on one
	code := tracked.UniqueCode
	lookups := map[string]reflect.Type{
		"queue-status":   reflect.TypeOf(EInvoiceQueueStatus{}),
		"error-reason":   reflect.TypeOf(EInvoiceErrorReason{}),
		"spv-attributes": reflect.TypeOf(EInvoiceSpvAttributes{}),
	}
	for action, typ := range lookups {
		var raw json.RawMessage
		if err := c.doJSON("GET", einvoicePath(action, code), "/revenues", nil, &raw); err != nil {
			t.Errorf("%s: %v", action, err)
			continue
		}
		t.Logf("%s keys: %v", action, rawKeys(t, raw))
		assertSchema(t, raw, typ, action)
	}

	if q, err := c.GetEInvoiceQueueStatus(code); err != nil {
		t.Error(err)
	} else {
		t.Logf("queue-status (%s invoice): Status=%s AllowExpeditedSending=%v SendOn set=%v", tracked.EInvoiceStatus.Kind(), q.Status, q.AllowExpeditedSending, q.SendOn != "")
	}
	if e, err := c.GetEInvoiceErrorReason(code); err != nil {
		t.Error(err)
	} else {
		t.Logf("error-reason (%s invoice): Reason=%s", tracked.EInvoiceStatus.Kind(), e.Reason)
	}
	if s, err := c.GetEInvoiceSpvAttributes(code); err != nil {
		t.Error(err)
	} else {
		t.Logf("spv-attributes (%s invoice): IsUploaded=%v UploadIndex set=%v", tracked.EInvoiceStatus.Kind(), s.IsUploaded, s.DisplayIndex() != "")
	}
}
