//go:build live

package client

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// Schema drift detection: Go's JSON decoding leaves silent zero values when
// the API renames or removes a field, so a wrong number would never error.
// These tests fetch the raw payloads and verify every field our structs map
// still exists, pinpointing drift the moment SOLO.ro changes their API

// assertSchema verifies every JSON key the struct expects exists in the live
// payload. Non-pointer fields are required, pointer fields are optional (the
// API legitimately omits or nulls them). Recurses into nested structs
func assertSchema(t *testing.T, raw json.RawMessage, typ reflect.Type, path string) {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Errorf("%s: payload is not a JSON object: %v", path, err)
		return
	}

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		tag := strings.Split(f.Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}

		val, present := obj[tag]
		if !present {
			if f.Type.Kind() != reflect.Ptr {
				t.Errorf("%s: required field %q missing from live payload (schema drift)", path, tag)
			}
			continue
		}

		ft := f.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct && string(val) != "null" {
			assertSchema(t, val, ft, path+"."+tag)
		}
	}
}

// rawItems decodes a list payload into raw per-item JSON
type rawItems struct {
	Items []json.RawMessage `json:"Items"`
}

// fetchRawList POSTs a list endpoint and returns the raw items
func fetchRawList(t *testing.T, c *Client, path, referer string) []json.RawMessage {
	t.Helper()
	var raw json.RawMessage
	if err := c.doJSON("POST", path, referer, newListRequest(0, 3), &raw); err != nil {
		t.Fatalf("fetching %s: %v", path, err)
	}
	var list rawItems
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatalf("%s: no Items array in payload: %v", path, err)
	}
	return list.Items
}

func TestLiveSchemaSummary(t *testing.T) {
	c := liveClient(t)

	var raw json.RawMessage
	if err := c.doJSON("GET", "/proxy/accounting/dashboard/summary", "/dashboard", nil, &raw); err != nil {
		t.Fatal(err)
	}
	assertSchema(t, raw, reflect.TypeOf(Summary{}), "summary")
}

func TestLiveSchemaRevenues(t *testing.T) {
	c := liveClient(t)

	items := fetchRawList(t, c, "/proxy/accounting/revenues/list", "/revenues")
	if len(items) == 0 {
		t.Skip("no revenues on account, cannot check item schema")
	}
	assertSchema(t, items[0], reflect.TypeOf(Revenue{}), "revenue")
}

func TestLiveSchemaExpenses(t *testing.T) {
	c := liveClient(t)

	items := fetchRawList(t, c, "/proxy/accounting/expenses/list", "/expenses")
	if len(items) == 0 {
		t.Skip("no expenses on account, cannot check item schema")
	}
	assertSchema(t, items[0], reflect.TypeOf(Expense{}), "expense")
}

func TestLiveSchemaQueueAndRejected(t *testing.T) {
	c := liveClient(t)

	queued := fetchRawList(t, c, "/proxy/accounting/expenses/queued", "/expenses")
	if len(queued) > 0 {
		assertSchema(t, queued[0], reflect.TypeOf(QueuedExpense{}), "queued")
	} else {
		t.Log("queue empty, item schema not checked")
	}

	rejected := fetchRawList(t, c, "/proxy/accounting/expenses/rejected", "/expenses")
	if len(rejected) > 0 {
		assertSchema(t, rejected[0], reflect.TypeOf(RejectedExpense{}), "rejected")
	} else {
		t.Log("no rejected expenses, item schema not checked")
	}
}

func TestLiveSchemaEFactura(t *testing.T) {
	c := liveClient(t)

	items := fetchRawList(t, c, "/proxy/accounting/e-invoice/list-expenses", "/e-factura")
	if len(items) == 0 {
		t.Skip("no e-factura documents, cannot check item schema")
	}
	assertSchema(t, items[0], reflect.TypeOf(EFactura{}), "efactura")
}

func TestLiveSchemaCompany(t *testing.T) {
	c := liveClient(t)

	id, err := c.DiscoverCompanyID()
	if err != nil {
		t.Fatalf("DiscoverCompanyID: %v", err)
	}

	var raw json.RawMessage
	path := "/proxy/accounting/company/basic-profile/company_" + id
	if err := c.doJSON("GET", path, "/settings", nil, &raw); err != nil {
		t.Fatal(err)
	}
	assertSchema(t, raw, reflect.TypeOf(CompanyInfoResponse{}), "companyResponse")

	var resp struct {
		Data json.RawMessage `json:"Data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil || string(resp.Data) == "" || string(resp.Data) == "null" {
		t.Fatalf("company response has no Data object: %v", err)
	}
	assertSchema(t, resp.Data, reflect.TypeOf(CompanyInfo{}), "company")
}

// Consistency: the same numbers reached through different endpoints must
// agree, otherwise the dashboard and the tax calculator would silently
// diverge from the per-section summaries

func TestLiveYearParameterRespected(t *testing.T) {
	c := liveClient(t)

	current, err := c.GetSummary()
	if err != nil {
		t.Fatal(err)
	}
	lastYear := current.Year - 1

	s, err := c.GetSummaryForYear(lastYear)
	if err != nil {
		t.Fatalf("GetSummaryForYear(%d): %v", lastYear, err)
	}
	if s.Year != lastYear {
		t.Errorf("requested year %d, API returned %d", lastYear, s.Year)
	}
}

func TestLiveCountsSchemaAndConsistency(t *testing.T) {
	c := liveClient(t)

	summary, err := c.GetSummary()
	if err != nil {
		t.Fatal(err)
	}

	// Schema of the two counts endpoints
	var raw json.RawMessage
	if err := c.doJSON("GET", "/proxy/accounting/revenues/summary", "/", nil, &raw); err != nil {
		t.Fatal(err)
	}
	assertSchema(t, raw, reflect.TypeOf(RevenueCounts{}), "revenueCounts")

	if err := c.doJSON("GET", "/proxy/accounting/expenses/summary", "/", nil, &raw); err != nil {
		t.Fatal(err)
	}
	assertSchema(t, raw, reflect.TypeOf(ExpenseCounts{}), "expenseCounts")

	// Consistency: the queued documents count must agree with the queue
	// list, which both the TUI Queue tab and the dashboard banner rely on.
	// Note: the dashboard's ExpensesAwaitingReview is NOT the queue size
	// (verified live: 3 queued docs with the field at 0), so it is not
	// compared here
	expCounts, err := c.GetExpenseCounts(0)
	if err != nil {
		t.Fatalf("GetExpenseCounts: %v", err)
	}
	queue, err := c.ListQueuedExpenses(0, 100)
	if err != nil {
		t.Fatalf("ListQueuedExpenses: %v", err)
	}
	if expCounts.QueuedExpenses != len(queue.Items) {
		t.Errorf("queued count %d disagrees with queue list length %d", expCounts.QueuedExpenses, len(queue.Items))
	}

	if _, err := c.GetRevenueCounts(0); err != nil {
		t.Fatalf("GetRevenueCounts: %v", err)
	}
	t.Logf("dashboard awaiting review: revenues %d, expenses %d", summary.RevenuesAwaitingReview, summary.ExpensesAwaitingReview)
}
