package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestEInvoiceStatusKind(t *testing.T) {
	cases := map[string]string{"Queued": EInvoiceQueued, "not-eligible": EInvoiceNotEligible, " SENT ": EInvoiceSent, "draft": EInvoiceUnknown}
	for code, want := range cases {
		s := &EInvoiceStatus{Code: code, EnableCloudFormation: true}
		if got := s.Kind(); got != want {
			t.Errorf("Kind(%q) = %q, want %q", code, got, want)
		}
	}
	var none *EInvoiceStatus
	if none.Kind() != EInvoiceUnknown || none.Tracked() {
		t.Error("nil status must be unknown and untracked")
	}
	r := Revenue{EInvoiceStatus: &EInvoiceStatus{Code: "skipped", EnableCloudFormation: false}}
	if r.TracksEInvoice() {
		t.Error("EnableCloudFormation=false must not be tracked")
	}
	r.EInvoiceStatus.EnableCloudFormation = true
	if !r.TracksEInvoice() {
		t.Error("skipped with EnableCloudFormation must be tracked")
	}
	r.IsExternalDocument = true
	if r.TracksEInvoice() {
		t.Error("external documents must not be tracked")
	}
}

func TestFindRevenueByCode(t *testing.T) {
	var searches []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			SearchText string
			StartIndex int
			MaxResults int
		}
		json.NewDecoder(r.Body).Decode(&body)
		searches = append(searches, fmt.Sprintf("%s@%d", body.SearchText, body.StartIndex))
		switch {
		case strings.EqualFold(body.SearchText, "INV-002"):
			fmt.Fprint(w, `{"Items":[{"UniqueCode":"code-2","SerialCode":"INV-002"}],"TotalResults":1}`)
		case body.SearchText != "":
			fmt.Fprint(w, `{"Items":[],"TotalResults":0}`)
		case body.StartIndex == 0:
			items := make([]string, 200)
			for i := range items {
				items[i] = fmt.Sprintf(`{"UniqueCode":"code-%d","SerialCode":"INV-%d"}`, i+100, i+100)
			}
			fmt.Fprintf(w, `{"Items":[%s],"TotalResults":201}`, strings.Join(items, ","))
		default:
			fmt.Fprint(w, `{"Items":[{"UniqueCode":"code-last","SerialCode":"INV-LAST"}],"TotalResults":201}`)
		}
	}))

	r, err := c.FindRevenueByCode("inv-002")
	if err != nil || r.UniqueCode != "code-2" {
		t.Fatalf("search match: %v, %+v", err, r)
	}

	searches = nil
	r, err = c.FindRevenueByCode("code-last")
	if err != nil || r.SerialCode != "INV-LAST" {
		t.Fatalf("paging match: %v, %+v", err, r)
	}
	if strings.Join(searches, ",") != "code-last@0,@0,@200" {
		t.Errorf("requests = %v", searches)
	}

	if _, err := c.FindRevenueByCode("missing"); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Errorf("not found = %v", err)
	}
	if _, err := c.FindRevenueByCode(" "); err == nil {
		t.Error("empty code must fail")
	}
}

func TestEInvoiceLookups(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Referer") != baseURL+"/revenues" {
			t.Errorf("%s %s referer %q", r.Method, r.URL.Path, r.Header.Get("Referer"))
		}
		switch r.URL.EscapedPath() {
		case "/proxy/accounting/e-invoice/queue-status/abc%2F1":
			fmt.Fprint(w, `{"Status":"InvoiceSendingScheduled","SendOn":"2026-10-14T00:00:00","AllowExpeditedSending":true}`)
		case "/proxy/accounting/e-invoice/error-reason/abc%2F1":
			fmt.Fprint(w, `{"Reason":"CustomerLiquidated"}`)
		case "/proxy/accounting/e-invoice/spv-attributes/abc%2F1":
			fmt.Fprint(w, `{"IsUploaded":true,"UploadIndex":5001234567,"DownloadIndex":null}`)
		default:
			t.Errorf("unexpected path %s", r.URL.EscapedPath())
		}
	}))

	q, err := c.GetEInvoiceQueueStatus("abc/1")
	if err != nil || !q.CanExpedite() || q.SendOn != "2026-10-14T00:00:00" {
		t.Fatalf("queue status: %v, %+v", err, q)
	}
	if !strings.Contains(q.Message(), "2026-10-14") {
		t.Errorf("scheduled message = %q", q.Message())
	}

	e, err := c.GetEInvoiceErrorReason("abc/1")
	if err != nil || !strings.Contains(e.Message(), "deregistered") {
		t.Fatalf("error reason: %v, %+v", err, e)
	}

	s, err := c.GetEInvoiceSpvAttributes("abc/1")
	if err != nil || s.DisplayIndex() != "5001234567" || s.DownloadIndex != "" {
		t.Fatalf("spv attributes: %v, %+v", err, s)
	}
	s.IsUploaded = false
	if s.DisplayIndex() != "" {
		t.Error("index must stay hidden until uploaded")
	}
}

func TestEInvoiceQueueMessages(t *testing.T) {
	for _, status := range []string{QueueInformationMissing, InvoiceSendingStarted, NoSpvAccess, "Other"} {
		q := EInvoiceQueueStatus{Status: status, AllowExpeditedSending: true}
		if q.Message() == "" || q.CanExpedite() {
			t.Errorf("%s: message %q, can expedite %v", status, q.Message(), q.CanExpedite())
		}
	}
	if (&EInvoiceQueueStatus{Status: InvoiceSendingScheduled}).CanExpedite() {
		t.Error("expedite needs AllowExpeditedSending")
	}
}

func TestSendEInvoice(t *testing.T) {
	status := "Queued"
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.URL.Path != "/proxy/accounting/e-invoice/manual-send/abc" || string(body) != "{}" {
			t.Errorf("%s %s body %s", r.Method, r.URL.Path, body)
		}
		fmt.Fprintf(w, `{"Status":%q}`, status)
	}))

	if err := c.SendEInvoice("abc"); err != nil {
		t.Fatalf("queued: %v", err)
	}
	for s, want := range map[string]string{
		"InvoiceNotFound":     "deleted",
		"InvalidStatus":       "cannot be sent manually",
		"InvoiceLinesMissing": "not enough invoice data",
		"ObsoleteInvoice":     "before 2024",
		"SpvAccessUnvailable": "not connected to the ANAF SPV",
		"Weird":               "(Weird)",
	} {
		status = s
		err := c.SendEInvoice("abc")
		var sendErr *EInvoiceSendError
		if !errors.As(err, &sendErr) || sendErr.Status != s || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v", s, err)
		}
	}
}

func TestExpediteEInvoice(t *testing.T) {
	answer := `{"Ok":true}`
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/proxy/accounting/e-invoice/expedited-send/abc" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		fmt.Fprint(w, answer)
	}))
	if err := c.ExpediteEInvoice("abc"); err != nil {
		t.Fatalf("ok: %v", err)
	}
	answer = `{"Ok":false}`
	if err := c.ExpediteEInvoice("abc"); err == nil || !strings.Contains(err.Error(), "send date could not be changed") {
		t.Errorf("Ok=false = %v", err)
	}
	answer = `{"Ok":false,"ErrorMessage":"Nu se poate"}`
	if err := c.ExpediteEInvoice("abc"); err == nil || !strings.Contains(err.Error(), "Nu se poate") {
		t.Errorf("Ok=false with message = %v", err)
	}
}

func TestEInvoiceSettings(t *testing.T) {
	settings := `{"IsConnected":false,"HasPersonalAccess":false,"HasDelegatedAccess":true,"DelegatedAccessStatus":"Confirmed","PersonalAccessStatus":null,"SendingDelayDays":"2","SendContactInfo":true,"IsTransitioning":false}`
	var posted string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") != baseURL+"/settings" {
			t.Errorf("referer %q", r.Header.Get("Referer"))
		}
		switch r.URL.Path {
		case "/proxy/accounting/e-invoice/settings":
			fmt.Fprint(w, settings)
		case "/proxy/accounting/e-invoice/settings/update":
			body, _ := io.ReadAll(r.Body)
			posted = string(body)
			fmt.Fprint(w, `{"Ok":true}`)
		}
	}))

	s, err := c.GetEInvoiceSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.IsConnected || !s.HasDelegatedAccess || s.DelegatedAccessStatus != "Confirmed" || s.PersonalAccessStatus != "" {
		t.Errorf("settings: %+v", s)
	}
	if s.SendingDelayDays == nil || *s.SendingDelayDays != 2 || s.SendContactInfo == nil || !*s.SendContactInfo {
		t.Errorf("preferences: %+v", s)
	}

	settings = `{"IsConnected":true}`
	if s, err = c.GetEInvoiceSettings(); err != nil || s.SendingDelayDays != nil || s.SendContactInfo != nil {
		t.Errorf("missing preferences must stay nil: %v, %+v", err, s)
	}

	if err := c.UpdateEInvoiceSettings(0, false); err != nil {
		t.Fatal(err)
	}
	if posted != `{"SendingDelayDays":0,"SendContactInfo":false}` {
		t.Errorf("update body = %s", posted)
	}
}

func TestEInvoiceSettingsUpdateRejected(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Ok":false,"ErrorCode":"ReadOnly"}`)
	}))
	err := c.UpdateEInvoiceSettings(2, true)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "ReadOnly" {
		t.Errorf("rejected update = %v", err)
	}
}

func TestEInvoiceLabels(t *testing.T) {
	if SendingDelayLabel(0) != "as soon as possible after issuing" || SendingDelayLabel(2) != "within 2 days of issuing" {
		t.Error("delay labels")
	}
	if !strings.Contains(ContactInfoLabel(true), "e-mail and phone") || !strings.Contains(ContactInfoLabel(false), "not included") {
		t.Error("contact info labels")
	}
	for _, k := range []string{EInvoiceNotEligible, EInvoiceQueued, EInvoiceSending, EInvoiceSent, EInvoiceError, EInvoiceSkipped, EInvoiceUnknown} {
		if EInvoiceStatusMeaning(k) == "" {
			t.Errorf("no meaning for %s", k)
		}
	}
}
