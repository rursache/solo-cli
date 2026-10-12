package client

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const testInvoiceCode = "inv_0123456789abcdef0123456789abcdef"

func TestGetInvoice(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/proxy/accounting/invoices/load/"+testInvoiceCode {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if !strings.HasSuffix(r.Header.Get("Referer"), "/invoices/"+testInvoiceCode+"/view") {
			t.Errorf("referer = %q", r.Header.Get("Referer"))
		}
		w.Write([]byte(`{"UniqueCode":"` + testInvoiceCode + `","SerialCode":"TST-0001","ClientName":"Client SRL",
			"IssueDate":"2026-03-02T00:00:00+02:00","DueDate":"2026-03-16T00:00:00+02:00","PaymentDate":"",
			"IsPaid":false,"Amount":1000,"VAT":0,"Total":1000,"Currency":{"ShortName":"RON"},
			"Status":{"Code":"issued","Name":"Emisă","IsCancelled":false},
			"LineItems":[{"ItemName":"Servicii","Quantity":2,"UnitPrice":500,"MeasureUnit":{"Code":"buc","LabelRo":"buc","LabelEn":"pcs"},
			"Amount":1000,"VatPercentage":0,"VAT":0,"Total":1000}]}`))
	}))

	inv, err := c.GetInvoice(testInvoiceCode)
	if err != nil {
		t.Fatalf("GetInvoice: %v", err)
	}
	if inv.SerialCode != "TST-0001" || inv.Total != 1000 || inv.Status == nil || inv.Status.Code != "issued" || len(inv.LineItems) != 1 {
		t.Fatalf("invoice = %+v", inv)
	}
	line := inv.LineItems[0]
	if line.ItemName != "Servicii" || line.Quantity != 2 || line.UnitPrice != 500 || line.MeasureUnit == nil || line.MeasureUnit.LabelEn != "pcs" {
		t.Errorf("line = %+v", line)
	}
}

func TestInvoiceCodeValidation(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request expected, got %s %s", r.Method, r.URL.Path)
	}))
	for _, code := range []string{"", ".", "..", "../x", "a/b", "a?b", "a b", "a%2Fb"} {
		if _, err := c.GetInvoice(code); err == nil {
			t.Errorf("GetInvoice(%q) must fail", code)
		}
		if err := c.CancelInvoice(code); err == nil {
			t.Errorf("CancelInvoice(%q) must fail", code)
		}
		if _, err := c.DownloadInvoiceOriginal(code); err == nil {
			t.Errorf("DownloadInvoiceOriginal(%q) must fail", code)
		}
	}
}

func TestInvoiceDownloads(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.Header.Get("Referer"), "/revenues") {
			t.Errorf("%s referer = %q", r.URL.Path, r.Header.Get("Referer"))
		}
		switch r.URL.Path {
		case "/proxy/accounting/invoices/download-options/" + testInvoiceCode:
			w.Write([]byte(`{"InvoiceCode":"` + testInvoiceCode + `","IsExternalDocument":false,"DocumentCode":null,"AvailableFormats":["InternalPdf","InternalXml","Future"]}`))
		case "/proxy/accounting/invoices/download/" + testInvoiceCode:
			w.Header().Set("Content-Type", "application/pdf")
			w.Write([]byte("%PDF-1.7"))
		case "/proxy/accounting/e-invoice/download/" + testInvoiceCode:
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte("<?xml version=\"1.0\"?><Invoice/>"))
		case "/api/financial-documents/download/doc_x":
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write([]byte("\xFF\xD8\xFF\xE0"))
		default:
			http.NotFound(w, r)
		}
	}))

	opts, err := c.GetInvoiceDownloadOptions(testInvoiceCode)
	if err != nil {
		t.Fatalf("GetInvoiceDownloadOptions: %v", err)
	}
	if !opts.Has(InvoiceFormatPDF) || !opts.Has(InvoiceFormatXML) || opts.Has(InvoiceFormatOriginal) || opts.DocumentCode != nil {
		t.Errorf("options = %+v", opts)
	}

	if f, err := c.DownloadInvoicePDF(testInvoiceCode); err != nil || f.Extension() != "pdf" {
		t.Errorf("pdf: %v, %+v", err, f)
	}
	if f, err := c.DownloadInvoiceXML(testInvoiceCode); err != nil || f.Extension() != "xml" {
		t.Errorf("xml: %v, %+v", err, f)
	}
	if f, err := c.DownloadInvoiceOriginal("doc_x"); err != nil || f.Extension() != "jpg" {
		t.Errorf("original: %v, %+v", err, f)
	}
	if _, err := c.DownloadInvoiceXML("inv_missing"); err == nil {
		t.Error("missing XML must fail")
	}
}

func TestMarkInvoicePaid(t *testing.T) {
	var gotBody string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		if r.Method != "POST" || !strings.HasSuffix(r.Header.Get("Referer"), "/revenues") {
			t.Errorf("request %s, referer %q", r.Method, r.Header.Get("Referer"))
		}
		switch r.URL.Path {
		case "/proxy/accounting/invoices/mark-paid/inv_ok":
			w.Write([]byte(`{"Ok":true,"Data":{"IsPaid":true,"PaymentDate":"2026-07-15T00:00:00"},"ErrorCode":null}`))
		case "/proxy/accounting/invoices/mark-paid/inv_null":
			w.Write([]byte(`{"Ok":true,"Data":null}`))
		case "/proxy/accounting/invoices/mark-paid/inv_early":
			w.Write([]byte(`{"Ok":false,"Data":null,"ErrorCode":"InvalidPaymentDate"}`))
		case "/proxy/accounting/invoices/mark-paid/inv_other":
			w.Write([]byte(`{"Ok":false,"ErrorCode":"Locked"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))

	// A summer date gets the EEST offset, whatever zone the caller passes
	day := time.Date(2026, 7, 15, 23, 30, 0, 0, time.FixedZone("far", -10*3600))
	state, err := c.MarkInvoicePaid("inv_ok", day)
	if err != nil || !state.IsPaid || state.PaymentDate != "2026-07-15T00:00:00" {
		t.Fatalf("MarkInvoicePaid: %v, %+v", err, state)
	}
	if gotBody != `"2026-07-15T00:00:00+03:00"` {
		t.Errorf("body = %s, want a bare JSON date string", gotBody)
	}

	winter := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	state, err = c.MarkInvoicePaid("inv_null", winter)
	if err != nil || !state.IsPaid || state.PaymentDate != "2026-01-05T00:00:00+02:00" {
		t.Errorf("null Data: %v, %+v", err, state)
	}

	if _, err := c.MarkInvoicePaid("inv_early", winter); !errors.Is(err, ErrInvalidPaymentDate) {
		t.Errorf("InvalidPaymentDate = %v", err)
	}
	if _, err := c.MarkInvoicePaid("inv_other", winter); err == nil || !strings.Contains(err.Error(), "Modificările nu au putut fi efectuate") || !strings.Contains(err.Error(), "Locked") {
		t.Errorf("unknown code = %v", err)
	}
}

func TestMarkInvoiceUnpaid(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != "POST" || string(body) != "{}" {
			t.Errorf("request %s body %s", r.Method, body)
		}
		switch r.URL.Path {
		case "/proxy/accounting/invoices/mark-unpaid/inv_ok":
			w.Write([]byte(`{"Ok":true}`))
		case "/proxy/accounting/invoices/mark-unpaid/inv_fail":
			w.Write([]byte(`{"Ok":false}`))
		case "/proxy/accounting/invoices/mark-unpaid/inv_msg":
			w.Write([]byte(`{"Ok":false,"ErrorCode":"X","ErrorMessage":"Factura este anulată"}`))
		}
	}))

	if err := c.MarkInvoiceUnpaid("inv_ok"); err != nil {
		t.Errorf("MarkInvoiceUnpaid: %v", err)
	}
	if err := c.MarkInvoiceUnpaid("inv_fail"); err == nil || !strings.Contains(err.Error(), "could not be saved") {
		t.Errorf("Ok=false = %v", err)
	}
	if err := c.MarkInvoiceUnpaid("inv_msg"); err == nil || !strings.Contains(err.Error(), "Factura este anulată") {
		t.Errorf("server message = %v", err)
	}
}

func TestInvoiceCancel(t *testing.T) {
	cancels := 0
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/proxy/accounting/invoices/pre-cancel/inv_new":
			w.Write([]byte(`{"Status":"Deletable","InvoiceIssueYear":2026}`))
		case "/proxy/accounting/invoices/pre-cancel/inv_old":
			w.Write([]byte(`{"Status":"Voidable","InvoiceIssueYear":2025}`))
		case "/proxy/accounting/invoices/cancel/inv_new":
			body, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || string(body) != "{}" {
				t.Errorf("cancel %s body %s", r.Method, body)
			}
			cancels++
			w.Write([]byte(`{"Status":"Deleted"}`))
		case "/proxy/accounting/invoices/cancel/inv_voided":
			w.Write([]byte(`{"Status":"Voided"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))

	pre, err := c.PreCancelInvoice("inv_new")
	if err != nil || !pre.Deletable() || pre.InvoiceIssueYear == nil || *pre.InvoiceIssueYear != 2026 {
		t.Errorf("pre-cancel new: %v, %+v", err, pre)
	}
	if pre, err := c.PreCancelInvoice("inv_old"); err != nil || pre.Deletable() {
		t.Errorf("pre-cancel old: %v, %+v", err, pre)
	}
	if err := c.CancelInvoice("inv_new"); err != nil || cancels != 1 {
		t.Errorf("CancelInvoice: %v", err)
	}
	if err := c.CancelInvoice("inv_voided"); err == nil || !strings.Contains(err.Error(), "Voided") {
		t.Errorf("non Deleted status = %v", err)
	}
}

func TestInvoiceAPIDateDST(t *testing.T) {
	for _, tc := range []struct {
		day  time.Time
		want string
	}{
		{time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC), "2026-03-29T00:00:00+02:00"},
		{time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC), "2026-03-30T00:00:00+03:00"},
		{time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC), "2026-10-25T00:00:00+03:00"},
		{time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC), "2026-10-26T00:00:00+02:00"},
	} {
		if got, err := InvoiceAPIDate(tc.day); err != nil || got != tc.want {
			t.Errorf("InvoiceAPIDate(%s) = %q, %v, want %q", tc.day.Format("2006-01-02"), got, err, tc.want)
		}
	}
}
