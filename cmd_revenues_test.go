package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// revenueMock layers the revenue and invoice endpoints over the e2e mock API
type revenueMock struct {
	mu   sync.Mutex
	hits []string
}

func (m *revenueMock) record(r *http.Request, body string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hits = append(m.hits, r.Method+" "+r.URL.Path+" "+body)
}

func (m *revenueMock) count(prefix string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, h := range m.hits {
		if strings.HasPrefix(h, prefix) {
			n++
		}
	}
	return n
}

func (m *revenueMock) last(prefix string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.hits) - 1; i >= 0; i-- {
		if strings.HasPrefix(m.hits[i], prefix) {
			return m.hits[i]
		}
	}
	return ""
}

const (
	invNew = "inv_0000000000000000000000000000000a"
	invOld = "inv_0000000000000000000000000000000b"
	invExt = "inv_0000000000000000000000000000000c"
)

func newRevenueMock(t *testing.T) (*mockAPI, *revenueMock) {
	t.Helper()
	api := newMockAPI(t)
	m := &revenueMock{}
	base := api.server.Config.Handler

	mux := http.NewServeMux()
	mux.Handle("/", base)
	mux.HandleFunc("/proxy/accounting/revenues/queued", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Items":[{"Id":21,"DocumentCode":"doc_q","DocumentName":"factura-client.pdf","DaysPassed":1,"IsOverdue":false}],"TotalResults":null}`)
	})
	mux.HandleFunc("/proxy/accounting/revenues/rejected", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Items":[{"Id":22,"DocumentCode":"doc_r","DocumentName":"poza.jpg","Reason":"Document ilizibil"}],"TotalResults":null}`)
	})
	mux.HandleFunc("/proxy/accounting/revenues/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			base.ServeHTTP(w, r)
			return
		}
		m.record(r, "")
		fmt.Fprint(w, `{}`)
	})
	mux.HandleFunc("/api/financial-documents/save/revenues/", func(w http.ResponseWriter, r *http.Request) {
		m.record(r, "")
		fmt.Fprint(w, `1`)
	})
	mux.HandleFunc("/proxy/accounting/invoices/load/", func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimPrefix(r.URL.Path, "/proxy/accounting/invoices/load/")
		paid, payDate := "false", ""
		if code == invOld {
			paid, payDate = "true", "2026-02-10T00:00:00+02:00"
		}
		if code != invNew && code != invOld {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `{"UniqueCode":%q,"SerialCode":"TST-%s","ClientName":"Client Exemplu SRL",
			"IssueDate":"2026-02-02T00:00:00+02:00","DueDate":"2026-02-16T00:00:00+02:00","PaymentDate":%q,"IsPaid":%s,
			"Amount":1500,"VAT":0,"Total":1500,"Currency":{"ShortName":"EUR"},
			"InvoiceLocalAmount":{"Amount":7500,"VAT":0,"Total":7500,"Currency":{"ShortName":"RON"},"ExchangeRate":{"RateValue":5}},
			"Status":{"Code":"issued","Name":"Emisă","IsCancelled":false},"EInvoiceStatus":{"Code":"sent","Name":"Trimisă"},
			"LineItems":[{"ItemName":"Dezvoltare software","Quantity":1.5,"UnitPrice":1000,"MeasureUnit":{"LabelRo":"ore"},"Amount":1500,"VatPercentage":0,"VAT":0,"Total":1500}]}`,
			code, code[len(code)-1:], payDate, paid)
	})
	mux.HandleFunc("/proxy/accounting/invoices/download-options/", func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimPrefix(r.URL.Path, "/proxy/accounting/invoices/download-options/")
		switch code {
		case invExt:
			fmt.Fprintf(w, `{"InvoiceCode":%q,"IsExternalDocument":true,"DocumentCode":"doc_ext","AvailableFormats":["ExternalFile"]}`, code)
		default:
			fmt.Fprintf(w, `{"InvoiceCode":%q,"IsExternalDocument":false,"DocumentCode":null,"AvailableFormats":["InternalPdf"]}`, code)
		}
	})
	mux.HandleFunc("/proxy/accounting/invoices/download/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		fmt.Fprint(w, "%PDF-1.7 mock")
	})
	mux.HandleFunc("/api/financial-documents/download/doc_ext", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		fmt.Fprint(w, "\x89PNG mock")
	})
	mux.HandleFunc("/proxy/accounting/invoices/mark-paid/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		m.record(r, string(body))
		if strings.Contains(string(body), "2026-02-03") {
			fmt.Fprint(w, `{"Ok":false,"Data":null,"ErrorCode":"InvalidPaymentDate"}`)
			return
		}
		fmt.Fprint(w, `{"Ok":true,"Data":{"IsPaid":true,"PaymentDate":"2026-02-05T00:00:00"}}`)
	})
	mux.HandleFunc("/proxy/accounting/invoices/mark-unpaid/", func(w http.ResponseWriter, r *http.Request) {
		m.record(r, "")
		fmt.Fprint(w, `{"Ok":true}`)
	})
	mux.HandleFunc("/proxy/accounting/invoices/pre-cancel/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, invNew) {
			fmt.Fprint(w, `{"Status":"Deletable","InvoiceIssueYear":2026}`)
			return
		}
		fmt.Fprint(w, `{"Status":"Voidable","InvoiceIssueYear":2026}`)
	})
	mux.HandleFunc("/proxy/accounting/invoices/cancel/", func(w http.ResponseWriter, r *http.Request) {
		m.record(r, "")
		fmt.Fprint(w, `{"Status":"Deleted"}`)
	})
	api.server.Config.Handler = mux
	return api, m
}

func TestCLIRevenueDocuments(t *testing.T) {
	api, m := newRevenueMock(t)
	e := newEnv(t, "good-password")

	out, errOut, code := e.run(t, api, "revenues")
	if code != 0 {
		t.Fatalf("revenues exit %d: %s", code, errOut)
	}
	if !strings.HasPrefix(out, "INV-001\t1000.50 RON\tPAID\tACME Corp") {
		t.Errorf("revenues output: %q", out)
	}
	if !strings.Contains(errOut, "1 rejected revenue document(s)") || !strings.Contains(errOut, "poza.jpg - Document ilizibil") {
		t.Errorf("stderr missing rejected warning: %q", errOut)
	}

	out, _, code = e.run(t, api, "revenues", "queue")
	if code != 0 || out != "factura-client.pdf\t1 days\t\t(ID: 21)\n" {
		t.Errorf("revenues queue: %d %q", code, out)
	}
	out, _, code = e.run(t, api, "rev", "rejected")
	if code != 0 || out != "poza.jpg\tDocument ilizibil\t(ID: 22)\n" {
		t.Errorf("revenues rejected: %d %q", code, out)
	}

	_, errOut, code = e.run(t, api, "revenues", "delete", "99", "--yes")
	if code != 1 || !strings.Contains(errOut, "no queued or rejected revenue document with ID 99") {
		t.Errorf("unknown id: %d %q", code, errOut)
	}
	_, errOut, code = e.run(t, api, "revenues", "delete", "21")
	if code != 1 || !strings.Contains(errOut, "--yes") {
		t.Errorf("delete without --yes: %d %q", code, errOut)
	}
	if n := m.count("DELETE"); n != 0 {
		t.Fatalf("%d deletes before confirmation", n)
	}
	out, errOut, code = e.run(t, api, "revenues", "rm", "-y", "22")
	if code != 0 || out != "Document deleted\n" {
		t.Errorf("delete: %d %q %q", code, out, errOut)
	}
	if m.count("DELETE /proxy/accounting/revenues/22") != 1 {
		t.Errorf("hits: %v", m.hits)
	}

	_, errOut, code = e.run(t, api, "revenues", "bogus")
	if code != 1 || !strings.Contains(errOut, "unknown revenues subcommand") {
		t.Errorf("bogus subcommand: %d %q", code, errOut)
	}
}
