package client

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestUpdateInvoiceMentions(t *testing.T) {
	var bodies []map[string]any
	answer := `{"Ok":true,"Data":null,"ErrorCode":null,"ErrorMessage":null}`
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/proxy/accounting/company/update-settings" || !strings.HasSuffix(r.Header.Get("Referer"), "/settings") {
			t.Errorf("unexpected request: %s %s referer %s", r.Method, r.URL.Path, r.Header.Get("Referer"))
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		bodies = append(bodies, body)
		w.Write([]byte(answer))
	}))

	text := "Capital social: 200 RON"
	if err := c.UpdateInvoiceMentions("abc", &text); err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateInvoiceMentions("abc", nil); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 || len(bodies[0]) != 2 || bodies[0]["CompanyCode"] != "company_abc" || bodies[0]["InvoiceMentions"] != text {
		t.Fatalf("bodies = %v", bodies)
	}
	if v, ok := bodies[1]["InvoiceMentions"]; !ok || v != nil {
		t.Errorf("clearing must send InvoiceMentions null, got %v", bodies[1])
	}

	answer = `{"Ok":false,"Data":null,"ErrorCode":"CompanyNotFound","ErrorMessage":null}`
	if err := c.UpdateInvoiceMentions("abc", &text); err == nil || !strings.Contains(err.Error(), "could not identify the company") {
		t.Errorf("CompanyNotFound = %v", err)
	}
	answer = `{"Ok":false,"Data":null,"ErrorCode":"Other","ErrorMessage":null}`
	if err := c.UpdateInvoiceMentions("abc", &text); err == nil || !strings.Contains(err.Error(), "Other") {
		t.Errorf("other error = %v", err)
	}
	if err := c.UpdateInvoiceMentions("", &text); err == nil {
		t.Error("missing company ID must fail")
	}
}
