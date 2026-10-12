package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"solo-cli/client"
)

func TestCLIInvoiceShow(t *testing.T) {
	api, _ := newRevenueMock(t)
	e := newEnv(t, "good-password")

	out, errOut, code := e.run(t, api, "invoice", invNew)
	if code != 0 {
		t.Fatalf("invoice exit %d: %s", code, errOut)
	}
	for _, want := range []string{
		"Invoice: TST-a\n", "Code: " + invNew + "\n", "Client: Client Exemplu SRL\n", "Issued: 2026-02-02\n",
		"Due: 2026-02-16\n", "Status: Emisă\n", "Payment: UNPAID\n", "e-Factura: Trimisă\n", "Total: 1500.00 EUR\n",
		"Local total: 7500.00 RON (rate 5.0000)\n", "\nDezvoltare software\t1.5 ore\t1000.00\t0%\t1500.00 EUR\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("invoice output missing %q:\n%s", want, out)
		}
	}

	_, errOut, code = e.run(t, api, "invoice", "../etc")
	if code != 1 || !strings.Contains(errOut, "invalid invoice code") {
		t.Errorf("bad code: %d %q", code, errOut)
	}
	_, errOut, code = e.run(t, api, "invoice")
	if code != 1 || !strings.Contains(errOut, "Usage:") {
		t.Errorf("no args: %d %q", code, errOut)
	}
}

func TestCLIInvoicePaidUnpaid(t *testing.T) {
	api, m := newRevenueMock(t)
	e := newEnv(t, "good-password")

	_, errOut, code := e.run(t, api, "invoice", "paid", invNew, "2026-01-30", "-y")
	if code != 1 || !strings.Contains(errOut, "before the invoice issue date") {
		t.Errorf("before issue date: %d %q", code, errOut)
	}
	future := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	_, errOut, code = e.run(t, api, "invoice", "paid", invNew, future, "-y")
	if code != 1 || !strings.Contains(errOut, "in the future") {
		t.Errorf("future date: %d %q", code, errOut)
	}
	_, errOut, code = e.run(t, api, "invoice", "paid", invNew, "2026-02-05")
	if code != 1 || !strings.Contains(errOut, "--yes") {
		t.Errorf("paid without --yes: %d %q", code, errOut)
	}
	if n := m.count("POST /proxy/accounting/invoices/mark-paid"); n != 0 {
		t.Fatalf("%d mark-paid calls before confirmation", n)
	}

	out, errOut, code := e.run(t, api, "invoice", "paid", invNew, "2026-02-05", "--yes")
	if code != 0 || out != "Invoice TST-a marked paid on 2026-02-05\n" {
		t.Errorf("paid: %d %q %q", code, out, errOut)
	}
	if got := m.last("POST /proxy/accounting/invoices/mark-paid/" + invNew); !strings.HasSuffix(got, ` "2026-02-05T00:00:00+02:00"`) {
		t.Errorf("mark-paid request = %q", got)
	}

	_, errOut, code = e.run(t, api, "invoice", "paid", invNew, "2026-02-03", "--yes")
	if code != 1 || !strings.Contains(errOut, "Data încasării trebuie să fie ulterioară datei emiterii") {
		t.Errorf("server InvalidPaymentDate: %d %q", code, errOut)
	}

	_, errOut, code = e.run(t, api, "invoice", "unpaid", invNew, "--yes")
	if code != 1 || !strings.Contains(errOut, "not marked paid") {
		t.Errorf("unpaid on unpaid invoice: %d %q", code, errOut)
	}
	out, errOut, code = e.run(t, api, "invoice", "unpaid", invOld, "-y")
	if code != 0 || out != "Invoice TST-b marked unpaid\n" || m.count("POST /proxy/accounting/invoices/mark-unpaid/"+invOld) != 1 {
		t.Errorf("unpaid: %d %q %q", code, out, errOut)
	}
}

func TestCLIInvoiceDownload(t *testing.T) {
	api, _ := newRevenueMock(t)
	e := newEnv(t, "good-password")
	dir := t.TempDir()

	out, errOut, code := e.run(t, api, "invoice", "download", invNew, "-o", dir)
	want := filepath.Join(dir, "Factura-TST-a.pdf")
	if code != 0 || out != want+"\n" {
		t.Fatalf("download: %d %q %q", code, out, errOut)
	}
	if data, _ := os.ReadFile(want); string(data) != "%PDF-1.7 mock" {
		t.Errorf("file content %q", data)
	}
	_, errOut, code = e.run(t, api, "invoice", "download", invNew, "-o", dir)
	if code != 1 || !strings.Contains(errOut, "--force") {
		t.Errorf("overwrite without --force: %d %q", code, errOut)
	}
	if _, _, code = e.run(t, api, "invoice", "dl", invNew, "-o", dir, "-f"); code != 0 {
		t.Errorf("--force download exit %d", code)
	}

	_, errOut, code = e.run(t, api, "invoice", "download", invNew, "--xml", "-o", dir)
	if code != 1 || !strings.Contains(errOut, "no e-Factura XML") {
		t.Errorf("xml not offered: %d %q", code, errOut)
	}
	_, errOut, code = e.run(t, api, "invoice", "download", invNew, "--xml", "--original")
	if code != 1 || !strings.Contains(errOut, "only one of") {
		t.Errorf("both formats: %d %q", code, errOut)
	}

	// An uploaded invoice has no SOLO PDF, the default falls back to the original
	out, errOut, code = e.run(t, api, "invoice", "download", invExt, "-o", filepath.Join(dir, "orig.png"))
	if code != 0 || out != filepath.Join(dir, "orig.png")+"\n" || !strings.Contains(errOut, "original file") {
		t.Errorf("external download: %d %q %q", code, out, errOut)
	}
}

func TestCLIInvoiceDelete(t *testing.T) {
	api, m := newRevenueMock(t)
	e := newEnv(t, "good-password")

	_, errOut, code := e.run(t, api, "invoice", "delete", invOld, "--yes")
	if code != 1 || !strings.Contains(errOut, "only the last issued invoice can be deleted") || !strings.Contains(errOut, "Voidable") {
		t.Errorf("old invoice: %d %q", code, errOut)
	}
	_, errOut, code = e.run(t, api, "invoice", "delete", invNew)
	if code != 1 || !strings.Contains(errOut, "--yes") {
		t.Errorf("delete without --yes: %d %q", code, errOut)
	}
	if n := m.count("POST /proxy/accounting/invoices/cancel"); n != 0 {
		t.Fatalf("%d cancels before confirmation", n)
	}
	out, errOut, code := e.run(t, api, "invoice", "rm", invNew, "-y")
	if code != 0 || out != "Invoice TST-a deleted\n" || m.count("POST /proxy/accounting/invoices/cancel/"+invNew) != 1 {
		t.Errorf("delete: %d %q %q", code, out, errOut)
	}
}

func TestParsePaymentDay(t *testing.T) {
	now := time.Date(2026, 5, 20, 15, 0, 0, 0, time.Local)
	if d, err := parsePaymentDay(nil, now); err != nil || d.Format("2006-01-02") != "2026-05-20" {
		t.Errorf("default: %v %v", d, err)
	}
	if d, err := parsePaymentDay([]string{"2026-05-20"}, now); err != nil || d.Day() != 20 {
		t.Errorf("today: %v %v", d, err)
	}
	for _, bad := range []string{"2026-05-21", "20-05-2026", "2026-02-30", "yesterday"} {
		if _, err := parsePaymentDay([]string{bad}, now); err == nil {
			t.Errorf("%q must fail", bad)
		}
	}
}

func TestDownloadFormat(t *testing.T) {
	f, rest, err := downloadFormat([]string{"inv_1", "--original"})
	if err != nil || f != client.InvoiceFormatOriginal || len(rest) != 1 {
		t.Errorf("--original: %q %v %v", f, rest, err)
	}
	if f, _, _ := downloadFormat([]string{"inv_1"}); f != client.InvoiceFormatPDF {
		t.Errorf("default = %q", f)
	}
	if _, _, err := downloadFormat([]string{"inv_1", "--pdf"}); err == nil {
		t.Error("unknown flag must fail")
	}
}
