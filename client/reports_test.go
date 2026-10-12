package client

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDownloadCashflowReport(t *testing.T) {
	const id = "0123456789abcdef0123456789abcdef"
	year := time.Now().Year()
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := fmt.Sprintf("/proxy/accounting/reports/cashflow-report/company_%s/%d", id, year-1)
		if r.URL.Path != want {
			t.Errorf("path = %s, want %s", r.URL.Path, want)
		}
		if !strings.HasSuffix(r.Header.Get("Referer"), "/documents/reports") {
			t.Errorf("referer = %s", r.Header.Get("Referer"))
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("%PDF-1.7 register"))
	}))
	c.CompanyStartYear = year - 2

	f, err := c.DownloadCashflowReport(id, year-1)
	if err != nil || f.Extension() != "pdf" {
		t.Fatalf("DownloadCashflowReport: %v, %+v", err, f)
	}
	for _, bad := range []int{year - 3, year + 1} {
		if _, err := c.DownloadCashflowReport(id, bad); err == nil || !strings.Contains(err.Error(), "available years") {
			t.Errorf("year %d must be rejected before any request, got %v", bad, err)
		}
	}
	if _, err := c.DownloadCashflowReport("", year); err == nil {
		t.Error("missing company ID must fail")
	}
}

func TestCashflowYears(t *testing.T) {
	now := time.Now().Year()
	c := &Client{}
	if first, last := c.CashflowYears(); first != 1990 || last != now {
		t.Errorf("unknown start = %d..%d", first, last)
	}
	c.CompanyStartYear = 2021
	if first, _ := c.CashflowYears(); first != 2021 {
		t.Errorf("start 2021 = %d", first)
	}
	c.CompanyStartYear = now + 5
	if first, _ := c.CashflowYears(); first != 1990 {
		t.Errorf("future start must be ignored, got %d", first)
	}
}
