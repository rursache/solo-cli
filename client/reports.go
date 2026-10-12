package client

import (
	"fmt"
	"time"
)

// CashflowYears is the range the cash flow register exists for, from
// CompanyStartYear (1990 when unknown) to the current year
func (c *Client) CashflowYears() (first, last int) {
	last = time.Now().Year()
	first = c.CompanyStartYear
	if first < 1990 || first > last {
		first = 1990
	}
	return first, last
}

// DownloadCashflowReport fetches the yearly Registrul de incasari si plati PDF
func (c *Client) DownloadCashflowReport(companyID string, year int) (*DownloadedFile, error) {
	if companyID == "" {
		return nil, fmt.Errorf("company ID not available")
	}
	if first, last := c.CashflowYears(); year < first || year > last {
		return nil, fmt.Errorf("no cash flow register for %d, available years are %d to %d", year, first, last)
	}
	path := fmt.Sprintf("/proxy/accounting/reports/cashflow-report/company_%s/%d", companyID, year)
	return c.downloadNamed(path, "/documents/reports", "cash flow register")
}
