//go:build live

package client

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Read-only checks of the invoice endpoints. Never calls mark-paid,
// mark-unpaid or cancel

func liveRevenues(t *testing.T, c *Client) []Revenue {
	t.Helper()
	resp, err := c.ListRevenues(0, 100, "")
	if err != nil {
		t.Fatalf("ListRevenues: %v", err)
	}
	if len(resp.Items) == 0 {
		t.Skip("no revenues on account")
	}
	return resp.Items
}

func TestLiveInvoiceDetail(t *testing.T) {
	c := liveClient(t)
	revenues := liveRevenues(t, c)

	var target *Revenue
	for i := range revenues {
		if !revenues[i].IsExternalDocument {
			target = &revenues[i]
			break
		}
	}
	if target == nil {
		t.Skip("no invoice issued in SOLO")
	}

	var raw json.RawMessage
	if err := c.doJSON("GET", "/proxy/accounting/invoices/load/"+target.UniqueCode, "/revenues", nil, &raw); err != nil {
		t.Fatal(err)
	}
	assertSchema(t, raw, reflect.TypeOf(InvoiceDetail{}), "invoice")
	var lines struct{ LineItems []json.RawMessage }
	json.Unmarshal(raw, &lines)
	if len(lines.LineItems) == 0 {
		t.Fatal("invoice without line items")
	}
	assertSchema(t, lines.LineItems[0], reflect.TypeOf(InvoiceDetailLine{}), "invoice.LineItems")

	inv, err := c.GetInvoice(target.UniqueCode)
	if err != nil {
		t.Fatalf("GetInvoice: %v", err)
	}
	if inv.UniqueCode != target.UniqueCode || inv.SerialCode != target.SerialCode || inv.IssueDate == "" {
		t.Error("invoice detail does not match the list row")
	}
	sum := 0.0
	for _, l := range inv.LineItems {
		sum += l.Total
	}
	if diff := sum - inv.Total; diff > 0.01 || diff < -0.01 {
		t.Error("line totals do not add up to the invoice total")
	}
}

func TestLiveInvoiceDownloadsAndPreCancel(t *testing.T) {
	c := liveClient(t)
	revenues := liveRevenues(t, c)

	// The list is newest first and only the newest invoice is deletable
	pre, err := c.PreCancelInvoice(revenues[0].UniqueCode)
	if err != nil {
		t.Fatalf("PreCancelInvoice: %v", err)
	}
	if pre.Status == "" || pre.InvoiceIssueYear == nil {
		t.Errorf("pre-cancel answer incomplete: status %q", pre.Status)
	}
	t.Logf("newest invoice pre-cancel status %s", pre.Status)
	if len(revenues) > 1 {
		if older, err := c.PreCancelInvoice(revenues[len(revenues)-1].UniqueCode); err != nil || older.Deletable() {
			t.Errorf("older invoice pre-cancel: %v, deletable %v", err, older != nil && older.Deletable())
		}
	}

	pdfDone, xmlDone := false, false
	for _, r := range revenues {
		if pdfDone && xmlDone {
			break
		}
		var raw json.RawMessage
		if err := c.doJSON("GET", "/proxy/accounting/invoices/download-options/"+r.UniqueCode, "/revenues", nil, &raw); err != nil {
			t.Fatalf("download-options: %v", err)
		}
		assertSchema(t, raw, reflect.TypeOf(InvoiceDownloadOptions{}), "download-options")
		opts, err := c.GetInvoiceDownloadOptions(r.UniqueCode)
		if err != nil {
			t.Fatalf("GetInvoiceDownloadOptions: %v", err)
		}
		if !pdfDone && opts.Has(InvoiceFormatPDF) {
			f, err := c.DownloadInvoicePDF(r.UniqueCode)
			if err != nil || f.Extension() != "pdf" {
				t.Errorf("PDF download: %v", err)
			}
			pdfDone = true
		}
		if !xmlDone && opts.Has(InvoiceFormatXML) {
			f, err := c.DownloadInvoiceXML(r.UniqueCode)
			if err != nil || f.Extension() != "xml" {
				t.Errorf("XML download: %v", err)
			}
			xmlDone = true
		}
		if opts.Has(InvoiceFormatOriginal) && opts.DocumentCode != nil {
			if _, err := c.DownloadInvoiceOriginal(*opts.DocumentCode); err != nil {
				t.Errorf("original download: %v", err)
			}
		}
	}
	t.Logf("downloaded pdf %v, xml %v", pdfDone, xmlDone)
}
